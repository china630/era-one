#!/usr/bin/env pwsh
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$logPath = Join-Path $Root "reports/comms-stage-C-5-e2e.log"
$scaleLog = Join-Path $Root "reports/comms-scale-60k.log"
if (-not (Test-Path (Split-Path $logPath -Parent))) { New-Item -ItemType Directory -Path (Split-Path $logPath -Parent) | Out-Null }

function Log-Line { param([string]$Line) Write-Host $Line; Add-Content -Path $logPath -Value $Line }

Log-Line "==> C-5 E2E $ts"

# Start comms-ai in background for loadgen smoke
$aiJob = Start-Job -ScriptBlock {
    Set-Location $using:Root
    $env:ERA_COMMS_AI_DEV = "1"
    $env:ERA_COMMS_AI_ADDR = ":18096"
    go run ./services/comms/ai/cmd/comms-ai
}
Start-Sleep -Seconds 2

try {
    go test ./services/comms/ai/... ./services/comms/cmd/loadgen-mailboxes/... -count=1 2>&1 | ForEach-Object { Log-Line $_ }
    if ($LASTEXITCODE -ne 0) { exit 1 }

    Log-Line "==> loadgen quick smoke (1000 mailboxes)"
    go run ./services/comms/cmd/loadgen-mailboxes -addr http://127.0.0.1:18096 -mailboxes 1000 -quick -workers 16 -log $scaleLog 2>&1 | ForEach-Object { Log-Line $_ }
    if ($LASTEXITCODE -ne 0) { exit 1 }

    & "$Root/scripts/run-comms-stage-gate.ps1" -Stage C-5 2>&1 | ForEach-Object { Log-Line $_ }
    if ($LASTEXITCODE -ne 0) { exit 1 }

    Log-Line "C-5 E2E PASS"
}
finally {
    Stop-Job $aiJob -ErrorAction SilentlyContinue | Out-Null
    Remove-Job $aiJob -Force -ErrorAction SilentlyContinue | Out-Null
}
