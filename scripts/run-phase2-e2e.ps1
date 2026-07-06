#!/usr/bin/env pwsh
# Phase-2 E2E smoke: multi-domain telemetry, detection, AI, SOAR, cases, assets (F2-*)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "==> Phase-2 E2E smoke" -ForegroundColor Cyan

if (-not (Test-Path "data/sigma-corpus/rules/era-sigma-0500.yml")) {
    Write-Host "Generating sigma corpus..."
    Push-Location scripts/gen-sigma-corpus
    go run .
    Pop-Location
}
$ruleCount = (Get-ChildItem data/sigma-corpus/rules -Filter *.yml).Count
Write-Host "Sigma rules: $ruleCount"
if ($ruleCount -lt 500) { Write-Host "FAIL: corpus < 500"; exit 1 }

Push-Location services/detection-engine; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/control-plane; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/ai-core; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/soar; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location
Push-Location services/platform; go test ./...; if ($LASTEXITCODE -ne 0) { exit 1 }; Pop-Location

Push-Location crates/era-agent
cargo test -q
if ($LASTEXITCODE -ne 0) { exit 1 }
Pop-Location

docker exec era-clickhouse clickhouse-client --user era --password era_dev_pw --query "SELECT 1" 2>$null | Out-Null
if ($LASTEXITCODE -eq 0) {
    Write-Host "Injecting multi-domain events (ERA_DOMAIN_STUB)..."
    $job = Start-Job {
        Set-Location $using:Root
        $env:ERA_DOMAIN_STUB = "1"
        $env:ERA_TAMPER_SIM = "1"
        $env:ERA_GATEWAY_ADDR = "http://127.0.0.1:50051"
        $env:ERA_CONTROL_PLANE_URL = "http://127.0.0.1:8090"
        cargo run -q -p era-agent 2>&1
    }
    Start-Sleep -Seconds 8
    Stop-Job $job -ErrorAction SilentlyContinue
    Remove-Job $job -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 4

    $net = docker exec era-clickhouse clickhouse-client --user era --password era_dev_pw --query "SELECT countIf(category='network') FROM era_xdr.events"
    $auth = docker exec era-clickhouse clickhouse-client --user era --password era_dev_pw --query "SELECT countIf(category='auth') FROM era_xdr.events"
    $file = docker exec era-clickhouse clickhouse-client --user era --password era_dev_pw --query "SELECT countIf(category='file') FROM era_xdr.events"
    Write-Host "Categories in CH: network=$net auth=$auth file=$file"
    if ([int]$net -lt 1 -or [int]$auth -lt 1 -or [int]$file -lt 1) {
        Write-Host "WARN: multi-domain E2E needs ingest-gateway + event-writer running" -ForegroundColor Yellow
    }
} else {
    Write-Host "ClickHouse not available - skipping live E2E" -ForegroundColor Yellow
}

try {
    $cov = Invoke-RestMethod -Uri "http://127.0.0.1:8090/api/v1/assets" -TimeoutSec 2
    Write-Host "Asset coverage: $($cov.coverage)"
} catch {
    Write-Host "control-plane :8090 not running (optional)"
}

try {
    $case = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8090/api/v1/cases" -ContentType "application/json" -Body '{"title":"Phase2 test","node_id":"node-01"}' -TimeoutSec 2
    Write-Host "Case created: $($case.id)"
} catch { }

try {
    $inv = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8091/api/v1/investigate" -ContentType "application/json" -Body '{"node_id":"node-01"}' -TimeoutSec 2
    Write-Host "AI verdict: $($inv.verdict) conf=$($inv.confidence)"
} catch {
    Write-Host "ai-core :8091 not running (optional)"
}

try {
    Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8092/api/v1/playbooks/isolate_host" -ContentType "application/json" -Body '{"node_id":"node-01"}' -TimeoutSec 2 | Out-Null
    Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8092/api/v1/playbooks/block_ip" -ContentType "application/json" -Body '{"ip":"10.0.0.5"}' -TimeoutSec 2 | Out-Null
    Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8092/api/v1/playbooks/create_ticket" -ContentType "application/json" -Body '{"title":"incident","case_id":"c1"}' -TimeoutSec 2 | Out-Null
    Write-Host "SOAR playbooks: OK"
} catch {
    Write-Host "soar :8092 not running (optional)"
}

Write-Host "Phase-2 smoke PASS (unit + corpus)" -ForegroundColor Green
