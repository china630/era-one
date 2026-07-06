#!/usr/bin/env pwsh

# Helm template CI gate (Stage 10a / S-01).

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot

Set-Location $Root



if (-not (Get-Command helm -ErrorAction SilentlyContinue)) {

    Write-Host "helm not installed - skip" -ForegroundColor Yellow

    exit 0

}



$out = Join-Path $env:TEMP "era-one-helm-template.yaml"

$sets = @(

    "platformServices.observe.enabled=true",

    "platformServices.pam.enabled=true",

    "platformServices.vm.enabled=true",

    "platformServices.ai-core.enabled=true",

    "platformServices.soar.enabled=true",

    "platformServices.event-writer.enabled=true",

    "minio.enabled=true",

    "postgres.enabled=true",

    "controlPlane.storeDriver=postgres"

)

$setArgs = $sets | ForEach-Object { "--set", $_ }

helm template era-one deploy/helm/era-one @setArgs | Out-File $out -Encoding utf8

if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }



$required = @(

    "ingest-gateway", "event-writer", "ai-core", "soar", "vm",

    "minio", "postgres", "observe", "pam", "8082"

)

foreach ($pat in $required) {

    if (-not (Select-String -Path $out -Pattern $pat -Quiet)) {

        Write-Error "helm template missing pattern: $pat"

    }

}

Write-Host "helm template PASS: $out" -ForegroundColor Green

