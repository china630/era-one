#!/usr/bin/env pwsh
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$logPath = Join-Path $Root "reports/comms-stage-C-1.1-e2e.log"
if (-not (Test-Path (Split-Path $logPath -Parent))) { New-Item -ItemType Directory -Path (Split-Path $logPath -Parent) | Out-Null }

function Log-Line { param([string]$Line) Write-Host $Line; Add-Content -Path $logPath -Value $Line }

Log-Line "==> C-1.1 E2E $ts"
Log-Line "--> go test mail-connect"
go test ./services/comms/mail-connect/... -count=1 2>&1 | ForEach-Object { Log-Line $_ }
if ($LASTEXITCODE -ne 0) { exit 1 }
Log-Line "--> stage gate C-1.1"
& "$Root/scripts/run-comms-stage-gate.ps1" -Stage C-1.1 2>&1 | ForEach-Object { Log-Line $_ }
if ($LASTEXITCODE -ne 0) { exit 1 }
Log-Line "C-1.1 E2E PASS"
