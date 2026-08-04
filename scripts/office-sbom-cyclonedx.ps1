#!/usr/bin/env pwsh
# ERA Office — generate CycloneDX-ish SBOM JSON from cargo metadata (air-gap; no Syft binary required).
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$manifest = Join-Path $Root "services/platform/docs-engine/Cargo.toml"
$outDir = Join-Path $Root "reports"
if (-not (Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir | Out-Null }
$out = Join-Path $outDir "office-docs-engine.cdx.json"
$metaJson = & cargo metadata --format-version 1 --manifest-path $manifest 2>$null
if (-not $metaJson) {
    Write-Host "FAIL: cargo metadata unavailable"
    exit 1
}
$meta = $metaJson | ConvertFrom-Json
$components = @()
foreach ($pkg in $meta.packages) {
    $components += [ordered]@{
        type = "library"
        name = $pkg.name
        version = $pkg.version
        licenses = @(@{ license = @{ id = $pkg.license } })
    }
}
$bom = [ordered]@{
    bomFormat = "CycloneDX"
    specVersion = "1.5"
    version = 1
    metadata = @{ component = @{ type = "application"; name = "era-docs-engine" } }
    components = $components
}
($bom | ConvertTo-Json -Depth 8) | Set-Content -Path $out -Encoding UTF8
Write-Host "SBOM CycloneDX written: $out ($($components.Count) components)"
if (-not (Test-Path $out)) { exit 1 }
Write-Host "O-H-3 SBOM CycloneDX PASS"
