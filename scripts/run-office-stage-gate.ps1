#!/usr/bin/env pwsh
# ERA Office — per-wave stage gate (Refs: Office-Sprint-Index.md, Office-Acceptance-System.md)
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet(
        'O-0', 'O-1', 'O-2', 'O-3', 'O-4', 'O-5', 'O-GA',
        'O-GOV', 'O-6', 'O-PILOT',
        'O1-GOV', 'O1-1', 'O1-2', 'O1-3', 'O1-4', 'O1-5', 'O1-6', 'O1-7', 'O1-8', 'O1-GA',
        'O-T-0', 'O-T-1', 'O-T-2', 'O-T-3', 'O-T-4', 'O-T-5', 'O-T-6', 'O-T-TE',
        'O-P-0', 'O-P-1', 'O-P-2', 'O-P-3', 'O-P-4',
        'O-PR-0', 'O-PR-1', 'O-PR-2', 'O-PR-3',
        'O-AI-0', 'O-AI-1', 'O-AI-2', 'O-AI-3',
        'O-H-1', 'O-H-2', 'O-H-3', 'O-H-4',
        'O-AUTH', 'O-AC', 'O-T-H', 'O-P-H', 'O-PR-H', 'O-AI-H', 'O-CLOSE',
        'O-FMT-0', 'O-FMT-1', 'O-FMT-2', 'O-FMT-3'
    )]
    [string]$Stage,

    [switch]$WriteSignoff
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$ts = Get-Date -Format "yyyyMMdd-HHmmss"
$reportDir = Join-Path $Root "reports"
if (-not (Test-Path $reportDir)) { New-Item -ItemType Directory -Path $reportDir | Out-Null }

$logPath = Join-Path $reportDir "office-stage-$Stage-$ts.log"
$signoffPath = Join-Path $reportDir "office-stage-$Stage-signoff.md"

function Write-Proof {
    param([string]$Id, [string]$Detail, [string]$Result)
    $line = "[$Result] $Id :: $Detail"
    $color = switch ($Result) {
        'PASS' { 'Green' }
        'FAIL' { 'Red' }
        'SKIP' { 'Yellow' }
        default { 'Gray' }
    }
    Write-Host $line -ForegroundColor $color
    Add-Content -Path $logPath -Value $line
}

function Test-FileExists {
    param([string]$Id, [string]$RelPath, [bool]$Required = $true)
    $full = Join-Path $Root $RelPath
    if (Test-Path $full) {
        Write-Proof $Id $RelPath "PASS"
        return $true
    }
    if (-not $Required) {
        Write-Proof $Id $RelPath "SKIP"
        return $true
    }
    Write-Proof $Id "missing: $RelPath" "FAIL"
    return $false
}

function Test-MarkdownHasTable {
    param([string]$Id, [string]$RelPath, [string]$Pattern)
    $full = Join-Path $Root $RelPath
    if (-not (Test-Path $full)) {
        Write-Proof $Id "missing file $RelPath" "FAIL"
        return $false
    }
    $body = Get-Content $full -Raw
    if ($body -match $Pattern) {
        Write-Proof $Id "$RelPath matches $Pattern" "PASS"
        return $true
    }
    Write-Proof $Id "$RelPath no match $Pattern" "FAIL"
    return $false
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
        Write-Proof $Id $Cmd "PASS"
        return $true
    }
    if (-not $Required) {
        Write-Proof $Id $Cmd "SKIP"
        return $true
    }
    Write-Proof $Id $Cmd "FAIL"
    return $false
}

function Get-O0RequiredFiles {
    return @(
        'docs/products/Office-Acceptance-System.md',
        'docs/Office-Evidence-Rules.md',
        'docs/Office-MVP-Spec.md',
        'docs/Office-Sprint-Index.md',
        'docs/Office-Implementation-Matrix.md',
        'docs/Office-Pilot-Gap-List.md',
        'docs/Office-Pilot-Readiness-Checklist.md',
        'docs/Office-Stage-O0-Spec.md',
        'docs/Office-Stage-O1-Spec.md',
        'docs/Office-Stage-O2-Spec.md',
        'docs/Office-Stage-O3-Spec.md',
        'docs/Office-Stage-O4-Spec.md',
        'docs/Office-Stage-O5-Spec.md',
        'docs/Office-Stage-OGA-Spec.md',
        'scripts/run-office-stage-gate.ps1',
        'proto/era/v1/drive.proto',
        'proto/era/v1/office.proto'
    )
}

