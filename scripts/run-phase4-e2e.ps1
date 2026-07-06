#!/usr/bin/env pwsh
# Phase-4 E2E smoke: STIX/TAXII hub, PII audit, national detection, compliance (F4-*)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "==> Phase-4 E2E smoke" -ForegroundColor Cyan

Push-Location services/national-hub; go mod tidy; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/compliance; go mod tidy; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/license; go test ./internal/license/... -run "TestPQC"; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/detection-engine; go test ./internal/tip/...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/platform; go test -run TestDevDefaultFederatedOff ./licensegate/...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location

Write-Host "Phase-4 smoke PASS" -ForegroundColor Green
