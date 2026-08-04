#!/usr/bin/env pwsh
# ERA Mail Moderation — Stage C-MM e2e (in-process via go test)
param()
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$reportDir = Join-Path $Root "reports"
if (-not (Test-Path $reportDir)) { New-Item -ItemType Directory -Path $reportDir | Out-Null }
$log = Join-Path $reportDir "comms-stage-C-MM-e2e.log"
$ts = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
@"
==> C-MM e2e $ts
go test ./services/comms/mail-moderation/... -count=1
"@ | Set-Content -Path $log -Encoding utf8

Push-Location (Join-Path $Root "services/comms/mail-moderation")
try {
    go test ./... -count=1 *>> $log
    if ($LASTEXITCODE -ne 0) { throw "go test failed" }
} finally {
    Pop-Location
}

Add-Content -Path $log -Value "C-MM E2E PASS"
Write-Host "C-MM E2E PASS → $log" -ForegroundColor Green
