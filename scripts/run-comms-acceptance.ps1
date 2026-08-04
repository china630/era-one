#!/usr/bin/env pwsh
# ERA Communications — acceptance gate (Wave C-1 Mail).
# Refs: Comms-Acceptance-System.md, Comms-MVP-Spec.md F-C*

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$reportDir = Join-Path $Root "reports"
if (-not (Test-Path $reportDir)) { New-Item -ItemType Directory -Path $reportDir | Out-Null }
$logPath = Join-Path $reportDir "comms-acceptance-$ts.log"

function Write-Proof {
    param([string]$Id, [string]$Cmd, [bool]$Ok)
    $mark = if ($Ok) { "PASS" } else { "FAIL" }
    $line = "[$mark] $Id :: $Cmd"
    Write-Host $line -ForegroundColor $(if ($Ok) { "Green" } else { "Red" })
    Add-Content -Path $logPath -Value $line
    return $Ok
}

Write-Host "==> ERA Communications acceptance gate" -ForegroundColor Cyan
Add-Content -Path $logPath -Value "==> ERA Communications acceptance $ts"

$allOk = $true

$checks = @(
    @{ Id = "F-C5/cargo-mail-core"; Cmd = "cargo test -p era-mail-core --quiet" },
    @{ Id = "F-C1/smtp-imap-e2e"; Cmd = "cargo test -p era-mail-core --test smtp_imap_e2e --quiet" },
    @{ Id = "F-C5/go-mail"; Cmd = "go test ./services/comms/mail/... -count=1" },
    @{ Id = "F-C2/autodiscover-golden"; Cmd = "go test ./services/comms/mail/internal/autodiscover/... -count=1" },
    @{ Id = "F-C3/policy"; Cmd = "go test ./services/comms/mail/internal/policy/... -count=1" },
    @{ Id = "CM1-6/licensegate"; Cmd = "go test ./services/platform/licensegate/... -count=1" },
    @{ Id = "CM1-1/comms-proto"; Cmd = "go test ./gen/go/era/v1/... -count=1" }
)

foreach ($c in $checks) {
    Write-Host "    $($c.Cmd)" -ForegroundColor DarkGray
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    Invoke-Expression $c.Cmd 2>&1 | Out-Null
    $code = $LASTEXITCODE
    $ErrorActionPreference = $prev
    $ok = ($code -eq 0)
    if (-not (Write-Proof $c.Id $c.Cmd $ok)) { $allOk = $false }
}

$summary = if ($allOk) { "ACCEPTANCE PASS" } else { "ACCEPTANCE FAIL" }
Write-Host ""
Write-Host $summary -ForegroundColor $(if ($allOk) { "Green" } else { "Red" })
Add-Content -Path $logPath -Value $summary
Write-Host "Log: $logPath"

if (-not $allOk) { exit 1 }
