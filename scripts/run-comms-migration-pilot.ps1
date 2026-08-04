# CG → IceWarp / ERA pilot wave (lab default: file dry-run without live IMAP)
param(
    [int]$MailboxCount = 50,
    [string]$MigrationAPI = "http://127.0.0.1:8350",
    [string]$SourceHost = $env:ERA_MIG_SOURCE_IMAP_HOST,
    [string]$TargetHost = $env:ERA_MIG_TARGET_IMAP_HOST,
    [switch]$UseCompose,
    [switch]$AllowDryRun
)
$ErrorActionPreference = "Stop"
$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$log = Join-Path $PSScriptRoot "..\reports\comms-migration-pilot-$ts.log"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null
function Log($msg) { $line = "$(Get-Date -Format o) PILOT $msg"; Write-Host $line; Add-Content $log $line }

if ($UseCompose) {
    Log "compose partner profile"
    & (Join-Path $PSScriptRoot "run-comms-pilot-field-lab.ps1") -UseCompose | Out-Null
}

$liveIMAP = -not [string]::IsNullOrWhiteSpace($env:ERA_MIG_SOURCE_IMAP_HOST)
if (-not $liveIMAP) {
    Log "ERA_MIG_SOURCE_IMAP_HOST unset — dry-run file jobs (not field cutover)"
    $AllowDryRun = $true
}
if (-not $SourceHost) { $SourceHost = "cg.lab.local" }
if (-not $TargetHost) { $TargetHost = "icewarp.lab.local" }

Log "discover CG source"
$discBody = @{
    source = "communigate"
    source_imap = @{
        host = $SourceHost
        port = 143
        user = "pilot@cg.lab.local"
        password_ref = "env:CG_LAB_PASSWORD"
    }
} | ConvertTo-Json -Depth 5
try {
    $disc = Invoke-WebRequest -Uri "$MigrationAPI/api/v1/migration/discover" -Method POST -Body $discBody -ContentType "application/json" -UseBasicParsing
    Log "discover: $($disc.Content)"
} catch {
    Log "discover skip (lab CG unavailable): $($_.Exception.Message)"
}

$ok = 0
for ($i = 1; $i -le $MailboxCount; $i++) {
    $mb = "pilot$i@mail.gov.az"
    if ($AllowDryRun -or -not $liveIMAP) {
        # File paths on host are not visible inside migration-api container — use archive smoke only.
        $jobBody = (@{
            source       = "imap"
            mailbox      = $mb
            archive_file = "dry.pst"
            mode         = "bulk"
        } | ConvertTo-Json -Depth 5)
    } else {
        $jobBody = (@{
            source       = "communigate"
            mailbox      = $mb
            target       = "era-mail-server"
            folder       = "INBOX"
            mode         = "bulk"
            mail_api_url = "http://127.0.0.1:8150"
            source_imap  = @{
                host         = $SourceHost
                port         = 143
                user         = "pilot$i@cg.lab.local"
                password_ref = "env:CG_LAB_PASSWORD"
            }
        } | ConvertTo-Json -Depth 5)
    }
    try {
        $resp = Invoke-WebRequest -Uri "$MigrationAPI/api/v1/migration/jobs" -Method POST -Body $jobBody -ContentType "application/json" -UseBasicParsing
        Log "job $i queued: $($resp.Content)"
        $ok++
    } catch {
        Log "job $i skip: $($_.Exception.Message)"
    }
    Start-Sleep -Milliseconds 50
}

Log "PILOT queued $ok / $MailboxCount jobs — log $log"
if ($ok -eq 0) { throw "PILOT failed: 0 jobs queued" }
if ($AllowDryRun -or -not $liveIMAP) {
    Log "PILOT LAB DRY-RUN PASS (file jobs; field IMAP still open)"
} else {
    Log "PILOT LIVE IMAP PASS"
}
