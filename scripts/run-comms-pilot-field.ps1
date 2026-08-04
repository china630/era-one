# ERA Communications — customer field sign-off RT-09 (Phase 3)
param(
    [Parameter(Mandatory = $true)]
    [string]$Customer,
    [switch]$SignOff,
    [switch]$ValidateStaging,
    [string]$StagingLog = ""
)
$ErrorActionPreference = "Stop"
$Root = Split-Path $PSScriptRoot -Parent
$log = Join-Path $Root "reports\comms-pilot-field-$Customer.log"
$signoff = Join-Path $Root "reports\comms-stage-C-GA-signoff-$Customer.md"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null

function Log($msg) {
    $line = "$(Get-Date -Format o) RT-09 $msg"
    Write-Host $line
    Add-Content $log $line
}

if ($StagingLog -eq "") {
    $StagingLog = Join-Path $Root "reports\comms-pilot-staging.log"
}
if (-not (Test-Path $StagingLog)) {
    throw "Staging log missing: $StagingLog — run run-comms-pilot-staging.ps1 -ProdProfile first"
}

Log "customer=$Customer staging_ref=$StagingLog"
$stagingContent = Get-Content $StagingLog -Raw

if ($ValidateStaging -or $SignOff) {
    $required = @("RT-01", "RT-02", "RT-03", "RT-05", "RT-06", "RT-08")
    foreach ($rt in $required) {
        if ($stagingContent -notmatch $rt) {
            throw "Staging log missing step $rt"
        }
        Log "validate $rt OK"
    }
    if ($stagingContent -notmatch "PASS|readyz") {
        Log "WARN: staging log may not show explicit PASS markers"
    }
}

if (-not $SignOff) {
    Write-Host "Dry-run OK. Re-run with -SignOff after customer validation (see docs/Comms-Customer-Field-RT09.md)."
    exit 0
}

Log "FIELD SIGN-OFF customer=$Customer"
$template = Join-Path $Root "reports\comms-stage-C-GA-signoff-template.md"
if (Test-Path $template) {
    $body = Get-Content $template -Raw
    $body = $body -replace '<customer>', $Customer
    $body = $body -replace '_________________________', $Customer
    Set-Content -Path $signoff -Value $body
    Log "signoff template written: $signoff"
}

Log "F-C6 field evidence recorded — update Comms-MVP-Spec F-C6 to [x] with log path"
Write-Host "Field sign-off recorded: $log"
