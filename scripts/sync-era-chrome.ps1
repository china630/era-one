# Sync shared ERA chrome assets from office-shell → control-shell and mail.
# Theme Matrix Phase B/C — do not hand-edit copies under control-shell or mail.
# Usage: pwsh -File scripts/sync-era-chrome.ps1
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$src = Join-Path $root "ui/office-shell/web"
$destinations = @(
  (Join-Path $root "ui/control-shell/web"),
  (Join-Path $root "ui/mail/web")
)
$files = @("era-chrome.css", "era-chrome.js")

if (-not (Test-Path $src)) { throw "Missing source: $src" }

foreach ($dst in $destinations) {
  if (-not (Test-Path $dst)) { New-Item -ItemType Directory -Path $dst -Force | Out-Null }
  foreach ($f in $files) {
    $from = Join-Path $src $f
    $to = Join-Path $dst $f
    if (-not (Test-Path $from)) { throw "Missing $from" }
    Copy-Item -Path $from -Destination $to -Force
    Write-Host "Synced $f → $dst"
  }
}

Write-Host "OK: era-chrome synced (office-shell is SSOT)."
