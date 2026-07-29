#!/usr/bin/env pwsh
# ERA Mail Moderation — Stage C-MM-H e2e (tests + IceWarp lab SKIP/PASS)
param()
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$reportDir = Join-Path $Root "reports"
if (-not (Test-Path $reportDir)) { New-Item -ItemType Directory -Path $reportDir | Out-Null }
$log = Join-Path $reportDir "comms-stage-C-MM-H-e2e.log"
$ts = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
@"
==> C-MM-H e2e $ts
go test ./services/comms/mail-moderation/... -count=1
"@ | Set-Content -Path $log -Encoding utf8

Push-Location (Join-Path $Root "services/comms/mail-moderation")
try {
    go test ./... -count=1 *>> $log
    if ($LASTEXITCODE -ne 0) { throw "go test failed" }
} finally {
    Pop-Location
}

Add-Content -Path $log -Value "--- IceWarp lab ---"
& (Join-Path $Root "scripts/run-comms-mm-icewarp-lab.ps1")
if ($LASTEXITCODE -ne 0) { throw "IceWarp lab failed" }
Get-Content (Join-Path $reportDir "comms-mm-icewarp-lab.log") | Add-Content -Path $log

Add-Content -Path $log -Value "C-MM-H E2E PASS"
Write-Host "C-MM-H E2E PASS -> $log" -ForegroundColor Green
