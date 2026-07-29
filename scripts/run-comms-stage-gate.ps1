#!/usr/bin/env pwsh
# ERA Communications — per-wave stage gate (Refs: Comms-Sprint-Index.md, Comms-Acceptance-System.md)
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('C-1', 'C-1.1', 'C-2', 'C-3', 'C-4', 'C-5', 'C-MIG', 'C-MM', 'C-MM-H', 'C-GA')]
    [string]$Stage,

    [switch]$WriteSignoff,
    [switch]$AllowSkipCH
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$reportDir = Join-Path $Root "reports"
if (-not (Test-Path $reportDir)) { New-Item -ItemType Directory -Path $reportDir | Out-Null }

$stageSlug = $Stage -replace '\.', ''
$logPath = Join-Path $reportDir "comms-stage-$Stage-$ts.log"
$signoffPath = Join-Path $reportDir "comms-stage-$Stage-signoff.md"

function Write-Proof {
    param([string]$Id, [string]$Cmd, [string]$Result)
    $line = "[$Result] $Id :: $Cmd"
    $color = switch ($Result) {
        'PASS' { 'Green' }
        'FAIL' { 'Red' }
        'SKIP' { 'Yellow' }
        default { 'Gray' }
    }
    Write-Host $line -ForegroundColor $color
    Add-Content -Path $logPath -Value $line
    return $Result
}

function Invoke-Check {
    param([string]$Id, [string]$Cmd, [bool]$Required = $true)
    Write-Host "    $Cmd" -ForegroundColor DarkGray
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    Invoke-Expression $Cmd 2>&1 | Out-Null
    $code = $LASTEXITCODE
    $ErrorActionPreference = $prev
    if ($code -eq 0) {
        Write-Proof $Id $Cmd "PASS" | Out-Null
        return $true
    }
    if (-not $Required) {
        Write-Proof $Id $Cmd "SKIP" | Out-Null
        return $true
    }
    Write-Proof $Id $Cmd "FAIL" | Out-Null
    return $false
}

function Test-ClickHouseUp {
    param([string]$Addr = "127.0.0.1:9000")
    try {
        $tcp = New-Object System.Net.Sockets.TcpClient
        $hostPart, $portPart = $Addr -split ':', 2
        if (-not $portPart) { $portPart = "9000" }
        $tcp.Connect($hostPart, [int]$portPart)
        $tcp.Close()
        return $true
    } catch {
        return $false
    }
}

function Get-StageChecks {
    param([string]$Wave)
    switch ($Wave) {
        'C-1' {
            return @(
                @{ Id = "F-C5/cargo-mail-core"; Cmd = "cargo test -p era-mail-core --quiet"; Required = $true },
                @{ Id = "F-C1/smtp-imap-e2e"; Cmd = "cargo test -p era-mail-core --test smtp_imap_e2e --quiet"; Required = $true },
                @{ Id = "F-C5/go-mail"; Cmd = "go test ./services/comms/mail/... -count=1"; Required = $true },
                @{ Id = "F-C2/autodiscover-golden"; Cmd = "go test ./services/comms/mail/internal/autodiscover/... -count=1"; Required = $true },
                @{ Id = "F-C3/policy"; Cmd = "go test ./services/comms/mail/internal/policy/... -count=1"; Required = $true },
                @{ Id = "CM1-6/licensegate"; Cmd = "go test ./services/platform/licensegate/... -count=1"; Required = $true },
                @{ Id = "CM1-1/comms-proto"; Cmd = "go test ./gen/go/era/v1/... -count=1"; Required = $true }
            )
        }
        'C-1.1' {
            return @(
                @{ Id = "F-C11/mail-connect"; Cmd = "go test ./services/comms/mail-connect/... -count=1"; Required = $true }
            )
        }
        'C-2' {
            return @(
                @{ Id = "F-C12/caldav"; Cmd = "go test ./services/comms/calendar/... -count=1"; Required = $true },
                @{ Id = "F-C13/ews"; Cmd = "go test ./services/comms/mail/internal/ews/... -count=1"; Required = $true },
                @{ Id = "F-C13b/calendar-audit"; Cmd = "go test -tags integration ./services/comms/mail/internal/calendaraudit/... -count=1"; Required = $true }
            )
        }
        'C-3' {
            return @(
                @{ Id = "F-C14/webmail"; Cmd = "go test ./ui/mail/... -count=1"; Required = $true }
            )
        }
        'C-4' {
            return @(
                @{ Id = "F-C21/chat"; Cmd = "go test ./services/comms/chat/... -count=1"; Required = $true },
                @{ Id = "F-C22/vcs"; Cmd = "go test ./services/comms/vcs/... -count=1"; Required = $true },
                @{ Id = "F-C23/chat-vcs-audit"; Cmd = "go test -tags integration ./services/comms/auditch/... -count=1"; Required = $false }
            )
        }
        'C-5' {
            return @(
                @{ Id = "F-C31/comms-ai"; Cmd = "go test ./services/comms/ai/... -count=1"; Required = $true },
                @{ Id = "F-C32/phishing-golden"; Cmd = "go test ./services/comms/ai/... -run F_C32 -count=1"; Required = $true },
                @{ Id = "F-C33/loadgen"; Cmd = "go test ./services/comms/cmd/loadgen-mailboxes/... -count=1"; Required = $true },
                @{ Id = "F-C34/budget"; Cmd = "go test ./services/comms/ai/... -run F_C34 -count=1"; Required = $true },
                @{ Id = "F-C32/audit-ch"; Cmd = "go test -tags integration ./services/comms/auditch/... -run AIInference -count=1"; Required = $false }
            )
        }
        'C-MIG' {
            return @(
                @{ Id = "F-C16/migration"; Cmd = "go test ./services/comms/migration/... -count=1"; Required = $true }
            )
        }
        'C-MM' {
            return @(
                @{ Id = "F-MM/mail-moderation"; Cmd = "go test -C services/comms/mail-moderation ./... -count=1"; Required = $true },
                @{ Id = "CM-MM-12/licensegate"; Cmd = "go test -C services/platform ./licensegate/... -count=1"; Required = $true }
            )
        }
        'C-MM-H' {
            return @(
                @{ Id = "F-MM-H/mail-moderation"; Cmd = "go test -C services/comms/mail-moderation ./... -count=1"; Required = $true },
                @{ Id = "CM-MM-H/licensegate"; Cmd = "go test -C services/platform ./licensegate/... -count=1"; Required = $true }
            )
        }
        default { return @() }
    }
}

