#!/usr/bin/env pwsh
# ERA Office — O-Pilot E2E orchestrator (Playwright + optional compose)
param(
    [switch]$UseCompose,
    [switch]$SkipPlaywright
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$reportDir = Join-Path $Root "reports"
if (-not (Test-Path $reportDir)) { New-Item -ItemType Directory -Path $reportDir | Out-Null }
$logPath = Join-Path $reportDir "office-stage-O-PILOT-e2e-$ts.log"

function Log-Line {
    param([string]$Line)
    Write-Host $Line
    Add-Content -Path $logPath -Value $Line
}

Log-Line "==> ERA Office O-Pilot E2E $ts"

if ($UseCompose) {
    Log-Line "[compose] docker compose up --wait"
    $compose = Join-Path $Root "deploy\docker-compose.office.yml"
    docker compose -f $compose up -d --wait
    if ($LASTEXITCODE -ne 0) {
        Log-Line "[FAIL] compose up"
        exit 1
    }
    $env:ERA_E2E_SKIP_SERVER = "1"
    $env:ERA_WORKSPACE_URL = "http://127.0.0.1:8170"
}

if (-not $SkipPlaywright) {
    Log-Line "[playwright] ui/office/e2e"
    $e2eDir = Join-Path $Root "ui\office\e2e"
    Push-Location $e2eDir
    if (-not (Test-Path "node_modules")) {
        npm install --no-audit --no-fund 2>&1 | Out-Null
    }
    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    npm test 2>&1 | Tee-Object -FilePath $logPath -Append
    $pwExit = $LASTEXITCODE
    $ErrorActionPreference = $prevEap
    Pop-Location
    if ($pwExit -ne 0) {
        Log-Line "[FAIL] Playwright exit $pwExit"
        exit $pwExit
    }
    Log-Line "[PASS] Playwright"
}

Log-Line "O-PILOT E2E PASS - log: $logPath"
