# CG → IceWarp pilot wave (Phase 2 lab, default 50 mailboxes)
param(
    [int]$MailboxCount = 50,
    [string]$MigrationAPI = "http://127.0.0.1:8350",
    [string]$SourceHost = $env:ERA_MIG_SOURCE_IMAP_HOST,
    [string]$TargetHost = $env:ERA_MIG_TARGET_IMAP_HOST,
    [switch]$UseCompose
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
    $mb = "pilot$i@mail.lab.local"
    $jobBody = @{
        source = "communigate"
        mailbox = $mb
        target = "icewarp"
        folder = "INBOX"
        mode = "bulk"
        source_imap = @{
            host = $SourceHost
            port = 143
            user = "pilot$i@cg.lab.local"
            password_ref = "env:CG_LAB_PASSWORD"
        }
        target_imap = @{
            host = $TargetHost
            port = 143
            user = $mb
            password_ref = "env:ICEWARP_LAB_PASSWORD"
        }
    } | ConvertTo-Json -Depth 5
    try {
        $resp = Invoke-WebRequest -Uri "$MigrationAPI/api/v1/migration/jobs" -Method POST -Body $jobBody -ContentType "application/json" -UseBasicParsing
        Log "job $i queued: $($resp.Content)"
        $ok++
    } catch {
        Log "job $i skip: $($_.Exception.Message)"
    }
    Start-Sleep -Milliseconds 100
}

Log "PILOT queued $ok / $MailboxCount jobs — log $log"
