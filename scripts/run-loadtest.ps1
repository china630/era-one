#!/usr/bin/env pwsh
# E2E smoke + load test helper (S1-9). Refs: AC2
param(
    [int]$Rate = 10000,
    [int]$DurationSec = 5
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "==> loadgen $Rate ev/s for ${DurationSec}s" -ForegroundColor Cyan
Push-Location services/ingest-gateway
go run ./cmd/loadgen -rate $Rate -duration "${DurationSec}s"
Pop-Location
