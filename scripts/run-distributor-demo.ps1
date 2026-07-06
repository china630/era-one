#!/usr/bin/env pwsh
# ERA XDR live demo for distributor / partner (see docs/distributor/Demo-For-Partners.md)
param(
    [switch]$SkipCompose,
    [switch]$NoBrowser,
    [switch]$Quick,
    [int]$HealthTimeoutSec = 180
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

function Wait-Healthy($Url, $TimeoutSec, [switch]$Mtls) {
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            if ($Mtls) {
                $tlsDir = Join-Path $Root "deploy\tls"
                $env:ERA_TLS_CA = Join-Path $tlsDir "ca.crt"
                $env:ERA_TLS_CLIENT_CERT = Join-Path $tlsDir "agent.crt"
                $env:ERA_TLS_CLIENT_KEY = Join-Path $tlsDir "agent.key"
                Push-Location (Join-Path $Root "services\platform")
                $out = go run ./cmd/mtls-health $Url 2>&1 | Out-String
                Pop-Location
                $j = $out.Trim() | ConvertFrom-Json -ErrorAction SilentlyContinue
                if ($j.status -eq "ok") { return $true }
            } else {
                $r = Invoke-RestMethod -Uri $Url -TimeoutSec 4
                if ($r.status -eq "ok") { return $true }
            }
        } catch { Start-Sleep -Seconds 3 }
    }
    return $false
}

function Invoke-ERA {
    param([string]$Method, [string]$Uri, [hashtable]$Body = $null)
    $params = @{
        Method      = $Method
        Uri         = $Uri
        ContentType = "application/json"
        Headers     = @{ "X-ERA-Role" = "admin"; "X-ERA-Actor" = "demo-host" }
    }
    if ($Body) { $params.Body = ($Body | ConvertTo-Json -Compress) }
    Invoke-RestMethod @params
}

Write-Host ""
Write-Host "==> ERA XDR distributor demo prep" -ForegroundColor Cyan
Write-Host "    Script: docs/distributor/Demo-For-Partners.md" -ForegroundColor DarkGray
Write-Host ""

if (-not $SkipCompose) {
    Write-Host "Starting prod stack (first run: image build ~15-25 min)..." -ForegroundColor Yellow
    docker compose -f deploy/docker-compose.prod.yml up -d --build
    if ($LASTEXITCODE -ne 0) { throw "docker compose failed" }
}

$services = @(
    @{ Name = "control-plane"; Url = "https://127.0.0.1:8090/healthz"; Mtls = $true },
    @{ Name = "event-writer"; Url = "http://127.0.0.1:8089/healthz"; Mtls = $false },
    @{ Name = "ingest-gateway"; Url = "https://127.0.0.1:8082/healthz"; Mtls = $true },
    @{ Name = "ai-core"; Url = "http://127.0.0.1:8091/healthz"; Mtls = $false },
    @{ Name = "soar"; Url = "http://127.0.0.1:8092/healthz"; Mtls = $false }
)

Write-Host "Waiting for health (up to ${HealthTimeoutSec}s)..." -ForegroundColor Cyan
foreach ($svc in $services) {
    if (-not (Wait-Healthy $svc.Url $HealthTimeoutSec -Mtls:($svc.Mtls))) {
        throw "$($svc.Name) not healthy at $($svc.Url). Check: docker compose -f deploy/docker-compose.prod.yml ps"
    }
    Write-Host "  $($svc.Name): OK" -ForegroundColor Green
}

Write-Host ""
Write-Host "==> Demo data (assets, case)" -ForegroundColor Cyan

@(
    @{ node_id = "demo-win-01"; hostname = "WS-FIN-042"; platform = "windows"; agent_version = "0.1.0" },
    @{ node_id = "demo-linux-01"; hostname = "SRV-LNX-DB01"; platform = "linux"; agent_version = "0.1.0" },
    @{ node_id = "node-load-0000"; hostname = "WS-FIN-042"; platform = "windows"; agent_version = "0.1.0-demo" }
) | ForEach-Object {
    Invoke-ERA -Method POST -Uri "http://127.0.0.1:8090/api/v1/assets/register" -Body $_ | Out-Null
}

$caseTitle = "Suspicious PowerShell - WS-FIN-042"
$existing = Invoke-RestMethod "http://127.0.0.1:8090/api/v1/cases"
$caseList = if ($existing.cases) { @($existing.cases) } else { @($existing) }
$demoCase = $caseList | Where-Object { $_.title -eq $caseTitle } | Select-Object -First 1