function Get-StageChecks {
    param([string]$Wave)
    switch ($Wave) {
        'O-0' {
            return @(
                @{ Id = 'O-0/drive-service'; Cmd = 'powershell -NoProfile -Command "if (Select-String -Path proto/era/v1/drive.proto -Pattern ''service DriveService'' -Quiet) { exit 0 } else { exit 1 }"'; Required = $true },
                @{ Id = 'O-0/gen-proto-list'; Cmd = 'powershell -NoProfile -Command "if ((Select-String -Path scripts/gen-proto.ps1 -Pattern ''era/v1/drive.proto'' -Quiet) -and (Select-String -Path scripts/gen-proto.ps1 -Pattern ''era/v1/office.proto'' -Quiet)) { exit 0 } else { exit 1 }"'; Required = $true },
                @{ Id = 'O-0/golden'; Cmd = 'go test -C gen/go ./era/v1/ -run "DriveService|DriveObjectWire|EradDocumentWire" -count=1'; Required = $true }
            )
        }
        'O-1' {
            return @(
                @{ Id = 'O-1/migration'; Cmd = 'cmd /c "if exist deploy\postgres\migrations\platform\001_drive.sql (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-1/drive-api'; Cmd = 'cmd /c "if exist services\platform\cmd\drive-api\main.go (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-1/drive-tests'; Cmd = 'go test -C services/platform ./drive/... -count=1'; Required = $true },
                @{ Id = 'O-1/license'; Cmd = 'go test -C services/platform ./licensegate/... -run "PlatformDrive|OfficeDocuments|OfficeDev" -count=1'; Required = $true },
                @{ Id = 'O-1/mail-drive'; Cmd = 'go test -C ui/mail ./... -run Drive -count=1'; Required = $true },
                @{ Id = 'O-1/compose'; Cmd = 'docker compose -f deploy/docker-compose.office.yml config'; Required = $true }
            )
        }
        'O-2' {
            return @(
                @{ Id = 'O-2/identity-cmd'; Cmd = 'cmd /c "if exist services\platform\cmd\identity-api\main.go (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-2/oidc-dockerfile'; Cmd = 'cmd /c "if exist deploy\dockerfiles\Dockerfile.identity-api (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-2/oidc-tests'; Cmd = 'go test -C services/platform ./internal/oidc/... -count=1'; Required = $true },
                @{ Id = 'O-2/workspace'; Cmd = 'go test -C services/platform ./workspace/... ./cmd/workspace/... -count=1'; Required = $true },
                @{ Id = 'O-2/ui-drive'; Cmd = 'go test -C ui/drive ./... -count=1'; Required = $true },
                @{ Id = 'O-2/compose'; Cmd = 'docker compose -f deploy/docker-compose.office.yml config'; Required = $true }
            )
        }
        'O-3' {
            return @(
                @{ Id = 'O-3/dockerfile'; Cmd = 'cmd /c "if exist deploy\dockerfiles\Dockerfile.docs-engine (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-3/drive-bind'; Cmd = 'cargo test -p era-docs-engine drive_bind --quiet'; Required = $true },
                @{ Id = 'O-3/proto-roundtrip'; Cmd = 'cargo test -p era-docs-engine --test proto_roundtrip --quiet'; Required = $true },
                @{ Id = 'O-3/ui-docs'; Cmd = 'go test -C ui/docs ./... -count=1'; Required = $true },
                @{ Id = 'O-3/compose-docs'; Cmd = 'docker compose -f deploy/docker-compose.office.yml --profile docs config'; Required = $true }
            )
        }
        'O-4' {
            return @(
                @{ Id = 'O-4/ws-coedit'; Cmd = 'cargo test -p era-docs-engine --test ws_coedit --quiet'; Required = $true },
                @{ Id = 'O-4/sync'; Cmd = 'cargo test -p era-docs-engine sync --quiet'; Required = $true },
                @{ Id = 'O-4/ui-docs'; Cmd = 'go test -C ui/docs ./... -count=1'; Required = $true }
            )
        }
        'O-5' {
            return @(
                @{ Id = 'O-5/golden-docx'; Cmd = 'cargo test -p era-docs-engine --test golden_docx --quiet'; Required = $true },
                @{ Id = 'O-5/golden-corpus'; Cmd = 'cargo test -p era-docs-engine --test golden_docx_corpus --quiet'; Required = $true },
                @{ Id = 'O-5/fuzz-smoke'; Cmd = 'cargo test -p era-docs-engine fuzz_docx_smoke --quiet'; Required = $true },
                @{ Id = 'O-5/sbom'; Cmd = 'pwsh -NoProfile -File ./scripts/office-sbom-gate.ps1'; Required = $true }
            )
        }
        'O-T-0' {
            return @(
                @{ Id = 'O-T-0/spec'; Cmd = 'cmd /c "if exist docs\Office-Stage-OT0-Spec.md (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-T-0/tables-vs-excel'; Cmd = 'cmd /c "if exist docs\ERA-Tables-vs-Excel.md (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-T-0/proto'; Cmd = 'powershell -NoProfile -Command "if (Select-String -Path proto/era/v1/office.proto -Pattern ''message EratSheet'' -Quiet) { exit 0 } else { exit 1 }"'; Required = $true },
                @{ Id = 'O-T-0/golden'; Cmd = 'go test -C gen/go ./era/v1/ -run "EratSheetWire" -count=1'; Required = $true }
            )
        }
        'O-T-1' {
            return @(
                @{ Id = 'O-T-1/crate'; Cmd = 'cmd /c "if exist services\platform\tables-engine\Cargo.toml (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-T-1/drive-bind'; Cmd = 'cargo test -p era-tables-engine drive_bind --quiet'; Required = $true }
            )
        }
        'O-T-2' {
            return @(
                @{ Id = 'O-T-2/calc'; Cmd = 'cargo test -p era-tables-engine calc --quiet'; Required = $true }
            )
        }
        'O-T-3' {
            return @(
                @{ Id = 'O-T-3/xlsx'; Cmd = 'cargo test -p era-tables-engine golden_xlsx --quiet'; Required = $true },
                @{ Id = 'O-T-3/fuzz'; Cmd = 'cargo test -p era-tables-engine fuzz_xlsx --quiet'; Required = $true }
            )
        }
        'O-T-4' {
            return @(
                @{ Id = 'O-T-4/ws'; Cmd = 'cargo test -p era-tables-engine --test ws_sheet_coedit --quiet'; Required = $true },
                @{ Id = 'O-T-4/sync'; Cmd = 'cargo test -p era-tables-engine sync --quiet'; Required = $true }
            )
        }
        'O-T-5' {
            return @(
                @{ Id = 'O-T-5/ui'; Cmd = 'go test -C ui/tables ./... -count=1'; Required = $true },
                @{ Id = 'O-T-5/workspace'; Cmd = 'go test -C services/platform ./workspace/... -run Tables -count=1'; Required = $true }
            )
        }
        'O-T-6' {
            return @(
                @{ Id = 'O-T-6/license'; Cmd = 'go test -C services/platform ./licensegate/... -run "OfficeTables|OfficeDev" -count=1'; Required = $true },
                @{ Id = 'O-T-6/sbom'; Cmd = 'pwsh -NoProfile -File ./scripts/office-sbom-gate.ps1'; Required = $true }
            )
        }
        'O-T-TE' {
            return @(
                @{ Id = 'O-T-TE/calc'; Cmd = 'cargo test -p era-tables-engine calc --quiet'; Required = $true },
                @{ Id = 'O-T-TE/ws'; Cmd = 'cargo test -p era-tables-engine --test ws_sheet_coedit --quiet'; Required = $true },
                @{ Id = 'O-T-TE/ui'; Cmd = 'go test -C ui/tables ./... -count=1'; Required = $true }
            )
        }
        'O-P-0' {
            return @(
                @{ Id = 'O-P-0/prd'; Cmd = 'cmd /c "if exist docs\products\PRD-Office-P3.md (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-P-0/proto'; Cmd = 'go test -C gen/go ./era/v1/ -run "ErapDeckWire" -count=1'; Required = $true }
            )
        }
        'O-P-1' {
            return @(
                @{ Id = 'O-P-1/crate'; Cmd = 'cmd /c "if exist services\platform\presentations-engine\Cargo.toml (exit /b 0) else (exit /b 1)"'; Required = $true }
            )
        }
        'O-P-2' {
            return @(
                @{ Id = 'O-P-2/golden'; Cmd = 'cargo test -p era-presentations-engine golden_pptx --quiet'; Required = $true }
            )
        }
        'O-P-3' {
            return @(
                @{ Id = 'O-P-3/ui'; Cmd = 'go test -C ui/presentations ./... -count=1'; Required = $true }
            )
        }
        'O-P-4' {
            return @(
                @{ Id = 'O-P-4/license'; Cmd = 'go test -C services/platform ./licensegate/... -run OfficePresentations -count=1'; Required = $true }
            )
        }
        'O-PR-0' {
            return @(
                @{ Id = 'O-PR-0/prd'; Cmd = 'cmd /c "if exist docs\products\PRD-Office-Projects.md (exit /b 0) else (exit /b 1)"'; Required = $true }
            )
        }
        'O-PR-1' {
            return @(
                @{ Id = 'O-PR-1/cmd'; Cmd = 'cmd /c "if exist services\platform\cmd\docs-projects\main.go (exit /b 0) else (exit /b 1)"'; Required = $true }
            )
        }
        'O-PR-2' {
            return @(
                @{ Id = 'O-PR-2/ui'; Cmd = 'go test -C ui/projects ./... -count=1'; Required = $true }
            )
        }
        'O-PR-3' {
            return @(
                @{ Id = 'O-PR-3/license'; Cmd = 'go test -C services/platform ./licensegate/... -run OfficeProjects -count=1'; Required = $true }
            )
        }
        'O-AI-0' {
            return @(
                @{ Id = 'O-AI-0/prd'; Cmd = 'cmd /c "if exist docs\products\PRD-Office-AI.md (exit /b 0) else (exit /b 1)"'; Required = $true }
            )
        }
        'O-AI-1' {
            return @(
                @{ Id = 'O-AI-1/cmd'; Cmd = 'cmd /c "if exist services\platform\cmd\docs-ai\main.go (exit /b 0) else (exit /b 1)"'; Required = $true }
            )
        }
        'O-AI-2' {
            return @(
                @{ Id = 'O-AI-2/license'; Cmd = 'go test -C services/platform ./licensegate/... -run OfficeAI -count=1'; Required = $true }
            )
        }
        'O-AI-3' {
            return @(
                @{ Id = 'O-AI-3/sbom'; Cmd = 'pwsh -NoProfile -File ./scripts/office-sbom-gate.ps1'; Required = $true }
            )
        }
        'O-FMT-0' {
            return @(
                @{ Id = 'O-FMT-0/catalog'; Cmd = 'cmd /c "if exist docs\Office-UI-Controls-Catalog.md (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-FMT-0/spec0'; Cmd = 'cmd /c "if exist docs\Office-Stage-OFMT0-Spec.md (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-FMT-0/spec1'; Cmd = 'cmd /c "if exist docs\Office-Stage-OFMT1-Spec.md (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-FMT-0/spec2'; Cmd = 'cmd /c "if exist docs\Office-Stage-OFMT2-Spec.md (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-FMT-0/spec3'; Cmd = 'cmd /c "if exist docs\Office-Stage-OFMT3-Spec.md (exit /b 0) else (exit /b 1)"'; Required = $true },
                @{ Id = 'O-FMT-0/inventory-ids'; Cmd = 'powershell -NoProfile -Command "if ((Select-String -Path docs/Office-UI-Feature-Inventory.md -Pattern ''DOC-FORMAT-PAINTER'' -Quiet) -and (Select-String -Path docs/Office-UI-Feature-Inventory.md -Pattern ''TBL-AVG-MIN-MAX-ROUND'' -Quiet) -and (Select-String -Path docs/Office-UI-Feature-Inventory.md -Pattern ''PRE-DUP-SLIDE'' -Quiet)) { exit 0 } else { exit 1 }"'; Required = $true },
                @{ Id = 'O-FMT-0/sprint-index'; Cmd = 'powershell -NoProfile -Command "if (Select-String -Path docs/Office-Sprint-Index.md -Pattern ''O-FMT-0'' -Quiet) { exit 0 } else { exit 1 }"'; Required = $true }
            )
        }
        'O-FMT-1' {
            return @(
                @{ Id = 'O-FMT-1/model'; Cmd = 'powershell -NoProfile -Command "if ((Select-String -Path services/platform/docs-engine/src/model.rs -Pattern ''list_level'' -Quiet) -and (Select-String -Path services/platform/docs-engine/src/model.rs -Pattern ''superscript'' -Quiet)) { exit 0 } else { exit 1 }"'; Required = $true },
                @{ Id = 'O-FMT-1/engine'; Cmd = 'cargo test -p era-docs-engine fmt_enrichment --quiet'; Required = $true },
                @{ Id = 'O-FMT-1/ui'; Cmd = 'go test -C ui/docs ./... -count=1'; Required = $true },
                @{ Id = 'O-FMT-1/inventory'; Cmd = 'powershell -NoProfile -Command "if (Select-String -Path docs/Office-UI-Feature-Inventory.md -Pattern ''DOC-FORMAT-PAINTER.*✅'' -Quiet) { exit 0 } else { exit 1 }"'; Required = $true }
            )
        }
        'O-FMT-2' {
            return @(
                @{ Id = 'O-FMT-2/calc'; Cmd = 'cargo test -p era-tables-engine avg_min_max_round --quiet'; Required = $true },
                @{ Id = 'O-FMT-2/ui-tables'; Cmd = 'go test -C ui/tables ./... -count=1'; Required = $true },
                @{ Id = 'O-FMT-2/ui-docs'; Cmd = 'go test -C ui/docs ./... -count=1'; Required = $true },
                @{ Id = 'O-FMT-2/inventory'; Cmd = 'powershell -NoProfile -Command "if ((Select-String -Path docs/Office-UI-Feature-Inventory.md -Pattern ''TBL-AVG-MIN-MAX-ROUND.*✅'' -Quiet) -and (Select-String -Path docs/Office-UI-Feature-Inventory.md -Pattern ''DOC-RULER.*✅'' -Quiet)) { exit 0 } else { exit 1 }"'; Required = $true }
            )
        }
        'O-FMT-3' {
            return @(
                @{ Id = 'O-FMT-3/ui'; Cmd = 'go test -C ui/presentations ./... -count=1'; Required = $true },
                @{ Id = 'O-FMT-3/inventory'; Cmd = 'powershell -NoProfile -Command "if (Select-String -Path docs/Office-UI-Feature-Inventory.md -Pattern ''PRE-DUP-SLIDE.*✅'' -Quiet) { exit 0 } else { exit 1 }"'; Required = $true }
            )
        }
        'O-H-1' {
            return @(
                @{ Id = 'O-H-1/adr-note'; Cmd = 'powershell -NoProfile -Command "if (Select-String -Path docs/adr/0026-sovereign-office-engine.md -Pattern ''OpLog-authoritative'' -Quiet) { exit 0 } else { exit 1 }"'; Required = $true }
            )
        }
        'O-H-2' {
            return @(
                @{ Id = 'O-H-2/docx'; Cmd = 'cargo test -p era-docs-engine --test golden_docx_corpus --quiet'; Required = $true }
            )
        }
        'O-H-3' {
            return @(
                @{ Id = 'O-H-3/syft'; Cmd = 'pwsh -NoProfile -File ./scripts/office-sbom-cyclonedx.ps1'; Required = $true }
            )
        }
        'O-H-4' {
            return @(
                @{ Id = 'O-H-4/e2e-files'; Cmd = 'cmd /c "if exist ui\office\e2e\docs-smoke.spec.ts (exit /b 0) else (exit /b 1)"'; Required = $true }
            )
        }
        'O-AUTH' {
            return @(
                @{ Id = 'O-AUTH/drive-spoof'; Cmd = 'go test -C services/platform ./drive/api/... -run "Spoof|JWTRequired|ServiceToken|LicenseDenied" -count=1'; Required = $true },
                @{ Id = 'O-AUTH/docs-auth'; Cmd = 'cargo test -p era-docs-engine create_without --quiet'; Required = $true },
                @{ Id = 'O-AUTH/tables-auth'; Cmd = 'cargo test -p era-tables-engine tables_create_without --quiet'; Required = $true },
                @{ Id = 'O-AUTH/workspace'; Cmd = 'go test -C services/platform ./workspace/... -run APIAuth -count=1'; Required = $true }
            )
        }
        'O-AC' {
            return @(
                @{ Id = 'O-AC/ws-coedit'; Cmd = 'cargo test -p era-docs-engine --test ws_coedit --quiet'; Required = $true },
                @{ Id = 'O-AC/mail-docs'; Cmd = 'go test -C ui/mail ./... -run "Documents|EditLink|VerifyIntent" -count=1'; Required = $true },
                @{ Id = 'O-AC/spec'; Cmd = 'cmd /c "if exist docs\Office-Stage-O-AUTH-Spec.md (exit /b 0) else (exit /b 1)"'; Required = $true }
            )
        }
        'O-T-H' {
            return @(
                @{ Id = 'O-T-H/ws'; Cmd = 'cargo test -p era-tables-engine --test ws_sheet_coedit --quiet'; Required = $true },
                @{ Id = 'O-T-H/license-http'; Cmd = 'cargo test -p era-tables-engine tables_create_without_license --quiet'; Required = $true },
                @{ Id = 'O-T-H/compose-engines'; Cmd = 'docker compose -f deploy/docker-compose.office.yml --profile office-engines config'; Required = $false },
                @{ Id = 'O-T-H/matrix'; Cmd = 'powershell -NoProfile -Command "if (Select-String -Path docs/Office-Implementation-Matrix.md -Pattern ''AC-T1'' -Quiet) { exit 0 } else { exit 1 }"'; Required = $true }
            )
        }
        'O-P-H' {
            return @(
                @{ Id = 'O-P-H/golden'; Cmd = 'cargo test -p era-presentations-engine --quiet'; Required = $true },
                @{ Id = 'O-P-H/compose-engines'; Cmd = 'docker compose -f deploy/docker-compose.office.yml --profile office-engines config'; Required = $false },
                @{ Id = 'O-P-H/matrix'; Cmd = 'powershell -NoProfile -Command "if (Select-String -Path docs/Office-Implementation-Matrix.md -Pattern ''AC-P1'' -Quiet) { exit 0 } else { exit 1 }"'; Required = $true }
            )
        }
        'O-PR-H' {
            return @(
                @{ Id = 'O-PR-H/cmd'; Cmd = 'go test -C services/platform ./cmd/docs-projects/... -count=1'; Required = $true },
                @{ Id = 'O-PR-H/matrix'; Cmd = 'powershell -NoProfile -Command "if (Select-String -Path docs/Office-Implementation-Matrix.md -Pattern ''AC-PR1'' -Quiet) { exit 0 } else { exit 1 }"'; Required = $true }
            )
        }
        'O-AI-H' {
            return @(
                @{ Id = 'O-AI-H/cmd'; Cmd = 'go test -C services/platform ./cmd/docs-ai/... -count=1'; Required = $true },
                @{ Id = 'O-AI-H/matrix'; Cmd = 'powershell -NoProfile -Command "if (Select-String -Path docs/Office-Implementation-Matrix.md -Pattern ''AC-AI1'' -Quiet) { exit 0 } else { exit 1 }"'; Required = $true }
            )
        }
        'O-CLOSE' {
            return @(
                @{ Id = 'O-CLOSE/matrix'; Cmd = 'powershell -NoProfile -Command "if (Select-String -Path docs/Office-Implementation-Matrix.md -Pattern ''all Scaffold'' -Quiet) { exit 0 } else { exit 1 }"'; Required = $true },
                @{ Id = 'O-CLOSE/drive-auth'; Cmd = 'go test -C services/platform ./drive/api/... -run "Spoof|JWTRequired" -count=1'; Required = $true },
                @{ Id = 'O-CLOSE/docs-ai'; Cmd = 'go test -C services/platform ./cmd/docs-ai/... -count=1'; Required = $true }
            )
        }
        # Legacy aliases (stash fine-grained)
        'O-GOV' { return @() }
        'O-6' {
            return @(
                @{ Id = 'O-6/compose'; Cmd = 'docker compose -f deploy/docker-compose.office.yml config'; Required = $true }
            )
        }
        'O1-1' {
            return @(
                @{ Id = 'O1-1/office-proto'; Cmd = 'cmd /c "if exist proto\era\v1\office.proto (exit /b 0) else (exit /b 1)"'; Required = $true }
            )
        }
        'O1-2' {
            return @(
                @{ Id = 'O1-2/docx-golden'; Cmd = 'cargo test -p era-docs-engine --test golden_docx --quiet'; Required = $false }
            )
        }
        'O1-3' {
            return @(
                @{ Id = 'O1-3/sync'; Cmd = 'cargo test -p era-docs-engine sync --quiet'; Required = $false }
            )
        }
        'O1-4' {
            return @(
                @{ Id = 'O1-4/drive-bind'; Cmd = 'cargo test -p era-docs-engine drive_bind --quiet'; Required = $false }
            )
        }
        'O1-5' {
            return @(
                @{ Id = 'O1-5/engine'; Cmd = 'cargo test -p era-docs-engine --quiet'; Required = $false }
            )
        }
        'O1-6' {
            return @(
                @{ Id = 'O1-6/ui-docs'; Cmd = 'go test -C ui/docs ./... -count=1'; Required = $false }
            )
        }
        'O1-7' {
            return @(
                @{ Id = 'O1-7/deeplink'; Cmd = 'go test -C ui/mail ./... -run Documents -count=1'; Required = $false }
            )
        }
        'O1-8' {
            return @(
                @{ Id = 'O1-8/license'; Cmd = 'go test -C services/platform ./licensegate/... -run OfficeDocuments -count=1'; Required = $true }
            )
        }
        'O-PILOT' {
            return @(
                @{ Id = 'O-PILOT/sbom'; Cmd = 'powershell -NoProfile -File ./scripts/office-sbom-gate.ps1'; Required = $false }
            )
        }
        default { return @() }
    }
}

