<#
.SYNOPSIS
  Headless Solo lab smoke: unit tests for desktop + pres/projects cores (no GUI).

.EXAMPLE
  .\scripts\smoke-office-solo-lab.ps1
#>
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$LogDir = Join-Path $Root "reports"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$Log = Join-Path $LogDir "office-solo-lab-smoke-$stamp.log"

function Invoke-CargoTest([string]$Package) {
  Write-Host "==> cargo test -p $Package --lib" -ForegroundColor Cyan
  Push-Location $Root
  try {
    # Cargo writes progress to stderr; do not treat as terminating errors.
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    & cargo test -p $Package --lib *>&1 | ForEach-Object { "$_" } | Tee-Object -FilePath $Log -Append | Out-Host
    $code = $LASTEXITCODE
    $ErrorActionPreference = $prev
    if ($code -ne 0) {
      throw "FAIL: cargo test -p $Package --lib (exit $code). Log: $Log"
    }
  }
  finally {
    Pop-Location
  }
}

"=== Solo lab smoke $stamp ===" | Set-Content -Encoding UTF8 $Log
Invoke-CargoTest "era-office-desktop"
Invoke-CargoTest "era-pres-core"
Invoke-CargoTest "era-projects-core"

Write-Host "PASS - log: $Log" -ForegroundColor Green
Write-Host "GUI checklist: docs/Office-Stage-Solo-Lab-Demo.md" -ForegroundColor Cyan
exit 0
