<#
.SYNOPSIS
  Build one ERA Office Solo Store SKU (NSIS) from the shared era-office-desktop binary.

.PARAMETER Sku
  docs | tables | presentations | projects | suite

.EXAMPLE
  .\scripts\build-office-sku.ps1 -Sku docs
#>
param(
  [Parameter(Mandatory = $true)]
  [ValidateSet("docs", "tables", "presentations", "projects", "suite")]
  [string]$Sku,

  [switch]$SkipBundle
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$TauriDir = Join-Path $Root "apps\era-office-desktop\src-tauri"
$Icons = Join-Path $TauriDir "icons"
$AssetsSrc = Join-Path $TauriDir "assets"
$Overlay = Join-Path $TauriDir "tauri.conf.sku-$Sku.json"

if (-not (Test-Path $Overlay)) {
  throw "Missing overlay: $Overlay"
}

# Placeholder SKU icons (copy base icon until branded assets exist)
$skuIconMap = @{
  docs = "sku-docs"
  tables = "sku-tables"
  presentations = "sku-pres"
  projects = "sku-projects"
  suite = $null
}
$name = $skuIconMap[$Sku]
if ($name) {
  foreach ($ext in @("png", "ico")) {
    $src = Join-Path $Icons "icon.$ext"
    $dst = Join-Path $Icons "$name.$ext"
    if ((Test-Path $src) -and -not (Test-Path $dst)) {
      Copy-Item $src $dst
    }
  }
}

$skuArg = switch ($Sku) {
  "docs" { "docs" }
  "tables" { "tables" }
  "presentations" { "presentations" }
  "projects" { "projects" }
  "suite" { "suite" }
}

Write-Host "==> Building SKU=$Sku (ERA_OFFICE_SKU=$skuArg)" -ForegroundColor Cyan
$env:ERA_OFFICE_SKU = $skuArg

Push-Location $TauriDir
try {
  if ($SkipBundle) {
    cargo build --release -p era-office-desktop
  }
  else {
    # Merge SKU overlay (productName, identifier, associations)
    cargo tauri build --config $Overlay
  }
}
finally {
  Pop-Location
}

$ReleaseDir = Join-Path $Root "target\release"
$Exe = Join-Path $ReleaseDir "era-office-desktop.exe"
if (Test-Path $Exe) {
  $destAssets = Join-Path $ReleaseDir "assets"
  Write-Host "==> Copying Solo assets next to exe: $destAssets" -ForegroundColor Cyan
  if (Test-Path $destAssets) {
    Remove-Item -Recurse -Force $destAssets
  }
  Copy-Item -Recurse $AssetsSrc $destAssets
  # Shortcut helper: write sku.cmd launcher
  $cmd = Join-Path $ReleaseDir "era-office-$Sku.cmd"
  @(
    "@echo off"
    "set ERA_OFFICE_SKU=$skuArg"
    "`"%~dp0era-office-desktop.exe`" --sku=$skuArg %*"
  ) | Set-Content -Encoding ASCII $cmd
  Write-Host "Launcher: $cmd" -ForegroundColor Green
}

$Nsis = Join-Path $Root "target\release\bundle\nsis"
if (Test-Path $Nsis) {
  Write-Host "==> NSIS artifacts:" -ForegroundColor Green
  Get-ChildItem $Nsis -Filter *.exe | ForEach-Object { Write-Host "  $($_.FullName)" }
}

Write-Host "Done SKU=$Sku" -ForegroundColor Green
