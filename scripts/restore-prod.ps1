#!/usr/bin/env pwsh

# Restore prod volumes from backup-prod.ps1 archive (S6-15) + optional pg restore.

param(

    [Parameter(Mandatory = $true)]

    [string]$BackupDir,

    [switch]$Force,

    [switch]$RestorePg

)



$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot

Set-Location $Root



if (-not (Test-Path $BackupDir)) {

    throw "backup dir not found: $BackupDir"

}



Write-Host "==> ERA XDR restore from $BackupDir" -ForegroundColor Cyan

if (-not $Force) {

    $confirm = Read-Host "This overwrites Docker volumes. Type RESTORE to continue"

    if ($confirm -ne "RESTORE") { exit 1 }

}



Get-ChildItem $BackupDir -Filter "*.tgz" | ForEach-Object {

    $vol = $_.BaseName

    Write-Host "restoring $vol"

    docker volume create $vol | Out-Null

    docker run --rm -v "${vol}:/data" -v "${BackupDir}:/backup:ro" alpine:3.20 `

        sh -c "rm -rf /data/* && tar xzf /backup/$($_.Name) -C /data"

}



$pgDump = Join-Path $BackupDir "postgres-era_cp.sql"

if ($RestorePg -and (Test-Path $pgDump)) {

    $pgContainer = docker ps --filter "name=era-prod-postgres" --format "{{.Names}}" | Select-Object -First 1

    if (-not $pgContainer) {

        Write-Warning "postgres container not running; start with --profile pg first"

    } else {

        $pgUser = if ($env:ERA_PG_USER) { $env:ERA_PG_USER } else { "era" }

        $pgDb = if ($env:ERA_PG_DATABASE) { $env:ERA_PG_DATABASE } else { "era_cp" }

        Write-Host "restoring postgres from $pgDump"

        Get-Content $pgDump -Raw | docker exec -i $pgContainer psql -U $pgUser -d $pgDb

    }

}



Write-Host "Restore complete. Run: docker compose -f deploy/docker-compose.prod.yml up -d" -ForegroundColor Green

