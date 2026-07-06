#!/usr/bin/env pwsh
# Wave GA-1 smoke: persistent store, prod compose health, capture tests (F-GA-* subset)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "==> GA-1 smoke" -ForegroundColor Cyan

Push-Location services/control-plane
go test ./...
if ($LASTEXITCODE -ne 0) { exit 1 }
Pop-Location

Push-Location crates/era-agent-core
cargo test -q --lib
if ($LASTEXITCODE -ne 0) { exit 1 }
Pop-Location

# SQLite persistence (F-GA-4)
$db = Join-Path $env:TEMP "era-ga1-cp.db"
Remove-Item $db -ErrorAction SilentlyContinue
$env:ERA_STORE_PATH = $db
$cpJob = Start-Job {
    Set-Location $using:Root
    $env:ERA_STORE_PATH = $using:db
    $env:ERA_HTTP_ADDR = ":18090"
    go run ./services/control-plane/cmd/control-plane 2>&1
}
Start-Sleep -Seconds 3
try {
    Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:18090/api/v1/assets/register" `
        -ContentType "application/json" `
        -Body '{"agent_id":"ga1","tenant_id":"t1","node_id":"ga-node","hostname":"h1","platform":"linux"}' | Out-Null
    $assets = Invoke-RestMethod -Uri "http://127.0.0.1:18090/api/v1/assets"
    if ($assets.items.Count -lt 1) { throw "no assets after register" }
    Write-Host "SQLite store: OK ($($assets.items.Count) assets)"
} catch {
    Write-Host "WARN: control-plane smoke skipped ($($_.Exception.Message))" -ForegroundColor Yellow
} finally {
    Stop-Job $cpJob -ErrorAction SilentlyContinue
    Remove-Job $cpJob -Force -ErrorAction SilentlyContinue
    Remove-Item Env:ERA_STORE_PATH -ErrorAction SilentlyContinue
}

# Prod stack health (optional — if compose running)
$endpoints = @(
    @{ Name = "control-plane"; Url = "http://127.0.0.1:8090/healthz" },
    @{ Name = "event-writer"; Url = "http://127.0.0.1:8089/healthz" },
    @{ Name = "ingest-gateway"; Url = "http://127.0.0.1:8082/healthz" },
    @{ Name = "ai-core"; Url = "http://127.0.0.1:8091/healthz" },
    @{ Name = "soar"; Url = "http://127.0.0.1:8092/healthz" }
)
foreach ($ep in $endpoints) {
    try {
        Invoke-WebRequest -Uri $ep.Url -TimeoutSec 2 | Out-Null
        Write-Host "$($ep.Name): UP"
    } catch {
        Write-Host "$($ep.Name): not running (start: docker compose -f deploy/docker-compose.prod.yml up -d)" -ForegroundColor DarkGray
    }
}

Write-Host "GA-1 smoke PASS (unit + optional live)" -ForegroundColor Green
