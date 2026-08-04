<#
.SYNOPSIS
  Pack portable Solo lab demo: exe + assets + 5 SKU launchers -> dist/office-solo-lab/

.PARAMETER SkipBuild
  Reuse existing target/release/era-office-desktop.exe (must exist).

.PARAMETER Zip
  Also write dist/office-solo-lab.zip

.EXAMPLE
  .\scripts\pack-office-solo-lab.ps1
  .\scripts\pack-office-solo-lab.ps1 -SkipBuild -Zip
#>
param(
  [switch]$SkipBuild,
  [switch]$Zip
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$TauriDir = Join-Path $Root "apps\era-office-desktop\src-tauri"
$AssetsSrc = Join-Path $TauriDir "assets"
$ReleaseDir = Join-Path $Root "target\release"
$Exe = Join-Path $ReleaseDir "era-office-desktop.exe"
$OutDir = Join-Path $Root "dist\office-solo-lab"

if (-not $SkipBuild) {
  Write-Host "==> cargo build --release -p era-office-desktop" -ForegroundColor Cyan
  Push-Location $Root
  try {
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    & cargo build --release -p era-office-desktop
    $code = $LASTEXITCODE
    $ErrorActionPreference = $prev
    if ($code -ne 0) { throw "cargo build failed: $code" }
  }
  finally {
    Pop-Location
  }
}

if (-not (Test-Path $Exe)) {
  throw "Missing $Exe - run without -SkipBuild or build release first"
}
if (-not (Test-Path $AssetsSrc)) {
  throw "Missing assets: $AssetsSrc"
}

if (Test-Path $OutDir) {
  Remove-Item -Recurse -Force $OutDir
}
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

Copy-Item $Exe (Join-Path $OutDir "era-office-desktop.exe")
Copy-Item -Recurse $AssetsSrc (Join-Path $OutDir "assets")

$skus = @("docs", "tables", "presentations", "projects", "suite")
foreach ($sku in $skus) {
  $cmdPath = Join-Path $OutDir "era-office-$sku.cmd"
  $lines = @(
    "@echo off"
    "set ERA_OFFICE_SKU=$sku"
    "`"%~dp0era-office-desktop.exe`" --sku=$sku %*"
  )
  $lines | Set-Content -Encoding ASCII $cmdPath
}

$readmeLines = @(
  "ERA Office Solo - lab portable pack"
  "==================================="
  "Requires: WebView2 Runtime on Windows."
  ""
  "Launchers (same exe, different --sku):"
  "  era-office-suite.cmd          Hub (Docs/Tables/Pres/Projects)"
  "  era-office-docs.cmd           Documents"
  "  era-office-tables.cmd         Tables"
  "  era-office-presentations.cmd  Presentations"
  "  era-office-projects.cmd       Projects"
  ""
  "Keep era-office-desktop.exe and assets\ in the same folder."
  ""
  "This is a LAB demo pack - not Tech Eval sign-off, not Store listing."
  "Checklist: docs/Office-Stage-Solo-Lab-Demo.md"
)
$readmeLines | Set-Content -Encoding UTF8 (Join-Path $OutDir "README-LAB.txt")

Write-Host "==> Packed: $OutDir" -ForegroundColor Green
Get-ChildItem $OutDir | ForEach-Object { Write-Host "  $($_.Name)" }

if ($Zip) {
  $zipPath = Join-Path $Root "dist\office-solo-lab.zip"
  if (Test-Path $zipPath) { Remove-Item -Force $zipPath }
  Compress-Archive -Path (Join-Path $OutDir "*") -DestinationPath $zipPath
  Write-Host "==> Zip: $zipPath" -ForegroundColor Green
}

Write-Host "Done." -ForegroundColor Green
