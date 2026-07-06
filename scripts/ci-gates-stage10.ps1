#!/usr/bin/env pwsh

# Stage 10 CI gates: edition-matrix, security tests, bundle golden, platform services (S-05).

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot

Set-Location $Root

$tests = @(
    "./services/platform/licensegate/...",
    "./services/platform/httpserver/...",
    "./services/observe/...",
    "./services/control-plane/internal/networkreconcile/...",
    "./services/update-service/internal/bundle/...",
    "./services/update-service/internal/api/...",
    "./services/pam/...",
    "./services/license/internal/license/...",
    "./services/ingest-gateway/internal/server/...",
    "./services/ingest-gateway/internal/pipeline/...",
    "./services/event-writer/internal/consumer/...",
    "./services/federated/...",
    "./services/national-hub/internal/taxii/..."
)

foreach ($t in $tests) {
    Write-Host "==> go test $t" -ForegroundColor Cyan
    go test $t
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

Write-Host "==> exposure score tests" -ForegroundColor Cyan
go test ./services/detection-engine/internal/exposure/...
if ($LASTEXITCODE -ne 0) { exit 1 }

Write-Host "==> ADR-0006 golden (deception/ctem/compliance/risk/ndr)" -ForegroundColor Cyan
go test ./services/detection-engine/internal/deception/... `
    ./services/detection-engine/internal/ctem/... `
    ./services/detection-engine/internal/compliance/... `
    ./services/detection-engine/internal/risk/... `
    ./services/detection-engine/internal/ndr/... -run Golden
if ($LASTEXITCODE -ne 0) { exit 1 }

Write-Host "==> hybrid relay + TI outbound" -ForegroundColor Cyan
go test ./services/control-plane/internal/hybrid/... -count=1
if ($LASTEXITCODE -ne 0) { exit 1 }

if (Test-Path "services/control-plane/internal/store/parity_test.go") {
    Write-Host "==> postgres parity (docker)" -ForegroundColor Cyan
    $pgName = "era-ci-pg-$(Get-Random)"
    $pgPort = 55432 + (Get-Random -Maximum 1000)
    $pgStarted = $false
    try {
        docker run -d --name $pgName -e POSTGRES_USER=era -e POSTGRES_PASSWORD=era_ci_pw -e POSTGRES_DB=era_cp -p "${pgPort}:5432" postgres:16-alpine 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            $pgStarted = $true
            $deadline = (Get-Date).AddSeconds(45)
            while ((Get-Date) -lt $deadline) {
                docker exec $pgName pg_isready -U era 2>$null | Out-Null
                if ($LASTEXITCODE -eq 0) { break }
                Start-Sleep -Seconds 2
            }
            $env:ERA_STORE_PG_DSN = "postgres://era:era_ci_pw@127.0.0.1:${pgPort}/era_cp?sslmode=disable"
            Push-Location services/control-plane
            go test ./internal/store/... -run TestPostgresParity -count=1
            if ($LASTEXITCODE -ne 0) { exit 1 }
            Pop-Location
        } else {
            Write-Host "docker unavailable - postgres parity skipped" -ForegroundColor Yellow
        }
    } finally {
        if ($pgStarted) {
            docker rm -f $pgName 2>$null | Out-Null
        }
        Remove-Item Env:ERA_STORE_PG_DSN -ErrorAction SilentlyContinue
    }
}

if (Test-Path "services/event-writer/internal/timeline/testdata/timeline_merged.golden.json") {
    Write-Host "==> workbench timeline golden" -ForegroundColor Cyan
    go test ./services/event-writer/internal/timeline/...
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

Write-Host "==> PII golden (agent)" -ForegroundColor Cyan
Push-Location crates/era-agent
$prev = $ErrorActionPreference
$ErrorActionPreference = "Continue"
cmd /c "cargo test golden_pii 2>nul"
$code = $LASTEXITCODE
$ErrorActionPreference = $prev
if ($code -ne 0) { Pop-Location; exit 1 }
Pop-Location

Write-Host "==> Agent budget + tamper prod guard" -ForegroundColor Cyan
Push-Location crates/era-agent-core
$prev = $ErrorActionPreference
$ErrorActionPreference = "Continue"
cmd /c "cargo test budget_guard:: 2>nul"
$code = $LASTEXITCODE
if ($code -eq 0) {
    cmd /c "cargo test tamper:: 2>nul"
    $code = $LASTEXITCODE
}
$ErrorActionPreference = $prev
if ($code -ne 0) { Pop-Location; exit 1 }
Pop-Location

Write-Host "==> era-plugin-vuln (L-05)" -ForegroundColor Cyan
Push-Location crates/era-plugin-vuln
$prev = $ErrorActionPreference
$ErrorActionPreference = "Continue"
cmd /c "cargo test 2>nul"
$code = $LASTEXITCODE
$ErrorActionPreference = $prev
Pop-Location
if ($code -ne 0) { exit 1 }

if (Get-Command helm -ErrorAction SilentlyContinue) {
    & "$PSScriptRoot/helm-template-check.ps1"
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

Write-Host "Stage 10 CI gates PASS" -ForegroundColor Green
