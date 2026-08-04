# ERA Communications — 40k mailbox / 200 TB scale soak (Phase 4)
param(
    [int]$MailboxCount = 40000,
    [int]$Workers = 200,
    [switch]$DryRun
)
$ErrorActionPreference = "Stop"
$env:ERA_MIG_WORKERS = "$Workers"
$env:ERA_MIG_SCALE = "1"
$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$log = Join-Path $PSScriptRoot "..\reports\comms-scale-40k-$ts.log"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null
function Log($msg) { $line = "$(Get-Date -Format o) SCALE $msg"; Write-Host $line; Add-Content $log $line }

Log "Phase 4 scale gate: mailboxes=$MailboxCount workers=$Workers"
Log "See docs/Field-Server-Sizing.md for soak 7x24 guidance"

if ($DryRun) {
    Log "DRY-RUN: would enqueue $MailboxCount jobs with sharding (orchestrator.ShardKey)"
    Log "SCALE DRY-RUN PASS"
    exit 0
}

# Batched enqueue (100 per batch) to avoid API flood
$batches = [math]::Ceiling($MailboxCount / 100)
for ($b = 0; $b -lt $batches; $b++) {
    $count = [Math]::Min(100, $MailboxCount - ($b * 100))
    Log "batch $($b+1)/$batches count=$count"
    & (Join-Path $PSScriptRoot "run-comms-migration-pilot.ps1") -MailboxCount $count 2>&1 | ForEach-Object { Log $_ }
}
Log "SCALE enqueue complete - partner soak sign-off required for PASS"
