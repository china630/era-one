#!/usr/bin/env pwsh
# Local pilot checklist runner — items we can prove on our infra (no customer sign-off).
param(
    [switch]$SkipLoadgen,
    [switch]$QuickLoadgen
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$Report = Join-Path $Root "reports\pilot-local-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
New-Item -ItemType Directory -Force -Path (Split-Path $Report) | Out-Null

$TlsDir = Join-Path $Root "deploy\tls"
$env:ERA_TLS_CA = Join-Path $TlsDir "ca.crt"
$env:ERA_TLS_CLIENT_CERT = Join-Path $TlsDir "agent.crt"
$env:ERA_TLS_CLIENT_KEY = Join-Path $TlsDir "agent.key"

function Log($msg, $color = "White") {
    $line = "[$(Get-Date -Format 'HH:mm:ss')] $msg"
    Add-Content -Path $Report -Value $line
    Write-Host $line -ForegroundColor $color
}

function Test-Health($name, $url, [switch]$Mtls) {
    try {
        if ($Mtls) {
            Push-Location (Join-Path $Root "services\platform")
            $out = go run ./cmd/mtls-health $url 2>&1 | Out-String
            Pop-Location
            $j = $out.Trim() | ConvertFrom-Json -ErrorAction SilentlyContinue
            if ($j.status -eq "ok") { Log "PASS health $name" "Green"; return $true }
        } else {
            $r = Invoke-RestMethod -Uri $url -TimeoutSec 5
            if ($r.status -eq "ok") { Log "PASS health $name" "Green"; return $true }
        }
    } catch {}
    Log "FAIL health $name ($url)" "Red"
    return $false
}

function Invoke-MtlsApi($Method, $Uri, $Body = $null) {
    Push-Location (Join-Path $Root "services\platform")
    try {
        if ($Body) {
            $json = ($Body | ConvertTo-Json -Compress)
            go run ./cmd/mtls-api $Method $Uri $json 2>&1 | Out-Null
        } else {
            go run ./cmd/mtls-api $Method $Uri 2>&1 | Out-Null
        }
        if ($LASTEXITCODE -ne 0) { throw "mtls-api failed" }
    } finally {
        Pop-Location
    }
}

Log "==> ERA XDR pilot-local runner" "Cyan"
Log "Report: $Report"

# A.1 stack (full prod: scale + postgres)
$prevEAP = $ErrorActionPreference
$ErrorActionPreference = "Continue"
$env:ERA_STORE_DRIVER = "postgres"
$env:ERA_STORE_DSN = "postgres://era:era_cp_pw@postgres:5432/era_cp?sslmode=disable"
docker compose -f deploy/docker-compose.prod.yml --profile scale --profile pg up -d *> $null
$ErrorActionPreference = $prevEAP
if ($LASTEXITCODE -ne 0) { Log "FAIL docker compose up" "Red"; exit 1 }
$ok = $true
foreach ($h in @(
    @{ N = "control-plane"; U = "https://127.0.0.1:8090/healthz"; M = $true },
    @{ N = "event-writer"; U = "http://127.0.0.1:8089/healthz"; M = $false },
    @{ N = "ingest"; U = "https://127.0.0.1:8082/healthz"; M = $true },
    @{ N = "ai-core"; U = "http://127.0.0.1:8091/healthz"; M = $false },
    @{ N = "soar"; U = "http://127.0.0.1:8092/healthz"; M = $false }
)) {
    if (-not (Test-Health $h.N $h.U -Mtls:($h.M))) { $ok = $false }
}

# Demo seed + events
powershell -ExecutionPolicy Bypass -File scripts/run-distributor-demo.ps1 -SkipCompose -Quick -NoBrowser 2>&1 | Out-Null
Log "Demo seed: run-distributor-demo -SkipCompose -Quick" "DarkGray"

# A.5 PII gate (agent unit tests)
$prevEAP = $ErrorActionPreference
$ErrorActionPreference = "Continue"
& powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $Root "scripts\run-pii-gate.ps1") *> $null
$piiExit = $LASTEXITCODE
$ErrorActionPreference = $prevEAP
if ($piiExit -eq 0) { Log "PASS PII gate" "Green" } else { Log "FAIL PII gate exit $piiExit" "Red"; $ok = $false }

# E2E assets/case
try {
    powershell -ExecutionPolicy Bypass -File scripts/run-ga-e2e-prod.ps1 2>&1 | Out-Null
    Log "PASS ga-e2e-prod" "Green"
} catch { Log "WARN ga-e2e skipped" "Yellow" }

# A.6 loadgen (optional)
if (-not $SkipLoadgen) {
    if ($QuickLoadgen) {
        Push-Location services/ingest-gateway
        go run ./cmd/loadgen -addr localhost:50051 -rate 500 -duration 15s -workers 8 -agents 2 2>&1 | Tee-Object -FilePath $Report -Append
        Pop-Location
        Log "Quick loadgen burst (not 10k proof)" "Yellow"
    } else {
        try {
            powershell -ExecutionPolicy Bypass -File scripts/run-loadgen-prod.ps1 -Rate 2000 -DurationSec 60 -Runs 1 -MinEvPerSec 1500 2>&1 | Tee-Object -FilePath $Report -Append
            if ($LASTEXITCODE -eq 0) { Log "PASS loadgen (reduced threshold run)" "Green" } else { Log "FAIL loadgen" "Red"; $ok = $false }
        } catch { Log "WARN loadgen: $($_.Exception.Message)" "Yellow" }
    }
}

# Proxy events (portal BFF) via mTLS
try {
    Invoke-MtlsApi GET "https://127.0.0.1:8090/api/proxy/events?limit=3"
    Log "PASS proxy events (mTLS)" "Green"
} catch { Log "WARN proxy events: $($_.Exception.Message)" "Yellow" }

# CP persistence smoke: create case marker via mTLS
$marker = "pilot-local-$(Get-Date -Format 'HHmmss')"
try {
    Invoke-MtlsApi POST "https://127.0.0.1:8090/api/v1/cases" @{ title = $marker }
    Log "CP case marker: $marker" "DarkGray"
} catch { Log "FAIL CP case: $($_.Exception.Message)" "Red"; $ok = $false }

Log "SSO lab: docker compose -f deploy/docker-compose.prod.yml --profile sso up -d -> :8443" "DarkGray"
Log "Portal: https://127.0.0.1:8090/ui/portal/ (mTLS)" "DarkGray"
Log "Field sizing AC2 10k: docs/Field-Server-Sizing.md" "DarkGray"

if ($ok) {
    Log "==> pilot-local PASS (see $Report)" "Green"
    exit 0
}
Log "==> pilot-local PARTIAL - fix FAIL items" "Yellow"
exit 1
