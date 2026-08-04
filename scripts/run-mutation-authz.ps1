#!/usr/bin/env pwsh
# Optional mutation / fake-test detector for AuthZ hotspots (nightly / manual).
# Not a PR-blocking gate yet — exit 0 with SKIP if tools missing.
param(
    [ValidateSet("go-httpauth", "all")]
    [string]$Target = "go-httpauth"
)

$ErrorActionPreference = "Continue"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "==> mutation authz target=$Target"

if ($Target -eq "go-httpauth" -or $Target -eq "all") {
    $pkg = "services/comms/internal/httpauth"
    if (-not (Test-Path (Join-Path $Root $pkg))) {
        Write-Host "SKIP: $pkg missing"
        exit 0
    }
    # Prefer go-mutesting if present; else run tests with -count=1 as smoke.
    if (Get-Command go-mutesting -ErrorAction SilentlyContinue) {
        Write-Host "Running go-mutesting on $pkg"
        go-mutesting "./$pkg"
        exit $LASTEXITCODE
    }
    Write-Host "go-mutesting not installed — running reinforced tests instead"
    go test -count=3 "./$pkg/..."
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Write-Host "HINT: install github.com/zimmski/go-mutesting for real mutation scores"
}

Write-Host "PASS (or SKIP) mutation authz helper"
exit 0
