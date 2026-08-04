#!/usr/bin/env pwsh
# Summer S3-4 — scale proof wrapper (quick CI vs full 60k sizing host)
param(
    [int]$Mailboxes = 1000,
    [switch]$Full60k,
    [string]$AI = "http://127.0.0.1:8096"
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
if ($Full60k) { $Mailboxes = 60000 }
$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$log = Join-Path $Root "reports\comms-scale-60k-$ts.log"
$env:ERA_COMMS_AI_DEV = "1"
$quick = @()
if (-not $Full60k) { $quick = @("-quick") }
go run -C services/comms/cmd/loadgen-mailboxes . @quick -mailboxes $Mailboxes -workers 16 -addr $AI -log $log
if ($LASTEXITCODE -ne 0) { throw "loadgen failed" }
Write-Host "PASS log $log"
