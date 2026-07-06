#!/usr/bin/env pwsh
# Preflight для field/sizing-сервера (ERA XDR field tests).
param(
    [int]$MinVCPU = 8,
    [int]$MinRAMGiB = 16,
    [int]$MinDiskGiB = 100,
    [switch]$StrictSizing  # 16 vCPU / 32 GiB для AC2
)

$ErrorActionPreference = "Continue"
$Root = Split-Path -Parent $PSScriptRoot
$fail = 0
$warn = 0

function Ok($msg) { Write-Host "  [OK] $msg" -ForegroundColor Green }
function Warn($msg) { Write-Host "  [WARN] $msg" -ForegroundColor Yellow; $script:warn++ }
function Fail($msg) { Write-Host "  [FAIL] $msg" -ForegroundColor Red; $script:fail++ }

Write-Host "==> ERA XDR field server preflight" -ForegroundColor Cyan
Write-Host "    Docs: docs/Field-Server-Setup.md" -ForegroundColor DarkGray
Write-Host ""

if ($StrictSizing) {
    $MinVCPU = 16
    $MinRAMGiB = 32
    Write-Host "Strict sizing mode (AC2 10k target)" -ForegroundColor Yellow
}

# Docker
Write-Host "-- Docker" -ForegroundColor Cyan
try {
    $dv = docker version --format "{{.Server.Version}}" 2>$null
    if ($LASTEXITCODE -eq 0 -and $dv) { Ok "docker server $dv" } else { Fail "docker not running" }
} catch { Fail "docker not found" }

try {
    docker compose version 2>$null | Out-Null
    if ($LASTEXITCODE -eq 0) { Ok "docker compose plugin" } else { Fail "docker compose missing" }
} catch { Fail "docker compose missing" }

# CPU / RAM
Write-Host "-- Resources" -ForegroundColor Cyan
if ($IsLinux) {
    $cpus = (nproc 2>$null)
    $memKb = (Get-Content /proc/meminfo | Select-String "^MemTotal:").ToString() -replace "\D", ""
    $ramGiB = [math]::Round([int64]$memKb / 1MB, 1)
} else {
    $cpus = (Get-CimInstance Win32_ComputerSystem).NumberOfLogicalProcessors
    $ramGiB = [math]::Round((Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory / 1GB, 1)
}
if ($cpus -ge $MinVCPU) { Ok "CPU logical=$cpus (min $MinVCPU)" } else { Warn "CPU logical=$cpus < $MinVCPU" }
if ($ramGiB -ge $MinRAMGiB) { Ok "RAM ${ramGiB} GiB (min $MinRAMGiB)" } else { Warn "RAM ${ramGiB} GiB < $MinRAMGiB" }

# Disk (repo drive or /)
Write-Host "-- Disk" -ForegroundColor Cyan
if ($IsLinux) {
    $df = df -BG $Root 2>$null | Select-Object -Last 1
    if ($df -match "(\d+)G") {
        $freeGiB = [int]$Matches[1]
        if ($freeGiB -ge $MinDiskGiB) { Ok "free ~${freeGiB}G on $Root" } else { Warn "free ${freeGiB}G < ${MinDiskGiB}G" }
    }
} else {
    $drive = (Resolve-Path $Root).Drive.Name
    $vol = Get-PSDrive $drive.TrimEnd(':') -ErrorAction SilentlyContinue
    if ($vol) {
        $freeGiB = [math]::Round($vol.Free / 1GB, 0)
        if ($freeGiB -ge $MinDiskGiB) { Ok "free ~${freeGiB} GiB on ${drive}" } else { Warn "free ${freeGiB} GiB < $MinDiskGiB GiB" }
    }
}

# Ports
Write-Host "-- Ports (listen check)" -ForegroundColor Cyan
$ports = @(50051, 8082, 8089, 8090, 8091, 8092, 9092)
foreach ($p in $ports) {
    $inUse = $false
    if ($IsLinux) {
        $inUse = (ss -tln 2>$null | Select-String ":$p\s") -ne $null
    } else {
        $inUse = (Get-NetTCPConnection -LocalPort $p -ErrorAction SilentlyContinue) -ne $null
    }
    if ($inUse) { Warn "port $p already in use (conflict if era stack runs)" } else { Ok "port $p free" }
}

# TLS certs
Write-Host "-- TLS (dev)" -ForegroundColor Cyan
$tlsDir = Join-Path $Root "deploy\tls"
foreach ($f in @("ca.crt", "server.crt", "agent.crt", "agent.key")) {
    $path = Join-Path $tlsDir $f
    if (Test-Path $path) { Ok $f } else { Warn "$f missing - run scripts/gen-dev-tls.ps1" }
}

# Repo tools
Write-Host "-- Toolchain (for loadgen from host)" -ForegroundColor Cyan
if (Get-Command go -ErrorAction SilentlyContinue) { Ok "go $(go version 2>$null | ForEach-Object { $_ -replace 'go version ', '' })" } else { Warn "go not in PATH (loadgen from host needs Go 1.22+)" }

Write-Host ""
if ($fail -gt 0) {
    Write-Host "Preflight FAIL: $fail error(s), $warn warning(s)" -ForegroundColor Red
    exit 1
}
if ($warn -gt 0) {
    Write-Host "Preflight PASS with $warn warning(s) - review before AC2" -ForegroundColor Yellow
    exit 0
}
Write-Host "Preflight PASS - proceed with docs/Field-Server-Setup.md step 5" -ForegroundColor Green
exit 0
