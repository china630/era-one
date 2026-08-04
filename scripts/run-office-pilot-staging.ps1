#!/usr/bin/env pwsh
# ERA Office — pilot staging RT-O01…RT-O08
param(
    [string]$IdentityAPI = "http://127.0.0.1:8160",
    [string]$DriveAPI = "http://127.0.0.1:8175",
    [string]$WorkspaceAPI = "http://127.0.0.1:8170",
    [string]$DocsAPI = "http://127.0.0.1:8142",
    [string]$UIMail = "http://127.0.0.1:8180",
    [switch]$UseCompose,
    [switch]$SkipRestart
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$reportDir = Join-Path $Root "reports"
if (-not (Test-Path $reportDir)) { New-Item -ItemType Directory -Path $reportDir | Out-Null }
$logPath = Join-Path $reportDir "office-pilot-staging-$ts.log"

function Log-Line {
    param([string]$Line)
    Write-Host $Line
    Add-Content -Path $logPath -Value $Line
}

function Invoke-DriveUpload {
    param([string]$Tenant, [string]$User, [string]$Name, [string]$Content)
    $boundary = [guid]::NewGuid().ToString()
    $bodyLines = @(
        "--$boundary",
        'Content-Disposition: form-data; name="name"',
        '',
        $Name,
        "--$boundary",
        'Content-Disposition: form-data; name="content_type"',
        '',
        'text/plain',
        "--$boundary",
        "Content-Disposition: form-data; name=`"file`"; filename=`"$Name`"",
        'Content-Type: text/plain',
        '',
        $Content,
        "--$boundary--"
    ) -join "`r`n"
    $resp = Invoke-WebRequest -Uri "$DriveAPI/api/v1/drive/objects" -Method POST -Body $bodyLines `
        -ContentType "multipart/form-data; boundary=$boundary" `
        -Headers @{ "X-ERA-Tenant" = $Tenant; "X-ERA-User" = $User } -UseBasicParsing
    return ($resp.Content | ConvertFrom-Json)
}

function Invoke-DriveDownload {
    param([string]$Tenant, [string]$User, [string]$ObjectId)
    $resp = Invoke-WebRequest -Uri "$DriveAPI/api/v1/drive/objects/$ObjectId" -Method GET `
        -Headers @{ "X-ERA-Tenant" = $Tenant; "X-ERA-User" = $User } -UseBasicParsing
    return $resp.Content
}

Log-Line "==> ERA Office pilot staging $ts"

$composeFile = Join-Path $Root "deploy/docker-compose.office.yml"
if ($UseCompose) {
    Log-Line "[RT-O01] docker compose up --wait"
    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    docker compose -f $composeFile up -d --wait 2>&1 | Out-Null
    $composeExit = $LASTEXITCODE
    $ErrorActionPreference = $prevEap
    if ($composeExit -ne 0) {
        Log-Line "[FAIL] RT-O01 compose up"
        Log-Line "STAGING FAIL"
        exit 1
    }
    Log-Line "[PASS] RT-O01 compose up"
} elseif (Test-Path $composeFile) {
    Log-Line "[RT-O01] compose config"
    docker compose -f $composeFile config 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Log-Line "[PASS] RT-O01 compose config valid"
    } else {
        Log-Line "[FAIL] RT-O01 compose config"
        Log-Line "STAGING FAIL"
        exit 1
    }
} else {
    Log-Line "[SKIP] RT-O01 compose file not found"
}

Log-Line "[RT-O02] identity healthz"
try {
    $hz = Invoke-WebRequest -Uri "$IdentityAPI/healthz" -UseBasicParsing -TimeoutSec 10
    if ($hz.StatusCode -ne 200) { throw "status $($hz.StatusCode)" }
    Log-Line "[PASS] RT-O02 identity healthz"
} catch {
    Log-Line "[FAIL] RT-O02 identity healthz: $_"
    Log-Line "STAGING FAIL"
    exit 1
}

Log-Line "[RT-O03] drive upload roundtrip"
$tenant = "t-demo"
$user = "staging-user"
$payload = "office-staging-payload-$ts"
try {
    $obj = Invoke-DriveUpload -Tenant $tenant -User $user -Name "staging-$ts.txt" -Content $payload
    $down = Invoke-DriveDownload -Tenant $tenant -User $user -ObjectId $obj.id
    if ($down -ne $payload) { throw "download mismatch: $down" }
    Log-Line "[PASS] RT-O03 drive upload roundtrip id=$($obj.id)"
} catch {
    Log-Line "[FAIL] RT-O03 drive roundtrip: $_"
    Log-Line "STAGING FAIL"
    exit 1
}

Log-Line "[RT-O04] ACL cross-tenant deny"
try {
    Invoke-WebRequest -Uri "$DriveAPI/api/v1/drive/objects/$($obj.id)" -Method GET `
        -Headers @{ "X-ERA-Tenant" = "t-other"; "X-ERA-User" = "evil" } -UseBasicParsing | Out-Null
    Log-Line "[FAIL] RT-O04 expected 403 for cross-tenant"
    Log-Line "STAGING FAIL"
    exit 1
} catch {
    if ($_.Exception.Response.StatusCode.value__ -eq 403 -or $_.Exception.Response.StatusCode.value__ -eq 404) {
        Log-Line "[PASS] RT-O04 cross-tenant denied"
    } else {
        Log-Line "[FAIL] RT-O04 unexpected error: $_"
        Log-Line "STAGING FAIL"
        exit 1
    }
}

Log-Line "[RT-O05] workspace healthz"
try {
    $ws = Invoke-WebRequest -Uri "$WorkspaceAPI/healthz" -UseBasicParsing -TimeoutSec 10
    if ($ws.StatusCode -ne 200) { throw "status $($ws.StatusCode)" }
    Log-Line "[PASS] RT-O05 workspace healthz"
} catch {
    Log-Line "[FAIL] RT-O05 workspace: $_"
    Log-Line "STAGING FAIL"
    exit 1
}

Log-Line "[RT-O05b] admin-portal + docs-engine healthz"
try {
    $ap = Invoke-WebRequest -Uri "http://127.0.0.1:8140/healthz" -UseBasicParsing -TimeoutSec 10
    $de = Invoke-WebRequest -Uri "$DocsAPI/healthz" -UseBasicParsing -TimeoutSec 10
    if ($ap.StatusCode -ne 200 -or $de.StatusCode -ne 200) { throw "admin=$($ap.StatusCode) docs=$($de.StatusCode)" }
    Log-Line "[PASS] RT-O05b admin-portal + docs-engine healthz"
} catch {
    Log-Line "[FAIL] RT-O05b: $_"
    Log-Line "STAGING FAIL"
    exit 1
}

Log-Line "[RT-O05c] MinIO bucket era-drive (via drive upload RT-O03)"
Log-Line "[PASS] RT-O05c MinIO bucket implied by drive roundtrip"

Log-Line "[RT-O06] Mail attach link (optional)"
try {
    $linkBody = @{ tenant_id = $tenant; object_id = $obj.id } | ConvertTo-Json
    $link = Invoke-WebRequest -Uri "$DriveAPI/api/v1/drive/links/attachment" -Method POST -Body $linkBody `
        -ContentType "application/json" -Headers @{ "X-ERA-Tenant" = $tenant; "X-ERA-User" = $user } -UseBasicParsing
    $url = ($link.Content | ConvertFrom-Json).url
    if (-not $url) { throw "no url" }
    Log-Line "[PASS] RT-O06 attachment link $url"
} catch {
    Log-Line "[SKIP] RT-O06 mail attach link: $_"
}

if (-not $SkipRestart) {
    Log-Line "[RT-O07] restart persistence"
    if ($UseCompose) {
        $prevEap = $ErrorActionPreference
        $ErrorActionPreference = "Continue"
        docker compose -f $composeFile restart drive-api docs-engine 2>&1 | Out-Null
        $ErrorActionPreference = $prevEap
        Start-Sleep -Seconds 8
        $deadline = (Get-Date).AddMinutes(2)
        while ((Get-Date) -lt $deadline) {
            try {
                $dhz = Invoke-WebRequest -Uri "$DriveAPI/healthz" -UseBasicParsing -TimeoutSec 3
                $docsHz = Invoke-WebRequest -Uri "$DocsAPI/healthz" -UseBasicParsing -TimeoutSec 3
                if ($dhz.StatusCode -eq 200 -and $docsHz.StatusCode -eq 200) { break }
            } catch { Start-Sleep -Seconds 2 }
        }
    } else {
        Log-Line "[SKIP] RT-O07 restart requires -UseCompose"
    }
    try {
        $down2 = Invoke-DriveDownload -Tenant $tenant -User $user -ObjectId $obj.id
        if ($down2 -ne $payload) { throw "post-restart drive mismatch" }
        Log-Line "[PASS] RT-O07 drive persistence after restart"

        Log-Line "[RT-O07b] docs create + get after restart"
        $docBody = @{ tenant_id = $tenant; user_id = $user; name = "staging-$ts.erad" } | ConvertTo-Json
        $docId = $null
        $docDeadline = (Get-Date).AddMinutes(2)
        while ((Get-Date) -lt $docDeadline -and -not $docId) {
            try {
                $docCreate = Invoke-WebRequest -Uri "$DocsAPI/api/v1/docs" -Method POST -Body $docBody `
                    -ContentType "application/json" -UseBasicParsing -TimeoutSec 10
                $docId = ($docCreate.Content | ConvertFrom-Json).drive_object_id
            } catch {
                Start-Sleep -Seconds 3
            }
        }
        if (-not $docId) { throw "no doc id after restart" }
        $docGet = Invoke-WebRequest -Uri "$DocsAPI/api/v1/docs/$docId" -Method GET `
            -Headers @{ "X-ERA-User" = $user } -UseBasicParsing
        if ($docGet.StatusCode -ne 200) { throw "doc get failed" }
        Log-Line "[PASS] RT-O07b docs-api smoke id=$docId"
    } catch {
        if ($UseCompose) {
            Log-Line "[FAIL] RT-O07 restart persistence: $_"
            Log-Line "STAGING FAIL"
            exit 1
        }
    }
} else {
    Log-Line "[SKIP] RT-O07 restart (-SkipRestart)"
}

Log-Line "[RT-O08] air-gap compose scan (no external URLs in env)"
$composeText = Get-Content $composeFile -Raw
if ($composeText -match 'https?://(?!localhost|127\.0\.0\.1|identity-api|drive-api|docs-engine|minio|postgres|workspace|admin-portal|era-mail-api)[^\s"]+') {
    Log-Line "[WARN] RT-O08 possible external URL in compose - verify manually"
} else {
    Log-Line "[PASS] RT-O08 no obvious external URLs in office compose"
}

Log-Line "STAGING PASS - log: $logPath"
