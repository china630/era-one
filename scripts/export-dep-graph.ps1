#!/usr/bin/env pwsh
# Export machine-readable dependency graphs for agents (no LLM required).
# Output: reports/deps/
param(
    [string]$RepoRoot = ""
)

$ErrorActionPreference = "Stop"
if (-not $RepoRoot) {
    $RepoRoot = (Resolve-Path (Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) "..")).Path
}
Set-Location $RepoRoot

$out = Join-Path $RepoRoot "reports\deps"
New-Item -ItemType Directory -Force -Path $out | Out-Null

Write-Host "==> cargo metadata → reports/deps/cargo-metadata.json"
$metaPath = Join-Path $out "cargo-metadata.json"
$prev = $ErrorActionPreference
$ErrorActionPreference = "Continue"
$meta = & cargo metadata --format-version 1 2>$null
$ErrorActionPreference = $prev
if ($meta) {
    Set-Content -Path $metaPath -Value ($meta -join "`n") -Encoding utf8
} else {
    Set-Content -Path $metaPath -Value "{}" -Encoding utf8
    Write-Host "WARN: cargo metadata returned empty"
}

Write-Host "==> go list platform + httpauth → reports/deps/go-deps.txt"
$goTargets = @(
    "./services/platform/licensegate",
    "./services/comms/internal/httpauth",
    "./services/ingest-gateway"
)
$lines = New-Object System.Collections.Generic.List[string]
foreach ($t in $goTargets) {
    $rel = $t -replace '^\./', ''
    if (-not (Test-Path (Join-Path $RepoRoot $rel))) { continue }
    [void]$lines.Add("# $t")
    $fmt = '{{.ImportPath}} deps={{len .Deps}}'
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $deps = & go list -f $fmt $t 2>$null
    $ErrorActionPreference = $prev
    if ($deps) {
        [void]$lines.Add([string]$deps)
    } else {
        [void]$lines.Add("# (go list skipped or failed)")
    }
}
$lines | Out-File -Encoding utf8 (Join-Path $out "go-deps.txt")

Write-Host "PASS — dep graph written under reports/deps/"
exit 0
