#!/usr/bin/env pwsh
# Phase-3 E2E smoke: WAF, NGFW, DLP, NDR, Federated, Deception (F3-*)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "==> Phase-3 E2E smoke" -ForegroundColor Cyan

Push-Location services/waf; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/ngfw; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/dlp; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/federated; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/deception; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/ctem; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/detection-engine; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/platform; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location

# F3-6: federated off by default
Push-Location services/platform
go test -run TestDevDefaultFederatedOff ./licensegate/...
if ($LASTEXITCODE -ne 0) { exit 1 }
Pop-Location

Write-Host "Phase-3 smoke PASS" -ForegroundColor Green
