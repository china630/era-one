# ERA One - Acceptance consistency gate (canon v1.2)
# Refs: docs/products/ERA-Product-Acceptance-Standard.md

[CmdletBinding()]
param(
    [string]$RepoRoot = ""
)

$ErrorActionPreference = "Stop"

if (-not $RepoRoot) {
    $here = Split-Path -Parent $MyInvocation.MyCommand.Path
    $RepoRoot = (Resolve-Path (Join-Path $here "..")).Path
}

$failures = New-Object System.Collections.Generic.List[string]

function Add-Fail {
    param([string]$Message)
    [void]$failures.Add($Message)
    Write-Host ("FAIL: " + $Message) -ForegroundColor Red
}

# Use \u2705 for checkmark to avoid file-encoding issues on Windows PowerShell 5.1
$check = [char]0x2705
$banned = @(
    @{ Name = "all-checkmark-bold";   Pattern = ("Scaffold AC \*\*all " + $check + "\*\*") },
    @{ Name = "matrix-all-checkmark"; Pattern = ("Matrix \*\*all " + $check + "\*\*") },
    @{ Name = "all-scaffold-green";   Pattern = ("all Scaffold " + $check + "|Scaffold AC all green|PRD AC all " + $check) },
    @{ Name = "ga-partner";           Pattern = 'ga \(partner\)' },
    @{ Name = "ga-greenfield";        Pattern = 'ga \(greenfield\)' }
)

$excludeName = @(
    "Acceptance-Honesty-Audit-20260730.md",
    "ERA-Product-Acceptance-Standard.md"
)

Write-Host ("Acceptance consistency check (canon v1.3) - " + $RepoRoot)

foreach ($relRoot in @("docs", "reports")) {
    $root = Join-Path $RepoRoot $relRoot
    if (-not (Test-Path -LiteralPath $root)) { continue }
    Get-ChildItem -LiteralPath $root -Recurse -File -Filter *.md | ForEach-Object {
        if ($excludeName -contains $_.Name) { return }
        $text = [System.IO.File]::ReadAllText($_.FullName)
        $rel = $_.FullName.Substring($RepoRoot.Length).TrimStart("\", "/")
        foreach ($b in $banned) {
            if ([regex]::IsMatch($text, $b.Pattern)) {
                Add-Fail ($b.Name + " in " + $rel)
            }
        }
    }
}

function Test-YamlHasBareGa {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) { return $false }
    $lines = [System.IO.File]::ReadAllLines($Path)
    foreach ($line in $lines) {
        if ($line -match '^\s*status:\s*ga\s*$') { return $true }
    }
    return $false
}

if (Test-YamlHasBareGa (Join-Path $RepoRoot "editions-comms.yaml")) {
    Add-Fail "editions-comms.yaml has status: ga; expected mvp until RT-09"
}
if (Test-YamlHasBareGa (Join-Path $RepoRoot "editions-office.yaml")) {
    Add-Fail "editions-office.yaml has status: ga; expected mvp until RT-O09"
}

$required = @(
    "docs\products\ERA-Product-Acceptance-Standard.md",
    "docs\Control-Implementation-Matrix.md",
    "docs\Comms-Implementation-Matrix.md",
    "docs\Office-Implementation-Matrix.md",
    "docs\Control-Product-Readiness-Matrix.md",
    "docs\Comms-Product-Readiness-Matrix.md",
    "docs\Office-Product-Readiness-Matrix.md",
    ".cursor\rules\task-acceptance.mdc"
)
foreach ($r in $required) {
    if (-not (Test-Path -LiteralPath (Join-Path $RepoRoot $r))) {
        Add-Fail ("missing required SSOT: " + $r)
    }
}

if ($failures.Count -gt 0) {
    Write-Host ""
    Write-Host (($failures.Count).ToString() + " acceptance consistency failure(s).") -ForegroundColor Red
    exit 1
}

Write-Host "PASS - no banned false-green / false-ga prose; SSOT files present." -ForegroundColor Green
exit 0
