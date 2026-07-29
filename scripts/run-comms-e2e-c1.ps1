#!/usr/bin/env pwsh
# ERA Communications — Wave C-1 E2E (G2). Refs: MVP-Comms-Mail-Sprint-1-Spec.md §4
param(
    [switch]$AllowSkipCH
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$logPath = Join-Path $Root "reports/comms-stage-C-1-e2e.log"
$reportDir = Split-Path $logPath -Parent
if (-not (Test-Path $reportDir)) { New-Item -ItemType Directory -Path $reportDir | Out-Null }

function Log-Line {
    param([string]$Line)
    Write-Host $Line
    Add-Content -Path $logPath -Value $Line
}

Log-Line "==> C-1 E2E $ts"

Log-Line "--> docker compose clickhouse"
$chReady = $false
try {
    docker compose -f deploy/docker-compose.dev.yml up -d clickhouse 2>&1 | ForEach-Object { Log-Line $_ }
    for ($i = 0; $i -lt 30; $i++) {
        try {
            $tcp = New-Object System.Net.Sockets.TcpClient
            $tcp.Connect("127.0.0.1", 9000)
            $tcp.Close()
            $chReady = $true
            break
        } catch {
            Start-Sleep -Seconds 2
        }
    }
} catch {
    Log-Line "WARN: docker unavailable: $_"
}
if (-not $chReady) {
    if ($AllowSkipCH) {
        Log-Line "SKIP: ClickHouse (AllowSkipCH)"
    } else {
        Log-Line "FAIL: ClickHouse not ready"
        exit 1
    }
} else {
    Log-Line "PASS: ClickHouse ready"
    $env:ERA_CH_ADDR = "127.0.0.1:9000"
}

Log-Line "--> stage gate C-1"
if ($AllowSkipCH) {
    & "$Root/scripts/run-comms-stage-gate.ps1" -Stage C-1 -AllowSkipCH 2>&1 | ForEach-Object { Log-Line $_ }
} else {
    & "$Root/scripts/run-comms-stage-gate.ps1" -Stage C-1 2>&1 | ForEach-Object { Log-Line $_ }
}
if ($LASTEXITCODE -ne 0) { exit 1 }

if ($chReady) {
    Log-Line "--> integration audit path"
    go test ./services/comms/mail/internal/audit/... -tags integration -count=1 2>&1 | ForEach-Object { Log-Line $_ }
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

Log-Line "C-1 E2E PASS"
exit 0