function Write-SignoffTemplate {
    param([string]$Wave, [string]$Path)
    $dateStr = Get-Date -Format 'yyyy-MM-dd HH:mm'
    $customerLine = if ($Wave -eq 'O-GA' -or $Wave -eq 'O1-GA') {
        '| Customer (field RT-O09) | N/A — field deferred | | |'
    } else {
        '| Customer (O-GA / O1-GA) | | | |'
    }
    $acceptedHint = if ($Wave -eq 'O-GA') {
        "**Stage O-GA (honesty) accepted:** [ ] Yes / [ ] No — customer field = RT-O09 open"
    } else {
        "**Stage $Wave accepted:** [ ] Yes / [ ] No"
    }
    $content = @"
# ERA Office - Stage Gate Signoff ($Wave)

**Date:** $dateStr
**Wave:** $Wave
**Gate log:** $logPath

## G1 - Auto tests / O-GOV file checks

- [ ] run-office-stage-gate.ps1 -Stage $Wave - PASS

## G2 - E2E section 4

- [ ] Log: reports/office-stage-$Wave-e2e.log

## G3 - Implementation Matrix

- [ ] docs/Office-Implementation-Matrix.md updated

## G4 - Office-MVP-Spec

- [ ] Wave $Wave -> [x] (if applicable)

## G5 - Editions (if applicable)

- [ ] editions-shared.yaml / editions-office.yaml

## G6 - Signoff

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Tech lead ERA | | | |
| Product owner | | | |
$customerLine

$acceptedHint
"@
    Set-Content -Path $Path -Value $content -Encoding UTF8
    Write-Host "Signoff template: $Path" -ForegroundColor Cyan
}

