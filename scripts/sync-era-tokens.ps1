# Sync SSOT ui/shared-tokens → office-shell / control-shell / mail.
# Theme Matrix Phase D. Usage: pwsh -File scripts/sync-era-tokens.ps1
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$src = Join-Path $root "ui/shared-tokens"
$destinations = @(
  (Join-Path $root "ui/office-shell/web/tokens"),
  (Join-Path $root "ui/control-shell/web/tokens"),
  (Join-Path $root "ui/mail/web/tokens")
)

if (-not (Test-Path $src)) { throw "Missing source: $src" }

$files = Get-ChildItem -Path $src -Filter "*.css" | Select-Object -ExpandProperty Name
if (-not $files -or $files.Count -eq 0) { throw "No CSS in $src" }

foreach ($dst in $destinations) {
  if (-not (Test-Path $dst)) { New-Item -ItemType Directory -Path $dst -Force | Out-Null }
  foreach ($f in $files) {
    Copy-Item -Path (Join-Path $src $f) -Destination (Join-Path $dst $f) -Force
  }
  Write-Host "Synced tokens → $dst"
}

Write-Host "OK: shared-tokens synced."
