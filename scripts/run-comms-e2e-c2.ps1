#!/usr/bin/env pwsh
# ERA Communications — Wave C-2 E2E (G2). Refs: Comms-Stage-C2-Spec.md §4
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$logPath = Join-Path $Root "reports/comms-stage-C-2-e2e.log"
$reportDir = Split-Path $logPath -Parent
if (-not (Test-Path $reportDir)) { New-Item -ItemType Directory -Path $reportDir | Out-Null }

function Log-Line {
    param([string]$Line)
    Write-Host $Line
    Add-Content -Path $logPath -Value $Line
}

Log-Line "==> C-2 E2E $ts"

Log-Line "--> docker compose clickhouse"
try {
    docker compose -f deploy/docker-compose.dev.yml up -d clickhouse 2>&1 | ForEach-Object { Log-Line $_ }
} catch {
    Log-Line "WARN: docker unavailable: $_"
}
$env:ERA_CH_ADDR = "127.0.0.1:9000"

Log-Line "--> stage gate C-2"
& "$Root/scripts/run-comms-stage-gate.ps1" -Stage C-2 2>&1 | ForEach-Object { Log-Line $_ }
if ($LASTEXITCODE -ne 0) { exit 1 }

Log-Line "--> CalDAV + EWS unit smoke"
go test ./services/comms/calendar/... ./services/comms/mail/internal/ews/... -count=1 2>&1 | ForEach-Object { Log-Line $_ }
if ($LASTEXITCODE -ne 0) { exit 1 }

Log-Line "C-2 E2E PASS"
exit 0