Write-Host "==> ERA Office stage gate: $Stage" -ForegroundColor Cyan
Add-Content -Path $logPath -Value "==> office stage gate $Stage $ts"

$allOk = $true

if ($Stage -eq 'O-0') {
    foreach ($f in (Get-O0RequiredFiles)) {
        if (-not (Test-FileExists -Id "O-0/file" -RelPath $f)) { $allOk = $false }
    }
    if (-not (Test-MarkdownHasTable -Id 'O-0/evidence' -RelPath 'docs/Office-Evidence-Rules.md' -Pattern 'Нет лога')) { $allOk = $false }
    if (-not (Test-MarkdownHasTable -Id 'O-0/matrix' -RelPath 'docs/Office-Implementation-Matrix.md' -Pattern 'AC-O3')) { $allOk = $false }
    $checks = Get-StageChecks -Wave 'O-0'
    foreach ($c in $checks) {
        if (-not (Invoke-Check -Id $c.Id -Cmd $c.Cmd -Required $c.Required)) { $allOk = $false }
    }
} elseif ($Stage -eq 'O-GOV') {
    foreach ($f in (Get-O0RequiredFiles)) {
        if (-not (Test-FileExists -Id "O-GOV/file" -RelPath $f)) { $allOk = $false }
    }
} elseif ($Stage -eq 'O1-GOV') {
    if (-not (Test-MarkdownHasTable -Id 'O1-GOV/matrix-p1' -RelPath 'docs/Office-Implementation-Matrix.md' -Pattern 'AC-O1')) { $allOk = $false }
    if (-not (Test-FileExists -Id 'O1-GOV/prd-p1' -RelPath 'docs/products/PRD-Office-P1.md')) { $allOk = $false }
} elseif ($Stage -eq 'O-GA') {
    foreach ($f in (Get-O0RequiredFiles)) {
        if (-not (Test-FileExists -Id "O-GA/file" -RelPath $f)) { $allOk = $false }
    }
    if (-not (Test-MarkdownHasTable -Id 'O-GA/evidence' -RelPath 'docs/Office-Evidence-Rules.md' -Pattern 'Нет лога')) { $allOk = $false }
    if (-not (Test-MarkdownHasTable -Id 'O-GA/matrix' -RelPath 'docs/Office-Implementation-Matrix.md' -Pattern 'AC-O3')) { $allOk = $false }
    foreach ($w in @('O-0', 'O-1', 'O-2', 'O-3', 'O-4', 'O-5')) {
        $wavePat = [regex]::Escape('**' + $w + '**') + '.*\[x\]'
        if (-not (Test-MarkdownHasTable -Id "O-GA/mvp-$w" -RelPath 'docs/Office-MVP-Spec.md' -Pattern $wavePat)) { $allOk = $false }
        if (-not (Test-MarkdownHasTable -Id "O-GA/index-$w" -RelPath 'docs/Office-Sprint-Index.md' -Pattern $wavePat)) { $allOk = $false }
    }
    foreach ($w in @('O-0', 'O-1', 'O-2', 'O-3', 'O-4', 'O-5')) {
        $checks = Get-StageChecks -Wave $w
        foreach ($c in $checks) {
            if (-not (Invoke-Check -Id $c.Id -Cmd $c.Cmd -Required $c.Required)) { $allOk = $false }
        }
    }
} elseif ($Stage -eq 'O1-GA') {
    foreach ($w in @('O1-1', 'O1-2', 'O1-3', 'O1-4', 'O1-5', 'O1-6', 'O1-7', 'O1-8')) {
        $checks = Get-StageChecks -Wave $w
        foreach ($c in $checks) {
            if (-not (Invoke-Check -Id $c.Id -Cmd $c.Cmd -Required $c.Required)) { $allOk = $false }
        }
    }
} else {
    $checks = Get-StageChecks -Wave $Stage
    if ($checks.Count -eq 0) {
        Write-Proof "$Stage/no-checks" 'implementation pending — no auto checks yet' 'SKIP'
    } else {
        foreach ($c in $checks) {
            if (-not (Invoke-Check -Id $c.Id -Cmd $c.Cmd -Required $c.Required)) { $allOk = $false }
        }
    }
}

if ($WriteSignoff) {
    Write-SignoffTemplate -Wave $Stage -Path $signoffPath
}

$summary = if ($allOk) { "STAGE GATE PASS ($Stage)" } else { "STAGE GATE FAIL ($Stage)" }
Write-Host ""
Write-Host $summary -ForegroundColor $(if ($allOk) { 'Green' } else { 'Red' })
Add-Content -Path $logPath -Value $summary
Write-Host "Log: $logPath"

if (-not $allOk) { exit 1 }
