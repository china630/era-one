#!/usr/bin/env pwsh
# Golden PII CI gate - reject unsanitized envelopes (S6-17).
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "==> PII gate (ingest + agent redaction tests)" -ForegroundColor Cyan

$goPkgs = @(
    "services/ingest-gateway/internal/ingest",
    "services/compliance/internal/report"
)
foreach ($p in $goPkgs) {
    Write-Host "go test $p" -ForegroundColor DarkGray
    Push-Location $p
    go test ./...
    if ($LASTEXITCODE -ne 0) { exit 1 }
    Pop-Location
}

Push-Location crates/era-agent
$prevEAP = $ErrorActionPreference
$ErrorActionPreference = "Continue"
cargo test -q pii
$cargoExit = $LASTEXITCODE
$ErrorActionPreference = $prevEAP
Pop-Location
if ($cargoExit -ne 0) {
    Write-Host "PII gate FAIL: cargo test pii exit $cargoExit" -ForegroundColor Red
    exit 1
}

Write-Host "PII gate PASS" -ForegroundColor Green
