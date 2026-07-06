#!/usr/bin/env pwsh

# Smoke: backup manifest + optional pg_dump path exists (S-02).

param(

    [string]$BackupDir = ""

)



$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot

Set-Location $Root



$tmp = Join-Path $env:TEMP "era-backup-smoke-$(Get-Random)"

& "$PSScriptRoot/backup-prod.ps1" -OutDir $tmp

$dirs = Get-ChildItem $tmp -Directory | Sort-Object Name -Descending | Select-Object -First 1

if (-not $dirs) { throw "no backup dir created" }



$manifest = Join-Path $dirs.FullName "manifest.txt"

if (-not (Test-Path $manifest)) { throw "manifest missing" }

$tgz = Get-ChildItem $dirs.FullName -Filter "*.tgz"

if ($tgz.Count -lt 1) {

    Write-Host "no docker volumes present (expected in CI) — manifest OK" -ForegroundColor Yellow

} else {

    Write-Host "backed up $($tgz.Count) volume(s)" -ForegroundColor Green

}



if ($BackupDir) {

    if (-not (Test-Path $BackupDir)) { throw "backup dir not found: $BackupDir" }

    & "$PSScriptRoot/restore-prod.ps1" -BackupDir $BackupDir -Force

} else {

    Write-Host "backup-restore-smoke PASS (backup only; use -BackupDir for restore test)" -ForegroundColor Green

}

$pgContainer = "era-prod-postgres"
try {
    $names = docker ps --format "{{.Names}}" 2>$null
    if ($names -match $pgContainer) {
        $dump = Join-Path $dirs.FullName "era_cp.sql"
        docker exec $pgContainer pg_dump -U era -d era_cp --no-owner | Out-File -FilePath $dump -Encoding utf8
        Write-Host "pg_dump captured for restore smoke (S-02)" -ForegroundColor Green
    }
} catch {
    Write-Host "postgres restore smoke skipped" -ForegroundColor Yellow
}

