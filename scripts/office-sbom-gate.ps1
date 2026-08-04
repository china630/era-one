#!/usr/bin/env pwsh
# ERA Office — zero GPL/AGPL runtime SBOM gate (MVP: Cargo metadata scan).
# LGPL / "Lesser" dual-license options are allowed (ADR-0026).
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot

function Get-DenyPatterns {
    $path = Join-Path $Root "deploy/sbom/office-deny-licenses.txt"
    $patterns = @()
    if (Test-Path $path) {
        Get-Content $path | ForEach-Object {
            $line = $_.Trim()
            if ($line -and -not $line.StartsWith('#')) {
                $patterns += $line
            }
        }
    }
    if ($patterns.Count -eq 0) {
        $patterns = @('AGPL', 'GNU Affero', 'GPL-2.0', 'GPL-3.0', 'GNU General Public License')
    }
    return $patterns
}

function Test-DeniedLicense {
    param([string]$License, [string[]]$Patterns)
    if ([string]::IsNullOrWhiteSpace($License)) { return $false }
    # Allow LGPL / Lesser GPL dual-license options.
    if ($License -match 'LGPL' -or $License -match 'Lesser') { return $false }
    if ($License -match 'AGPL' -or $License -match 'GNU Affero') { return $true }
    # Word-boundary GPL-2/3 so "LGPL-2.1" is not matched.
    if ($License -match '(?<![A-Za-z])GPL-[23]') { return $true }
    if ($License -match 'GNU General Public License') { return $true }
    foreach ($p in $Patterns) {
        if ($p -match 'LGPL|Lesser') { continue }
        if ($p -eq 'AGPL' -and $License -match 'AGPL') { return $true }
        if ($p -match 'Affero' -and $License -match 'Affero') { return $true }
        if ($p -match 'GPL-2' -and $License -match '(?<![A-Za-z])GPL-2') { return $true }
        if ($p -match 'GPL-3' -and $License -match '(?<![A-Za-z])GPL-3') { return $true }
        if ($p -match 'General Public' -and $License -match 'GNU General Public License') { return $true }
    }
    return $false
}

$fail = 0
$denyPatterns = Get-DenyPatterns
Write-Host "==> office SBOM gate (Cargo metadata licenses)"
$manifest = Join-Path $Root "services/platform/docs-engine/Cargo.toml"
$prevEap = $ErrorActionPreference
$ErrorActionPreference = "Continue"
$metaJson = & cargo metadata --format-version 1 --manifest-path $manifest 2>$null
$code = $LASTEXITCODE
$ErrorActionPreference = $prevEap
if ($code -ne 0 -or -not $metaJson) {
    Write-Host "FAIL: cargo metadata unavailable (required for AC-O5)"
    exit 1
}
$meta = $metaJson | ConvertFrom-Json
foreach ($pkg in $meta.packages) {
    if (Test-DeniedLicense -License $pkg.license -Patterns $denyPatterns) {
        Write-Host "FAIL: denied runtime license on $($pkg.name): $($pkg.license)"
        $fail = 1
    }
}

Write-Host "==> office SBOM gate (Go modules - known GPL importers)"
$goSum = Join-Path $Root "services/platform/go.sum"
if (Test-Path $goSum) {
    if (Select-String -Path $goSum -Pattern "github.com/hashicorp/go-plugin" -Quiet) {
        Write-Host "WARN: known module present (review): github.com/hashicorp/go-plugin"
    }
}

if ($fail -ne 0) {
    Write-Host "SBOM GATE FAIL"
    exit 1
}
Write-Host "SBOM GATE PASS (MVP scan)"