if (-not $demoCase) {
    $demoCase = Invoke-ERA -Method POST -Uri "http://127.0.0.1:8090/api/v1/cases" -Body @{
        title         = $caseTitle
        node_id       = "node-load-0000"
        detection_id  = "sigma-powershell-encoded-001"
    }
    Write-Host "  Case created: $caseTitle" -ForegroundColor Green
} else {
    Write-Host "  Case exists (skip duplicate): $caseTitle" -ForegroundColor DarkGray
}

Invoke-ERA -Method PATCH -Uri "http://127.0.0.1:8090/api/v1/cases/$($demoCase.id)" -Body @{
    status   = "open"
    assignee = "Petrov A. (SOC)"
} | Out-Null

try {
    Invoke-ERA -Method POST -Uri "http://127.0.0.1:8090/api/v1/cases/$($demoCase.id)/notes" -Body @{
        body = "PowerShell encoded command on WS-FIN-042. Recommend isolation pending review."
    } | Out-Null
} catch {
    Write-Host "  WARN: case note skipped" -ForegroundColor DarkGray
}

$assetsResp = Invoke-RestMethod "http://127.0.0.1:8090/api/v1/assets"
$assetCount = if ($assetsResp.assets) { $assetsResp.assets.Count } else { $assetsResp.Count }
Write-Host "  Assets: $assetCount" -ForegroundColor Green

Write-Host ""
Write-Host "==> Event pipeline (loadgen -> ingest -> Kafka -> ClickHouse)" -ForegroundColor Cyan

$rate = if ($Quick) { 150 } else { 400 }
$duration = if ($Quick) { "12s" } else { "25s" }
Push-Location services/ingest-gateway
try {
    go run ./cmd/loadgen -addr localhost:50051 -rate $rate -duration $duration -workers 8 -agents 2
    if ($LASTEXITCODE -ne 0) { throw "loadgen exit $LASTEXITCODE" }
} finally {
    Pop-Location
}

Write-Host "Waiting for ClickHouse ingest (~10s)..." -ForegroundColor DarkGray
Start-Sleep -Seconds 10

$evResp = Invoke-RestMethod "http://127.0.0.1:8089/api/events?limit=10"
$evCount = if ($evResp.events) { $evResp.events.Count } else { 0 }
Write-Host "  Events in lake: $evCount (last 10)" -ForegroundColor $(if ($evCount -gt 0) { "Green" } else { "Yellow" })

Write-Host ""
Write-Host "==> AI Core investigate (ERA AI)" -ForegroundColor Cyan
try {
    $inv = Invoke-ERA -Method POST -Uri "http://127.0.0.1:8091/api/v1/investigate" -Body @{
        node_id      = "node-load-0000"
        tenant_id    = "tenant-load-000"
        detection_id = "demo-partner-001"
    }
    Write-Host "  Verdict: $($inv.verdict) | confidence: $($inv.confidence)" -ForegroundColor Green
    if ($inv.narrative) {
        $short = $inv.narrative
        if ($short.Length -gt 120) { $short = $short.Substring(0, 120) + "..." }
        Write-Host "  Narrative: $short" -ForegroundColor DarkGray
    }
} catch {
    Write-Host "  WARN: investigate - $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host "  (if 0 events, re-run loadgen or increase -duration)" -ForegroundColor DarkGray
}

Write-Host ""
Write-Host "==> SOAR isolate host (ERA Response)" -ForegroundColor Cyan
try {
    $iso = Invoke-ERA -Method POST -Uri "http://127.0.0.1:8092/api/v1/playbooks/isolate_host" -Body @{
        node_id = "node-load-0000"
    }
    Write-Host "  Playbook: $($iso.playbook) | status: $($iso.status)" -ForegroundColor Green
    Write-Host "  $($iso.detail)" -ForegroundColor DarkGray

    $ticket = Invoke-ERA -Method POST -Uri "http://127.0.0.1:8092/api/v1/playbooks/create_ticket" -Body @{
        title   = "INC: isolate node-load-0000 (demo)"
        case_id = "demo-case"
    }
    Write-Host "  Ticket: $($ticket.status) - $($ticket.detail)" -ForegroundColor Green
} catch {
    Write-Host "  WARN: SOAR - $($_.Exception.Message)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Demo ready. Open for presentation:" -ForegroundColor White
Write-Host "  SOC Portal:   http://127.0.0.1:8090/ui/portal/" -ForegroundColor White
Write-Host "  Events UI:    http://127.0.0.1:8089/ui/" -ForegroundColor White
Write-Host "  SOAR actions: http://127.0.0.1:8092/api/v1/actions" -ForegroundColor DarkGray
Write-Host ""
Write-Host "  Next: docs/distributor/Demo-For-Partners.md (15-20 min)" -ForegroundColor DarkGray
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

if (-not $NoBrowser) {
    Start-Process "http://127.0.0.1:8090/ui/portal/"
}
