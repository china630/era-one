#!/usr/bin/env pwsh
param([switch]$AllowSkipCH)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$logPath = Join-Path $Root "reports/comms-stage-C-GA-e2e.log"
if (-not (Test-Path (Split-Path $logPath -Parent))) { New-Item -ItemType Directory -Path (Split-Path $logPath -Parent) | Out-Null }

function Log-Line { param([string]$Line) Write-Host $Line; Add-Content -Path $logPath -Value $Line }

Log-Line "==> C-GA E2E $ts"
if ($AllowSkipCH) {
  & "$Root/scripts/run-comms-stage-gate.ps1" -Stage C-GA -AllowSkipCH 2>&1 | ForEach-Object { Log-Line $_ }
} else {
  & "$Root/scripts/run-comms-stage-gate.ps1" -Stage C-GA 2>&1 | ForEach-Object { Log-Line $_ }
}
if ($LASTEXITCODE -ne 0) { exit 1 }
Log-Line "C-GA E2E PASS"
