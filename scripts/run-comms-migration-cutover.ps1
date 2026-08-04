# RT-11 cutover rehearsal — final delta + bridge switch checklist (lab)
param(
    [string]$MigrationAPI = "http://127.0.0.1:8350",
    [string]$MailBridge = "http://127.0.0.1:8151",
    [switch]$UseCompose
)
$ErrorActionPreference = "Stop"
$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$log = Join-Path $PSScriptRoot "..\reports\comms-migration-cutover-$ts.log"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null
function Log($msg) { $line = "$(Get-Date -Format o) CUTOVER $msg"; Write-Host $line; Add-Content $log $line }

if ($UseCompose) {
    & (Join-Path $PSScriptRoot "run-comms-pilot-field-lab.ps1") -UseCompose | Out-Null
}

Log "Step 1: final delta migration jobs (mode=delta)"
if ([string]::IsNullOrWhiteSpace($env:ERA_MIG_SOURCE_IMAP_HOST)) {
    $deltaBody = @{
        source       = "imap"
        mailbox      = "pilot@mail.gov.az"
        archive_file = "delta.pst"
        mode         = "delta"
    } | ConvertTo-Json -Depth 5
} else {
    $deltaBody = @{
        source      = "communigate"
        mailbox     = "pilot@mail.gov.az"
        target      = "era-mail-server"
        mode        = "delta"
        folder      = "INBOX"
        source_imap = @{ host = $env:ERA_MIG_SOURCE_IMAP_HOST; port = 143; user = "pilot@cg.lab.local"; password_ref = "env:CG_LAB_PASSWORD" }
    } | ConvertTo-Json -Depth 5
}
try {
    $r = Invoke-WebRequest -Uri "$MigrationAPI/api/v1/migration/jobs" -Method POST -Body $deltaBody -ContentType "application/json" -UseBasicParsing
    Log "delta job: $($r.Content)"
} catch {
    Log "delta skip: $($_.Exception.Message)"
}

Log "Step 2: rerun idempotency probe"
$rerun = Invoke-WebRequest -Uri "$MigrationAPI/api/v1/migration/rerun" -Method POST -Body '{"source_uids":["pilot@mail.lab.local:INBOX:1","pilot@mail.lab.local:INBOX:1"]}' -ContentType "application/json" -UseBasicParsing
Log "rerun: $($rerun.Content)"

Log "Step 3: switch Autodiscover to Bridge (manual MX/AD — see Comms-Cutover-Rehearsal-Runbook.md)"
$ad = Invoke-WebRequest -Uri "$MailBridge/autodiscover/autodiscover.xml?email=pilot@mail.gov.az" -UseBasicParsing
Log "bridge autodiscover EwsUrl present: $($ad.Content -match '/ews/Exchange.asmx')"

Log "Step 4: Outlook VM send/receive smoke - MANUAL (partner site RT-11)"
Log "Step 5: decommission CG SMTP after MX cutover - MANUAL"
Log "CUTOVER rehearsal log complete: $log"
