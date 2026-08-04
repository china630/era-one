<#
.SYNOPSIS
  Verify all SKU overlays + one release build + launchers; write evidence log.
  Builds the shared exe once (-SkipBundle), then writes era-office-<sku>.cmd for each SKU.

.EXAMPLE
  .\scripts\build-office-sku-matrix.ps1
#>
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$TauriDir = Join-Path $Root "apps\era-office-desktop\src-tauri"
$LogDir = Join-Path $Root "reports"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$Log = Join-Path $LogDir "office-sku-build-$stamp.log"

$skus = @("docs", "tables", "presentations", "projects", "suite")
"=== SKU matrix $stamp ===" | Set-Content -Encoding UTF8 $Log

foreach ($sku in $skus) {
  $overlay = Join-Path $TauriDir "tauri.conf.sku-$sku.json"
  if (-not (Test-Path $overlay)) {
    throw "Missing overlay $overlay"
  }
  $json = Get-Content $overlay -Raw | ConvertFrom-Json
  "OK overlay $sku identifier=$($json.identifier) productName=$($json.productName)" |
    Tee-Object -FilePath $Log -Append | Out-Host
}

Write-Host "==> Shared release build (suite SkipBundle)" -ForegroundColor Cyan
Push-Location $Root
try {
  $prev = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  & powershell -NoProfile -File (Join-Path $PSScriptRoot "build-office-sku.ps1") -Sku suite -SkipBundle *>&1 |
    ForEach-Object { "$_" } | Tee-Object -FilePath $Log -Append | Out-Host
  $code = $LASTEXITCODE
  $ErrorActionPreference = $prev
  if ($code -ne 0) { throw "FAIL suite SkipBundle exit $code. Log: $Log" }
}
finally {
  Pop-Location
}

$ReleaseDir = Join-Path $Root "target\release"
$exe = Join-Path $ReleaseDir "era-office-desktop.exe"
$assets = Join-Path $ReleaseDir "assets"
if (-not (Test-Path $exe)) { throw "Missing $exe" }
if (-not (Test-Path $assets)) { throw "Missing $assets" }

foreach ($sku in $skus) {
  $cmd = Join-Path $ReleaseDir "era-office-$sku.cmd"
  @(
    "@echo off"
    "set ERA_OFFICE_SKU=$sku"
    "`"%~dp0era-office-desktop.exe`" --sku=$sku %*"
  ) | Set-Content -Encoding ASCII $cmd
  "Launcher $cmd" | Add-Content -Encoding UTF8 $Log
}

Write-Host "PASS - log: $Log" -ForegroundColor Green
Write-Host "All SKU overlays verified; shared exe + assets + 5 launchers." -ForegroundColor Cyan
exit 0
