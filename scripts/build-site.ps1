#!/usr/bin/env pwsh
# Сборка артефакта публичного сайта ERA One (site/) для деплоя на отдельный хостинг.
# Использование:
#   ./scripts/build-site.ps1              # → dist/site/
#   ./scripts/build-site.ps1 D:\publish\era-one

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Out = if ($args.Count -gt 0) { $args[0] } else { Join-Path $Root "dist\site" }

Write-Host "==> build portal pricing SSOT" -ForegroundColor Cyan
python (Join-Path $Root "scripts\build_portal.py")

Write-Host "==> calculator golden tests" -ForegroundColor Cyan
node (Join-Path $Root "site\test\calculator.test.js")

Write-Host "==> copy site/ -> $Out" -ForegroundColor Cyan
if (Test-Path $Out) { Remove-Item -Recurse -Force $Out }
New-Item -ItemType Directory -Path $Out -Force | Out-Null
Copy-Item -Path (Join-Path $Root "site\*") -Destination $Out -Recurse -Force
Remove-Item -Recurse -Force (Join-Path $Out "test") -ErrorAction SilentlyContinue

$count = (Get-ChildItem $Out -Recurse -File).Count
Write-Host "OK: site artifact at $Out ($count files)" -ForegroundColor Green
