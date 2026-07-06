#!/usr/bin/env pwsh
# Full GA acceptance runner: smoke + unit tests + software gates
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "==> ERA XDR Full GA runner" -ForegroundColor Cyan

powershell -ExecutionPolicy Bypass -File scripts/run-ga1-smoke.ps1
if ($LASTEXITCODE -ne 0) { exit 1 }

powershell -ExecutionPolicy Bypass -File scripts/run-pii-gate.ps1
if ($LASTEXITCODE -ne 0) { exit 1 }

powershell -ExecutionPolicy Bypass -File scripts/run-ga-e2e-prod.ps1
if ($LASTEXITCODE -ne 0) { exit 1 }

$pkgs = @(
    "services/control-plane",
    "services/ingest-gateway",
    "services/event-writer",
    "services/detection-engine",
    "services/ai-core",
    "services/soar",
    "services/compliance",
    "services/federated",
    "services/license",
    "services/platform/custody",
    "services/platform/licensegate",
    "services/national-hub",
    "services/vm"
)
foreach ($p in $pkgs) {
    Write-Host "go test $p" -ForegroundColor DarkGray
    Push-Location $p
    go test ./...
    if ($LASTEXITCODE -ne 0) { exit 1 }
    Pop-Location
}

Push-Location crates/era-agent
cargo test -q
if ($LASTEXITCODE -ne 0) { exit 1 }
Pop-Location

Push-Location crates/era-collectors
cargo test -q
if ($LASTEXITCODE -ne 0) { exit 1 }
Pop-Location

if (Get-Command helm -ErrorAction SilentlyContinue) {
    helm template era-one deploy/helm/era-one/ | Out-Null
    Write-Host "Helm template: OK"
}

Write-Host "Loadgen prod (optional, requires stack): scripts/run-loadgen-prod.ps1" -ForegroundColor DarkGray
Write-Host "Full GA software runner PASS" -ForegroundColor Green
