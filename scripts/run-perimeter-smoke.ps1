#!/usr/bin/env pwsh
# Perimeter MVP smoke: waf + ngfw + dlp unit tests + optional live healthz
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$LogDir = Join-Path $Root "reports"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
$Stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$Log = Join-Path $LogDir "perimeter-smoke-$Stamp.log"

function Log([string]$m) {
    Write-Host $m
    Add-Content -Path $Log -Value $m
}

Log "==> Perimeter smoke"

foreach ($svc in @("waf", "ngfw", "dlp")) {
    Push-Location "services/$svc"
    go test ./...
    if ($LASTEXITCODE -ne 0) { Pop-Location; exit 1 }
    Pop-Location
    Log "$svc tests: PASS"
}

foreach ($ep in @(
    @{ Name = "waf"; Url = "http://127.0.0.1:8093/healthz" },
    @{ Name = "ngfw"; Url = "http://127.0.0.1:8094/healthz" },
    @{ Name = "dlp"; Url = "http://127.0.0.1:8095/healthz" }
)) {
    try {
        Invoke-WebRequest -Uri $ep.Url -TimeoutSec 2 | Out-Null
        Log "$($ep.Name): UP"
    } catch {
        Log "$($ep.Name) live: not running (optional: compose --profile perimeter)"
    }
}

Log "Perimeter smoke PASS — log $Log"
Write-Host "Perimeter smoke PASS" -ForegroundColor Green
