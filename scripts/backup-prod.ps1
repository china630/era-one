#!/usr/bin/env pwsh

# Backup prod volumes (S6-15) + Postgres pg_dump (P0-4/S-02).

param(

    [string]$OutDir = "backups",

    [string]$ComposeFile = "deploy/docker-compose.prod.yml",

    [switch]$PgDump

)



$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot

Set-Location $Root



$stamp = Get-Date -Format "yyyyMMdd-HHmmss"

$dest = Join-Path $OutDir "era-one-$stamp"

New-Item -ItemType Directory -Force -Path $dest | Out-Null



Write-Host "==> ERA XDR prod backup -> $dest" -ForegroundColor Cyan



$volumes = @(

    "era-one-prod_control-plane-data",

    "era-one-prod_clickhouse-prod-data",

    "era-one-prod_kafka-prod-data",

    "era-one-prod_minio-prod-data",

    "era-one-prod_service-desk-data",

    "era-one-prod_pam-data",

    "era-one-prod_postgres-prod-data"

)



foreach ($vol in $volumes) {

    $exists = docker volume ls -q --filter "name=$vol"

    if (-not $exists) {

        Write-Host "skip missing volume $vol" -ForegroundColor DarkYellow

        continue

    }

    $tar = Join-Path $dest "$vol.tgz"

    Write-Host "backing up $vol"

    docker run --rm -v "${vol}:/data:ro" -v "${dest}:/backup" alpine:3.20 `

        sh -c "cd /data && tar czf /backup/$(Split-Path $tar -Leaf) ."

}



$pgContainer = docker ps --filter "name=era-prod-postgres" --format "{{.Names}}" | Select-Object -First 1

if ($PgDump -or $pgContainer) {

    if ($pgContainer) {

        $pgUser = if ($env:ERA_PG_USER) { $env:ERA_PG_USER } else { "era" }

        $pgDb = if ($env:ERA_PG_DATABASE) { $env:ERA_PG_DATABASE } else { "era_cp" }

        $dumpPath = Join-Path $dest "postgres-era_cp.sql"

        Write-Host "pg_dump from $pgContainer -> $dumpPath"

        docker exec $pgContainer pg_dump -U $pgUser -d $pgDb --no-owner --clean | Out-File -FilePath $dumpPath -Encoding utf8

    } elseif ($PgDump) {

        Write-Warning "pg_dump requested but era-prod-postgres container not running"

    }

}



@"

ERA XDR backup

timestamp=$stamp

compose=$ComposeFile

pg_dump=$([bool]$pgContainer)

"@ | Set-Content (Join-Path $dest "manifest.txt")



Write-Host "Backup complete: $dest" -ForegroundColor Green

