#!/usr/bin/env pwsh
# ERA Office — weekly acceptance smoke (Refs: Office-Acceptance-System.md)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "==> ERA Office acceptance smoke" -ForegroundColor Cyan

$stages = @('O-GOV', 'O-PILOT')
# Add closed waves here as implementation progresses:
# $stages += @('O-0', 'O-1', ...)

$failed = @()
foreach ($s in $stages) {
    Write-Host ""
    Write-Host "--- Stage $s ---" -ForegroundColor Yellow
    & "$PSScriptRoot/run-office-stage-gate.ps1" -Stage $s
    if ($LASTEXITCODE -ne 0) { $failed += $s }
}

Write-Host ""
Write-Host "==> Summary" -ForegroundColor Cyan
if ($failed.Count -eq 0) {
    Write-Host "OFFICE ACCEPTANCE PASS ($($stages -join ', '))" -ForegroundColor Green
    Write-Host "Matrix: docs/Office-Implementation-Matrix.md"
    Write-Host "Gap:    docs/Office-Pilot-Gap-List.md"
    exit 0
}

Write-Host "OFFICE ACCEPTANCE FAIL: $($failed -join ', ')" -ForegroundColor Red
exit 1
