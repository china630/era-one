# ERA Communications — 1k mailbox migration pilot
param(
    [int]$MailboxCount = 1000,
    [string]$MigrationAPI = "http://127.0.0.1:8350",
    [switch]$UseCompose
)
$ErrorActionPreference = "Stop"
$env:ERA_MIG_WORKERS = if ($env:ERA_MIG_WORKERS) { $env:ERA_MIG_WORKERS } else { "48" }
$env:ERA_MIG_SCALE = "pilot1k"
$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$log = Join-Path $PSScriptRoot "..\reports\comms-migration-pilot-1k-$ts.log"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null
function Log($msg) { $line = "$(Get-Date -Format o) PILOT-1K $msg"; Write-Host $line; Add-Content $log $line }

if ($UseCompose) {
    & (Join-Path $PSScriptRoot "run-comms-pilot-field-lab.ps1") -UseCompose | Out-Null
}

Log "workers=$env:ERA_MIG_WORKERS scale=$env:ERA_MIG_SCALE mailboxes=$MailboxCount"
& (Join-Path $PSScriptRoot "run-comms-migration-pilot.ps1") -MailboxCount $MailboxCount -MigrationAPI $MigrationAPI 2>&1 | ForEach-Object { Log $_ }
Log "PILOT-1K complete — log $log"
