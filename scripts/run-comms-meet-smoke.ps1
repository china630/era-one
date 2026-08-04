# L-5 meet join smoke (air-gap static + room API)
param(
    [string]$MeetBase = "http://127.0.0.1:8280"
)
$ErrorActionPreference = "Stop"
$log = Join-Path $PSScriptRoot "..\reports\comms-meet-smoke.log"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null
function Log($msg) { $line = "$(Get-Date -Format o) MEET $msg"; Write-Host $line; Add-Content $log $line }

# Unit path when no HTTP server: go test
Push-Location (Join-Path $PSScriptRoot "..\ui\meet")
go test . -count=1 2>&1 | ForEach-Object { Log $_ }
$code = $LASTEXITCODE
Pop-Location
if ($code -ne 0) { throw "meet unit smoke failed" }
Log "meet smoke PASS (join UI + livekit stub)"
Write-Host "PASS — see $log"
