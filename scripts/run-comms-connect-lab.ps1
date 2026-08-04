# B-CONN / RT-10 Connect lab against dovecot-lab (not customer field)
param(
    [string]$ConnectAPI = "http://127.0.0.1:8250",
    [string]$IMAPHost = "dovecot-lab",
    [int]$IMAPPort = 143,
    [switch]$UseCompose
)
$ErrorActionPreference = "Stop"
$log = Join-Path $PSScriptRoot "..\reports\comms-connect-lab.log"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null
function Log($msg) { $line = "$(Get-Date -Format o) CONNECT-LAB $msg"; Write-Host $line; Add-Content $log $line }

if ($UseCompose) {
    $compose = Join-Path $PSScriptRoot "..\deploy\docker-compose.comms.yml"
    $dev = Join-Path $PSScriptRoot "..\deploy\docker-compose.comms.dev.yml"
    docker compose -f $compose -f $dev up -d --wait dovecot-lab era-mail-connect 2>&1 | ForEach-Object { Log "compose: $_" }
}

$env:ERA_CONNECT_SECRET_LAB1 = if ($env:ERA_CONNECT_SECRET_LAB1) { $env:ERA_CONNECT_SECRET_LAB1 } else { "lab" }
$env:ERA_CONNECT_IMAP_HOST = if ($env:ERA_CONNECT_IMAP_HOST) { $env:ERA_CONNECT_IMAP_HOST } else { "dovecot-lab" }

Log "autodiscover"
try {
    $ad = Invoke-WebRequest -Uri "$ConnectAPI/api/v1/connect/autodiscover.xml?email=lab1@mail.gov.az" -UseBasicParsing
    Log "autodiscover: $($ad.Content.Substring(0, [Math]::Min(280, $ad.Content.Length)))"
    if ($ad.Content -notmatch "dovecot-lab|IMAP") {
        Log "autodiscover note: IMAP host block may be empty until ERA_CONNECT_IMAP_HOST in container"
    }
} catch {
    Log "autodiscover skip (API down): $($_.Exception.Message)"
}

Log "register mailbox (IMAP $IMAPHost`:$IMAPPort)"
$reg = @{
    tenant_id    = "t-demo"
    email        = "lab1@mail.gov.az"
    address      = "imap://${IMAPHost}:${IMAPPort}"
    username     = "lab1@mail.gov.az"
    password_ref = "vault://lab1"
} | ConvertTo-Json
try {
    Invoke-WebRequest -Uri "$ConnectAPI/api/v1/connect/mailboxes" -Method POST -Body $reg -ContentType "application/json" -UseBasicParsing | Out-Null
} catch {
    Log "register note: $($_.Exception.Message)"
}

Log "sync"
$syncBody = @{ tenant_id = "t-demo"; mailbox = "lab1@mail.gov.az" } | ConvertTo-Json
$sync = Invoke-WebRequest -Uri "$ConnectAPI/api/v1/connect/sync" -Method POST -Body $syncBody -ContentType "application/json" -UseBasicParsing
$job = $sync.Content | ConvertFrom-Json
Log "sync job: $($sync.Content)"
$okCount = 0
if ($null -ne $job.items_ok) { $okCount = [int]$job.items_ok }
elseif ($null -ne $job.ItemsOK) { $okCount = [int]$job.ItemsOK }
if ($okCount -le 0) { throw "RT-10 lab: items_ok must be > 0, got $($sync.Content)" }
Log "RT-10 Connect lab PASS (items_ok=$okCount) — see $log"
Write-Host "PASS — see $log"
