#!/usr/bin/env pwsh
# ERA One — local quality gates orchestrator
# Refs: docs/products/ERA-Product-Acceptance-Standard.md · .cursor/skills/quality-gates
param(
    [switch]$SkipAcceptance,
    [switch]$SkipLint,
    [switch]$SkipSecrets,
    [switch]$SkipVuln,
    [switch]$WithE2E,
    [switch]$WithDepGraph
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$failed = New-Object System.Collections.Generic.List[string]

function Invoke-Step {
    param([string]$Name, [scriptblock]$Block)
    Write-Host ""
    Write-Host "==> $Name" -ForegroundColor Cyan
    try {
        & $Block
        if ($LASTEXITCODE -ne 0 -and $null -ne $LASTEXITCODE) {
            [void]$failed.Add($Name)
            Write-Host "FAIL: $Name (exit $LASTEXITCODE)" -ForegroundColor Red
        } else {
            Write-Host "PASS: $Name" -ForegroundColor Green
        }
    } catch {
        [void]$failed.Add($Name)
        Write-Host "FAIL: $Name — $_" -ForegroundColor Red
    }
}

if (-not $SkipAcceptance) {
    Invoke-Step "acceptance-consistency" {
        & pwsh -NoProfile -File (Join-Path $Root "scripts\check-acceptance-consistency.ps1")
    }
}

if (-not $SkipLint) {
    Invoke-Step "go-lint (scoped)" {
        if (Get-Command golangci-lint -ErrorAction SilentlyContinue) {
            golangci-lint run ./services/platform/licensegate/... ./services/platform/httpserver/... ./services/comms/internal/httpauth/... ./services/ingest-gateway/internal/server/...
        } else {
            Write-Host "SKIP golangci-lint (not installed); go vet fallback"
            go vet ./services/platform/licensegate/... ./services/comms/internal/httpauth/...
        }
    }
    Invoke-Step "rust-clippy (era-agent-core)" {
        Push-Location (Join-Path $Root "crates\era-agent-core")
        try {
            cargo clippy --all-targets -- -A warnings 2>&1 | Out-Host
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        } finally {
            Pop-Location
        }
    }
}

if (-not $SkipSecrets) {
    Invoke-Step "secrets (gitleaks)" {
        if (Get-Command gitleaks -ErrorAction SilentlyContinue) {
            gitleaks detect --source "$Root" --config (Join-Path $Root ".gitleaks.toml") --no-git -v
        } else {
            Write-Host "SKIP gitleaks (not installed locally; CI runs gitleaks-action)"
        }
    }
}

if (-not $SkipVuln) {
    Invoke-Step "govulncheck (scoped)" {
        if (Get-Command govulncheck -ErrorAction SilentlyContinue) {
            govulncheck ./services/platform/licensegate/... ./services/comms/internal/httpauth/...
        } else {
            Write-Host "SKIP govulncheck (not installed; CI installs it)"
        }
    }
    Invoke-Step "cargo-deny advisories" {
        if (Get-Command cargo-deny -ErrorAction SilentlyContinue) {
            cargo deny check advisories
        } else {
            Write-Host "SKIP cargo-deny (not installed; CI installs it)"
        }
    }
}

if ($WithDepGraph) {
    Invoke-Step "dep-graph" {
        & pwsh -NoProfile -File (Join-Path $Root "scripts\export-dep-graph.ps1")
    }
}

if ($WithE2E) {
    Invoke-Step "office-e2e-smoke" {
        & pwsh -NoProfile -File (Join-Path $Root "scripts\run-office-e2e.ps1") -SkipPlaywright:$false
    }
}

Write-Host ""
if ($failed.Count -gt 0) {
    Write-Host ("Quality gates FAILED: " + ($failed -join ", ")) -ForegroundColor Red
    exit 1
}
Write-Host "Quality gates PASS" -ForegroundColor Green
exit 0
