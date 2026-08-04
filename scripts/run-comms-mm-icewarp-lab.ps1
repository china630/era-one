#!/usr/bin/env pwsh
# ERA Mail Moderation - IceWarp lab evidence (MM-H-7).
# Without ERA_MM_ICEWARP_HOST: documents SKIP (CI-safe).
# With host: SMTP smoke to moderation proxy :2535 + optional upstream ping.
param(
    [string]$IceWarpHost = $env:ERA_MM_ICEWARP_HOST,
    [int]$ModerationSmtpPort = 2535,
    [string]$ReportName = "comms-mm-icewarp-lab.log"
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$reportDir = Join-Path $Root "reports"
if (-not (Test-Path $reportDir)) { New-Item -ItemType Directory -Path $reportDir | Out-Null }
$log = Join-Path $reportDir $ReportName
$ts = Get-Date -Format "yyyy-MM-dd HH:mm:ss"

function Write-Log([string]$msg) {
    Add-Content -Path $log -Value $msg
    Write-Host $msg
}

Set-Content -Path $log -Value "==> IceWarp lab $ts" -Encoding utf8

if ([string]::IsNullOrWhiteSpace($IceWarpHost)) {
    Write-Log "ERA_MM_ICEWARP_HOST unset - SKIP (unit/e2e cover SMTP path without live IceWarp)"
    Write-Log "ICEWARP LAB SKIP"
    Write-Host "IceWarp lab SKIP -> $log" -ForegroundColor Yellow
    exit 0
}

Write-Log "ERA_MM_ICEWARP_HOST=$IceWarpHost"
Write-Log "Checking moderation SMTP localhost:$ModerationSmtpPort"

try {
    $tcp = New-Object System.Net.Sockets.TcpClient
    $iar = $tcp.BeginConnect("127.0.0.1", $ModerationSmtpPort, $null, $null)
    $ok = $iar.AsyncWaitHandle.WaitOne(3000, $false)
    if (-not $ok) { throw "timeout connecting to moderation SMTP :$ModerationSmtpPort" }
    $tcp.EndConnect($iar)
    $tcp.Close()
    Write-Log "moderation SMTP :$ModerationSmtpPort reachable"
} catch {
    Write-Log "FAIL: $_"
    Write-Log "ICEWARP LAB FAIL"
    exit 1
}

$hostPart = $IceWarpHost
$portPart = 25
if ($IceWarpHost -match '^(?<h>[^:]+):(?<p>\d+)$') {
    $hostPart = $Matches['h']
    $portPart = [int]$Matches['p']
}
Write-Log "Pinging IceWarp SMTP ${hostPart}:${portPart}"
try {
    $tcp2 = New-Object System.Net.Sockets.TcpClient
    $iar2 = $tcp2.BeginConnect($hostPart, $portPart, $null, $null)
    $ok2 = $iar2.AsyncWaitHandle.WaitOne(5000, $false)
    if (-not $ok2) { throw "timeout connecting to IceWarp SMTP" }
    $tcp2.EndConnect($iar2)
    $tcp2.Close()
    Write-Log "IceWarp SMTP reachable"
} catch {
    Write-Log "FAIL IceWarp: $_"
    Write-Log "ICEWARP LAB FAIL"
    exit 1
}

Write-Log "Checklist: set ERA_MM_UPSTREAM=$IceWarpHost, approve via :8360, confirm mailbox delivery"
Write-Log "ICEWARP LAB PASS"
Write-Host "IceWarp lab PASS -> $log" -ForegroundColor Green
