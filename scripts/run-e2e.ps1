#!/usr/bin/env pwsh
# E2E smoke: agent → gateway → Kafka → ClickHouse (Sprint-1 §4)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "==> E2E smoke (Sprint-1)" -ForegroundColor Cyan
Write-Host "Требуется: docker compose up, ingest-gateway :50051, event-writer :8089"

$count = docker exec era-clickhouse clickhouse-client --user era --password era_dev_pw --query "SELECT count() FROM era_xdr.events" 2>$null
Write-Host "ClickHouse events before: $count"

Write-Host "Запуск era-agent (stub, 6 сек)..."
$job = Start-Job {
    Set-Location $using:Root
    $env:ERA_CAPTURE_STUB = "1"
    $env:ERA_GATEWAY_ADDR = "http://127.0.0.1:50051"
    cargo run -q -p era-agent 2>&1
}
Start-Sleep -Seconds 6
Stop-Job $job -ErrorAction SilentlyContinue
Remove-Job $job -Force -ErrorAction SilentlyContinue

Start-Sleep -Seconds 3
$after = docker exec era-clickhouse clickhouse-client --user era --password era_dev_pw --query "SELECT count() FROM era_xdr.events"
$pii = docker exec era-clickhouse clickhouse-client --user era --password era_dev_pw --query "SELECT countIf(payload LIKE '%SECRET123%' OR payload LIKE '%alice%') FROM era_xdr.events"

Write-Host "ClickHouse events after:  $after"
Write-Host "PII leaks in payload:    $pii"
if ([int]$after -gt [int]$count -and [int]$pii -eq 0) {
    Write-Host "E2E PASS" -ForegroundColor Green
} else {
    Write-Host "E2E FAIL" -ForegroundColor Red
    exit 1
}
