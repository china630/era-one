#!/usr/bin/env pwsh
# Resolve MVP smoke: unit tests + optional live healthz/verdict
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$LogDir = Join-Path $Root "reports"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
$Stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$Log = Join-Path $LogDir "resolve-smoke-$Stamp.log"

function Log([string]$m) {
    Write-Host $m
    Add-Content -Path $Log -Value $m
}

Log "==> Resolve smoke"

Push-Location services/resolve
go test ./...
if ($LASTEXITCODE -ne 0) { Pop-Location; exit 1 }
Pop-Location
Log "resolve tests: PASS"

try {
    Invoke-WebRequest -Uri "http://127.0.0.1:8134/healthz" -TimeoutSec 2 | Out-Null
    Log "resolve: UP"
    $body = '{"qname":"lab.malware.test","qtype":"A"}'
    $v = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8134/api/v1/resolve/verdict" -ContentType "application/json" -Body $body
    Log "verdict action=$($v.action) source=$($v.source)"
} catch {
    Log "resolve live: not running (optional: compose --profile resolve)"
}

Log "Resolve smoke PASS — log $Log"
Write-Host "Resolve smoke PASS" -ForegroundColor Green