function Write-SignoffTemplate {
    param([string]$Wave, [string]$Path)
    $dateStr = Get-Date -Format 'yyyy-MM-dd HH:mm'
    $content = @"
# ERA Communications - Stage Gate Signoff ($Wave)

**Date:** $dateStr
**Wave:** $Wave
**Gate log:** $logPath

## G1 - Auto tests

- [ ] run-comms-stage-gate.ps1 -Stage $Wave - PASS

## G2 - E2E section 4

- [ ] Log: reports/comms-stage-$Wave-e2e.log

## G3 - Implementation Matrix

- [ ] docs/Comms-Implementation-Matrix.md updated

## G4 - Comms-MVP-Spec

- [ ] Wave $Wave -> [x]

## G5 - Editions (if applicable)

- [ ] editions-comms.yaml + licensegate test

## G6 - Signoff

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Tech lead ERA | | | |
| Product owner | | | |
| Customer (C-GA only) | | | |

**Stage $Wave accepted:** [ ] Yes / [ ] No
"@
    Set-Content -Path $Path -Value $content -Encoding UTF8
    Write-Host "Signoff template: $Path" -ForegroundColor Cyan
}

Write-Host "==> ERA Communications stage gate: $Stage" -ForegroundColor Cyan
Add-Content -Path $logPath -Value "==> stage gate $Stage $ts"

$allOk = $true

if ($Stage -eq 'C-GA') {
    foreach ($mvpWave in @('C-1', 'C-1.1', 'C-2', 'C-3')) {
        Write-Host "--> MVP prerequisite: $mvpWave" -ForegroundColor Cyan
        $checks = Get-StageChecks -Wave $mvpWave
        foreach ($c in $checks) {
            if (-not (Invoke-Check -Id $c.Id -Cmd $c.Cmd -Required $c.Required)) { $allOk = $false }
        }
    }
    $mvpSpec = Join-Path $Root "docs/Comms-MVP-Spec.md"
    if (Test-Path $mvpSpec) {
        $body = Get-Content $mvpSpec -Raw
        foreach ($w in @('C-1', 'C-1.1', 'C-2', 'C-3')) {
            if ($body -notmatch "\*\*Wave $w\*\*.*\[x\]") {
                Write-Proof "C-GA/wave-$w-closed" "Comms-MVP-Spec Wave $w -> [x]" "FAIL" | Out-Null
                $allOk = $false
            } else {
                Write-Proof "C-GA/wave-$w-closed" "Comms-MVP-Spec Wave $w -> [x]" "PASS" | Out-Null
            }
        }
    }
} else {
    $checks = Get-StageChecks -Wave $Stage
    foreach ($c in $checks) {
        if (-not (Invoke-Check -Id $c.Id -Cmd $c.Cmd -Required $c.Required)) { $allOk = $false }
    }
}

# F-C4 ClickHouse integration (C-1 and C-GA only)
if ($Stage -eq 'C-1' -or $Stage -eq 'C-GA') {
    $chAddr = if ($env:ERA_CH_ADDR) { $env:ERA_CH_ADDR } else { "127.0.0.1:9000" }
    if (Test-ClickHouseUp -Addr $chAddr) {
        $env:ERA_CH_ADDR = $chAddr
        $chCmd = "go test ./services/comms/mail/internal/audit/... -tags integration -count=1"
        if (-not (Invoke-Check -Id "F-C4/ch-audit-e2e" -Cmd $chCmd -Required $true)) { $allOk = $false }
    } else {
        if ($AllowSkipCH) {
            Write-Proof "F-C4/ch-audit-e2e" "clickhouse at $chAddr (AllowSkipCH)" "SKIP" | Out-Null
            Write-Host "    WARN: ClickHouse not reachable - F-C4 skipped (AllowSkipCH)" -ForegroundColor Yellow
        } else {
            Write-Proof "F-C4/ch-audit-e2e" "clickhouse at $chAddr required" "FAIL" | Out-Null
            $allOk = $false
        }
    }
}

if ($WriteSignoff) {
    Write-SignoffTemplate -Wave $Stage -Path $signoffPath
}

$summary = if ($allOk) { "STAGE GATE PASS ($Stage)" } else { "STAGE GATE FAIL ($Stage)" }
Write-Host ""
Write-Host $summary -ForegroundColor $(if ($allOk) { "Green" } else { "Red" })
Add-Content -Path $logPath -Value $summary
Write-Host "Log: $logPath"

if (-not $allOk) { exit 1 }
