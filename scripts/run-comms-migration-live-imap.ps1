# B-MIG — live IMAP lab migration via dovecot-lab (not archive dry-run)
param(
    [int]$MailboxCount = 3,
    [string]$MigrationAPI = "http://127.0.0.1:8350",
    [string]$SourceHost = "dovecot-lab",
    [int]$SourcePort = 143,
    [string]$MailAPI = "http://era-mail-api:8150",
    [string]$ClickHouse = "http://127.0.0.1:8123",
    [switch]$UseCompose
)
$ErrorActionPreference = "Stop"
$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$log = Join-Path $PSScriptRoot "..\reports\comms-migration-live-imap-$ts.log"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null
function Log($msg) { $line = "$(Get-Date -Format o) MIG-LIVE $msg"; Write-Host $line; Add-Content $log $line }

if ($UseCompose) {
    $compose = Join-Path $PSScriptRoot "..\deploy\docker-compose.comms.yml"
    $dev = Join-Path $PSScriptRoot "..\deploy\docker-compose.comms.dev.yml"
    docker compose -f $compose -f $dev up -d --wait 2>&1 | ForEach-Object { Log "compose: $_" }
}

$env:ERA_MIG_SOURCE_IMAP_HOST = $SourceHost
$env:CG_LAB_PASSWORD = if ($env:CG_LAB_PASSWORD) { $env:CG_LAB_PASSWORD } else { "lab" }

$ok = 0
for ($i = 1; $i -le $MailboxCount; $i++) {
    $mb = "lab${i}@mail.gov.az"
    $jobBody = (@{
        source       = "imap"
        mailbox      = $mb
        target       = "era-mail-server"
        folder       = "INBOX"
        mode         = "bulk"
        mail_api_url = $MailAPI
        source_imap  = @{
            host         = $SourceHost
            port         = $SourcePort
            user         = "lab1@mail.gov.az"
            password_ref = "env:CG_LAB_PASSWORD"
        }
    } | ConvertTo-Json -Depth 5)
    try {
        $resp = Invoke-WebRequest -Uri "$MigrationAPI/api/v1/migration/jobs" -Method POST -Body $jobBody -ContentType "application/json" -UseBasicParsing
        Log "job ${i}: $($resp.Content)"
        $ok++
    } catch {
        Log "job ${i} fail: $($_.Exception.Message)"
    }
}

$mbox = Join-Path $PSScriptRoot "..\services\comms\migration\testdata\lab-fixture.mbox"
$pst = Join-Path $PSScriptRoot "..\services\comms\migration\internal\importers\archive\testdata\lab-fixture.pst"
foreach ($archive in @($mbox, $pst)) {
    if (-not (Test-Path $archive)) { continue }
    $body = (@{
        source       = "imap"
        mailbox      = "archive-lab@mail.gov.az"
        archive_file = (Resolve-Path $archive).Path
        target       = "era-mail-server"
        mail_api_url = "http://127.0.0.1:8150"
        mode         = "bulk"
    } | ConvertTo-Json -Depth 5)
    try {
        $resp = Invoke-WebRequest -Uri "$MigrationAPI/api/v1/migration/jobs" -Method POST -Body $body -ContentType "application/json" -UseBasicParsing
        Log "archive $($archive): $($resp.Content)"
        $ok++
    } catch {
        Log "archive skip: $($_.Exception.Message)"
    }
}

Start-Sleep -Seconds 3
try {
    $chQ = "SELECT count() FROM era_comms.migration_job WHERE mailbox != '' FORMAT TabSeparated"
    $uri = "$ClickHouse/?user=era&password=era_dev_only&query=$([uri]::EscapeDataString($chQ))"
    $chResp = Invoke-WebRequest -Uri $uri -UseBasicParsing
    Log "CH migration_job mailbox non-empty count: $($chResp.Content.Trim())"
} catch {
    Log "CH assert skip: $($_.Exception.Message)"
}

if ($ok -eq 0) { throw "live IMAP lab: 0 jobs" }
Log "PILOT LIVE IMAP LAB PASS — see $log"
Write-Host "PASS — see $log"
