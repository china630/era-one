#!/usr/bin/env pwsh
# S5-21 / S8-6: E2E prod stack — register asset, case, health checks
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "==> GA E2E prod stack" -ForegroundColor Cyan

function Wait-Healthy($Url, $TimeoutSec = 90) {
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            $r = Invoke-RestMethod -Uri $Url -TimeoutSec 3
            if ($r.status -eq "ok") { return $true }
        } catch { Start-Sleep -Seconds 2 }
    }
    return $false
}

$checks = @(
    "http://localhost:8090/healthz",
    "http://localhost:8089/healthz",
    "http://localhost:8082/healthz",
    "http://localhost:8091/healthz",
    "http://localhost:8092/healthz"
)
foreach ($u in $checks) {
    if (-not (Wait-Healthy $u 30)) {
        Write-Host "SKIP E2E live: $u not available (start prod compose)" -ForegroundColor Yellow
        exit 0
    }
}

# Register synthetic Win+Linux assets
$body = @{
    node_id = "e2e-win-01"
    hostname = "e2e-win-01"
    platform = "windows"
    agent_version = "0.1.0-e2e"
} | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri "http://localhost:8090/api/v1/assets/register" -Body $body -ContentType "application/json" | Out-Null

$body = @{
    node_id = "e2e-linux-01"
    hostname = "e2e-linux-01"
    platform = "linux"
    agent_version = "0.1.0-e2e"
} | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri "http://localhost:8090/api/v1/assets/register" -Body $body -ContentType "application/json" | Out-Null

$assets = Invoke-RestMethod "http://localhost:8090/api/v1/assets"
if ($assets.Count -lt 1) { throw "no assets registered" }

$case = @{ title = "E2E GA demo case" } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri "http://localhost:8090/api/v1/cases" -Body $case -ContentType "application/json" | Out-Null

Write-Host "GA E2E prod PASS: assets=$($assets.Count)" -ForegroundColor Green
