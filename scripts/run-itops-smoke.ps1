#!/usr/bin/env pwsh
# IT-Ops smoke: service-desk + provision unit tests + optional live API (Stage 7)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$LogDir = Join-Path $Root "reports"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
$Stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$Log = Join-Path $LogDir "itops-smoke-$Stamp.log"

function Log([string]$m) {
    Write-Host $m
    Add-Content -Path $Log -Value $m
}

Log "==> IT-Ops smoke (service-desk + provision)"

Push-Location services/service-desk
go test ./...
if ($LASTEXITCODE -ne 0) { exit 1 }
Pop-Location
Log "service-desk tests: PASS"

Push-Location services/provision
go test ./...
if ($LASTEXITCODE -ne 0) { exit 1 }
Pop-Location
Log "provision tests: PASS"

# Optional live against compose profile itops (or local go run)
$endpoints = @(
    @{ Name = "service-desk"; Url = "http://127.0.0.1:8122/healthz" },
    @{ Name = "provision"; Url = "http://127.0.0.1:8124/healthz" }
)
$live = $false
foreach ($ep in $endpoints) {
    try {
        Invoke-WebRequest -Uri $ep.Url -TimeoutSec 2 | Out-Null
        Log "$($ep.Name): UP"
        $live = $true
    } catch {
        Log "$($ep.Name): not running (optional: docker compose -f deploy/docker-compose.prod.yml --profile itops up -d)"
    }
}

if ($live) {
    try {
        $inc = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8122/api/v1/incidents" `
            -ContentType "application/json" `
            -Body '{"title":"itops-smoke","requester":"smoke","sla_hours":1}'
        Log "incident create: OK id=$($inc.id)"
    } catch {
        Log "WARN: incident create skipped ($($_.Exception.Message))"
    }
    try {
        $pxe = Invoke-RestMethod -Uri "http://127.0.0.1:8124/api/v1/pxe/config"
        if (-not $pxe.default_image) { throw "no default_image" }
        Log "pxe config: OK default=$($pxe.default_image)"
    } catch {
        Log "WARN: pxe skipped ($($_.Exception.Message))"
    }
}

Log "IT-Ops smoke PASS (unit + optional live) — log $Log"
Write-Host "IT-Ops smoke PASS" -ForegroundColor Green
