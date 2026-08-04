#!/usr/bin/env pwsh
# Observe Path B smoke: unit tests + optional live healthz
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$LogDir = Join-Path $Root "reports"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
$Stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$Log = Join-Path $LogDir "observe-smoke-$Stamp.log"

function Log([string]$m) {
    Write-Host $m
    Add-Content -Path $Log -Value $m
}

Log "==> Observe smoke"

Push-Location services/observe
go test ./...
if ($LASTEXITCODE -ne 0) { exit 1 }
Pop-Location
Log "observe tests: PASS"

try {
    Invoke-WebRequest -Uri "http://127.0.0.1:8132/healthz" -TimeoutSec 2 | Out-Null
    Log "observe: UP"
    $poll = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8132/api/v1/snmp/poll?target=127.0.0.1"
    Log "snmp poll metrics_source=$($poll.metrics_source)"
} catch {
    Log "observe live: not running (optional: compose --profile observe)"
}

Log "Observe smoke PASS — log $Log"
Write-Host "Observe smoke PASS" -ForegroundColor Green
