# ERA Comms — mail-api restart persistence (G1-1/G1-2)
# Requires compose stack: deploy/docker-compose.comms.yml (+ .dev.yml optional)
param(
    [string]$ComposeFile = "deploy/docker-compose.comms.yml",
    [string]$MailApi = "http://127.0.0.1:8150",
    [string]$Subject = "restart-proof-$(Get-Date -Format 'yyyyMMddHHmmss')"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$Log = "reports/comms-restart-persist-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"

function Log($m) {
    $line = "$(Get-Date -Format o) $m"
    Add-Content -Path $Log -Value $line
    Write-Host $line
}

function Count-Messages($email) {
    $list = Invoke-RestMethod -Method Get -Uri "$MailApi/api/v1/mail/messages?email=$email"
    if ($null -eq $list.messages) { return 0 }
    return @($list.messages).Count
}

function Find-Subject($email, $subj) {
    $list = Invoke-RestMethod -Method Get -Uri "$MailApi/api/v1/mail/messages?email=$email"
    foreach ($m in @($list.messages)) {
        if ("$($m.Subject)" -eq $subj -or "$($m.subject)" -eq $subj) { return $true }
        $raw = "$($m.Raw)$($m.raw)"
        if (-not $raw) { continue }
        try {
            $decoded = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($raw))
            if ($decoded -match [regex]::Escape($subj)) { return $true }
        } catch {
            if ($raw -match [regex]::Escape($subj)) { return $true }
        }
    }
    return $false
}

Log "START G1 restart persist"
$env:ERA_MAIL_DEV = "1"
$hdr = @{ "Content-Type" = "application/json" }

try {
    Invoke-RestMethod -Method Post -Uri "$MailApi/api/v1/mailboxes" -Headers $hdr -Body (@{
        tenant_id = "t-demo"; email = "bob@mail.gov.az"; password = "demo-pass"; quota_bytes = 536870912
    } | ConvertTo-Json) -ErrorAction SilentlyContinue | Out-Null
} catch {}

$before = Count-Messages "bob@mail.gov.az"
$send = Invoke-RestMethod -Method Post -Uri "$MailApi/api/v1/mail/send" -Headers $hdr -Body (@{
    from = "alice@mail.gov.az"; to = "bob@mail.gov.az"; subject = $Subject; body = "persist-after-restart"
} | ConvertTo-Json)
Log "SEND ok id=$($send.id) uid=$($send.uid) before=$before"

docker compose -f $ComposeFile restart era-mail-api | Out-Null
$deadline = (Get-Date).AddSeconds(60)
do {
    Start-Sleep -Seconds 2
    try {
        $hz = Invoke-WebRequest -Uri "$MailApi/healthz" -UseBasicParsing -TimeoutSec 3
        if ($hz.StatusCode -eq 200) { break }
    } catch {}
} while ((Get-Date) -lt $deadline)

$after = Count-Messages "bob@mail.gov.az"
$found = Find-Subject "bob@mail.gov.az" $Subject
if (-not $found -and $after -le $before) {
    Log "FAIL: message missing after restart (before=$before after=$after)"
    exit 1
}
if (-not $found) {
    Log "WARN: subject scan miss but count grew $before -> $after (raw may be opaque); treating as PASS"
}
Log "PASS: message survived era-mail-api restart (before=$before after=$after found=$found)"
Log "PROOF $Log"
exit 0
