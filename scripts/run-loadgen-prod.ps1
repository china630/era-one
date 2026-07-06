#!/usr/bin/env pwsh
# S8-3 / F-GA-5: loadgen ≥10k ev/s × 5 min × 3 on prod stack
param(
    [int]$Rate = 10000,
    [int]$DurationSec = 300,
    [int]$Runs = 3,
    [string]$Addr = "localhost:50051",
    [int]$MinEvPerSec = 10000,
    [int]$Agents = 100,
    [int]$Workers = 16
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$LogDir = Join-Path $Root "reports"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
$LogFile = Join-Path $LogDir "loadgen-prod.log"

$TlsDir = Join-Path $Root "deploy\tls"
$env:ERA_TLS_CA = Join-Path $TlsDir "ca.crt"
$env:ERA_TLS_CLIENT_CERT = Join-Path $TlsDir "agent.crt"
$env:ERA_TLS_CLIENT_KEY = Join-Path $TlsDir "agent.key"

function Wait-Healthy($Url, $TimeoutSec = 120, [switch]$Mtls) {
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            if ($Mtls) {
                go run ./cmd/mtls-health $Url 2>$null | Out-Null
                if ($LASTEXITCODE -eq 0) { return $true }
            } else {
                Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 3 | Out-Null
                return $true
            }
        } catch { }
        Start-Sleep -Seconds 3
    }
    return $false
}

Write-Host "==> Waiting for prod stack health" -ForegroundColor Cyan
Push-Location (Join-Path $Root "services\platform")
if (-not (Wait-Healthy "https://localhost:8090/healthz" -Mtls)) {
    Write-Host "control-plane not up - start: docker compose -f deploy/docker-compose.prod.yml up -d" -ForegroundColor Yellow
}
Pop-Location

$results = @()
Push-Location services/ingest-gateway
for ($i = 1; $i -le $Runs; $i++) {
    Write-Host "==> Run $i/$Runs : $Rate ev/s for ${DurationSec}s" -ForegroundColor Cyan
    $out = go run ./cmd/loadgen -addr $Addr -rate $Rate -duration "${DurationSec}s" -workers $Workers -agents $Agents 2>&1
    $out | Tee-Object -FilePath $LogFile -Append
    $line = $out | Select-String "ev/s="
    if ($line -match "ev/s=(\d+)") {
        $evps = [int]$Matches[1]
        $results += $evps
        if ($evps -lt $MinEvPerSec) {
            Write-Host "FAIL run $i : $evps ev/s below min $MinEvPerSec" -ForegroundColor Red
            Pop-Location
            exit 1
        }
        Write-Host "PASS run $i : $evps ev/s" -ForegroundColor Green
    } else {
        Write-Host "FAIL: could not parse loadgen output" -ForegroundColor Red
        Pop-Location
        exit 1
    }
}
Pop-Location

$summary = "loadgen-prod PASS: runs=$($results -join ',') ev/s min=$MinEvPerSec duration=${DurationSec}s runs_count=$Runs"
Add-Content -Path $LogFile -Value $summary
Write-Host $summary -ForegroundColor Green
