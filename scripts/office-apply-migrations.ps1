#!/usr/bin/env pwsh
# Apply ERA Office Postgres migrations (platform + optional comms).
param(
    [string]$DatabaseUrl = $env:ERA_OFFICE_DATABASE_URL,
    [string]$MigrationsRoot = (Join-Path $PSScriptRoot "..\deploy\postgres\migrations"),
    [switch]$Comms,
    [switch]$All = $true
)
$ErrorActionPreference = "Stop"

if (-not $DatabaseUrl) {
    throw "ERA_OFFICE_DATABASE_URL or -DatabaseUrl required"
}

function Apply-Dir([string]$dir, [string[]]$names) {
    foreach ($name in $names) {
        $path = Join-Path $dir $name
        if (-not (Test-Path $path)) {
            throw "migration not found: $path"
        }
        Write-Host "==> applying $name"
        $prev = $ErrorActionPreference
        $ErrorActionPreference = "Continue"
        psql $DatabaseUrl -v ON_ERROR_STOP=1 -f $path 2>&1 | Write-Host
        $code = $LASTEXITCODE
        $ErrorActionPreference = $prev
        if ($code -ne 0) {
            throw "migration failed: $name (exit $code)"
        }
    }
}

$platform = Join-Path $MigrationsRoot "platform"
Apply-Dir $platform @(
    "001_drive.sql",
    "002_docs_sessions.sql",
    "005_projects.sql",
    "006_projects_collab.sql",
    "007_projects_w2.sql",
    "008_drive_lock.sql"
)

if ($All -or $Comms) {
    $comms = Join-Path $MigrationsRoot "comms"
    if (Test-Path $comms) {
        $files = Get-ChildItem $comms -Filter "*.sql" | Sort-Object Name | ForEach-Object { $_.Name }
        Apply-Dir $comms $files
    }
}

Write-Host "==> migrations applied"
