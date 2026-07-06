#!/usr/bin/env pwsh
# Helm upgrade/rollback helper (S7-16).
param(
    [string]$Release = "era-one",
    [string]$Namespace = "era-one",
    [string]$Chart = "deploy/helm/era-one",
    [string]$Values = "",
    [switch]$Rollback,
    [int]$Revision = 0
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

if (-not (Get-Command helm -ErrorAction SilentlyContinue)) {
    throw "helm not found in PATH"
}

if ($Rollback) {
    if ($Revision -le 0) {
        helm rollback $Release --namespace $Namespace
    } else {
        helm rollback $Release $Revision --namespace $Namespace
    }
    exit $LASTEXITCODE
}

$args = @("upgrade", $Release, $Chart, "--install", "--namespace", $Namespace, "--create-namespace")
if ($Values -ne "") {
    $args += @("-f", $Values)
}
Write-Host "==> helm $($args -join ' ')" -ForegroundColor Cyan
helm @args
exit $LASTEXITCODE
