# ERA Office Tech Eval - connectivity + API smoke (JWT via workspace BFF).
# Prerequisite (full TE):
#   docker compose -f deploy/docker-compose.office.yml `
#     --profile docs --profile office-engines up -d --wait --build
param(
    [string]$Base = "http://127.0.0.1:8170",
    [string]$Email = "alice@mail.gov.az",
    [string]$Password = "1234",
    [switch]$RequireEngines
)

$ErrorActionPreference = "Stop"
$script:failed = 0
$script:log = New-Object System.Collections.Generic.List[string]

function Write-Step([string]$msg, [bool]$ok) {
    if ($ok) {
        $line = "[PASS] $msg"
        Write-Host $line -ForegroundColor Green
    } else {
        $line = "[FAIL] $msg"
        Write-Host $line -ForegroundColor Red
        $script:failed++
    }
    [void]$script:log.Add($line)
}

function Test-Health([string]$name, [string]$url, [switch]$Optional) {
    try {
        $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 8
        if ($r.StatusCode -ne 200) { throw "status $($r.StatusCode)" }
        Write-Step "$name $url" $true
        return $true
    } catch {
        if ($Optional -and -not $RequireEngines) {
            Write-Host "[SKIP] $name ($($_.Exception.Message))" -ForegroundColor Yellow
            [void]$script:log.Add("[SKIP] $name")
            return $false
        }
        Write-Step "$name $url - $($_.Exception.Message)" $false
        return $false
    }
}

function Get-Json([string]$method, [string]$url, $headers, $body = $null) {
    $params = @{
        Uri             = $url
        Method          = $method
        Headers         = $headers
        UseBasicParsing = $true
        TimeoutSec      = 30
    }
    if ($null -ne $body) {
        # PS 5.1 string -Body can be UTF-16; send explicit UTF-8 bytes for BFF/engines.
        $json = if ($body -is [string]) { $body } else { $body | ConvertTo-Json -Compress -Depth 8 }
        $params.ContentType = "application/json; charset=utf-8"
        $params.Body = [System.Text.Encoding]::UTF8.GetBytes($json)
    }
    return Invoke-WebRequest @params
}

Write-Host "=== Office TE connectivity ($Base) ===" -ForegroundColor Cyan

[void](Test-Health "workspace" "$Base/healthz")
[void](Test-Health "identity" "http://127.0.0.1:8160/healthz")
[void](Test-Health "drive-api" "http://127.0.0.1:8175/healthz")
$docsOk = Test-Health "docs-engine" "http://127.0.0.1:8142/healthz" -Optional
$tablesOk = Test-Health "tables-engine" "http://127.0.0.1:8143/healthz" -Optional
$presOk = Test-Health "presentations-engine" "http://127.0.0.1:8144/healthz" -Optional
$projOk = Test-Health "docs-projects" "http://127.0.0.1:8145/healthz" -Optional
$aiOk = Test-Health "docs-ai" "http://127.0.0.1:8146/healthz" -Optional

foreach ($path in @("/drive/", "/docs/", "/tables/", "/presentations/", "/projects/", "/office-ai/")) {
    try {
        $r = Invoke-WebRequest -Uri "$Base$path" -UseBasicParsing -TimeoutSec 8
        Write-Step "UI $path ($($r.StatusCode))" ($r.StatusCode -eq 200)
    } catch {
        Write-Step "UI $path - $($_.Exception.Message)" $false
    }
}

$token = $null
try {
    $tokResp = Get-Json "POST" "$Base/oauth2/staging/token" @{} @{ email = $Email; password = $Password }
    $token = ($tokResp.Content | ConvertFrom-Json).access_token
    if (-not $token) { throw "empty access_token" }
    Write-Step "staging token for $Email" $true
} catch {
    Write-Step "staging token - $($_.Exception.Message)" $false
}

if ($token) {
    $auth = @{ Authorization = "Bearer $token" }
    $suffix = Get-Date -Format "HHmmss"

    try {
        $r = Get-Json "GET" "$Base/api/v1/drive/folders/_root/children" $auth
        Write-Step "GET /api/v1/drive/folders/_root/children via BFF ($($r.StatusCode))" ($r.StatusCode -eq 200)
    } catch {
        Write-Step "GET /api/v1/drive/folders/_root/children - $($_.Exception.Message)" $false
    }

    try {
        $spoof = Invoke-WebRequest -Uri "http://127.0.0.1:8175/api/v1/drive/objects" -Headers @{
            "X-ERA-Tenant" = "t-demo"
            "X-ERA-User"   = "u-alice"
        } -UseBasicParsing -TimeoutSec 8
        Write-Step "Drive rejects header-only spoof (got $($spoof.StatusCode))" $false
    } catch {
        $code = 0
        if ($_.Exception.Response) {
            $code = [int]$_.Exception.Response.StatusCode
        }
        Write-Step "Drive rejects header-only spoof (HTTP $code)" ($code -eq 401 -or $code -eq 403)
    }

    if ($docsOk) {
        try {
            $r = Get-Json "POST" "$Base/api/v1/docs" $auth @{ name = "te-conn-$suffix.erad" }
            $id = ($r.Content | ConvertFrom-Json).drive_object_id
            Write-Step "POST /api/v1/docs -> $id" (($r.StatusCode -eq 200) -and [bool]$id)
            if ($id) {
                $g = Get-Json "GET" "$Base/api/v1/docs/$id" $auth
                Write-Step "GET /api/v1/docs/$id" ($g.StatusCode -eq 200)
                $s = Get-Json "POST" "$Base/api/v1/docs/$id/snapshot" $auth "{}"
                Write-Step "POST /api/v1/docs/$id/snapshot (PutVersion)" ($s.StatusCode -eq 200)
            }
            # Regression: fixed Drive name "import.erad" → 409 → engine 502 on 2nd import.
            # Unique names: unit `import_erad_names_are_unique_and_not_fixed`. Here: endpoint live ×2.
            $bogusDocxB64 = "QQ=="
            foreach ($n in @("te-import-a-$suffix.docx", "te-import-b-$suffix.docx")) {
                try {
                    $ir = Get-Json "POST" "$Base/api/v1/docs/import" $auth @{
                        name         = $n
                        docx_base64  = $bogusDocxB64
                    }
                    Write-Step "POST /api/v1/docs/import $n ($($ir.StatusCode))" ($ir.StatusCode -eq 400)
                } catch {
                    $code = 0
                    if ($_.Exception.Response) { $code = [int]$_.Exception.Response.StatusCode }
                    Write-Step "POST /api/v1/docs/import $n (HTTP $code; want 400 not 409/502)" ($code -eq 400)
                }
            }
        } catch {
            Write-Step "Docs API - $($_.Exception.Message)" $false
        }
    }

    if ($tablesOk) {
        try {
            $r = Get-Json "POST" "$Base/api/v1/tables" $auth @{ name = "te-conn-$suffix.erat" }
            $id = ($r.Content | ConvertFrom-Json).drive_object_id
            Write-Step "POST /api/v1/tables -> $id" (($r.StatusCode -eq 200) -and [bool]$id)
            if ($id) {
                $g = Get-Json "GET" "$Base/api/v1/tables/$id" $auth
                Write-Step "GET /api/v1/tables/$id" ($g.StatusCode -eq 200)
            }
        } catch {
            Write-Step "Tables API - $($_.Exception.Message)" $false
        }
    }

    if ($presOk) {
        try {
            $r = Get-Json "POST" "$Base/api/v1/presentations" $auth @{ name = "te-conn-$suffix.erap" }
            $id = ($r.Content | ConvertFrom-Json).drive_object_id
            Write-Step "POST /api/v1/presentations -> $id" (($r.StatusCode -eq 200) -and [bool]$id)
            if ($id) {
                $g = Get-Json "GET" "$Base/api/v1/presentations/$id" $auth
                Write-Step "GET /api/v1/presentations/$id" ($g.StatusCode -eq 200)
            }
        } catch {
            Write-Step "Presentations API - $($_.Exception.Message)" $false
        }
    }

    if ($projOk) {
        try {
            $r = Get-Json "GET" "$Base/api/v1/projects/board" $auth
            Write-Step "GET /api/v1/projects/board" ($r.StatusCode -eq 200)
            $c = Get-Json "POST" "$Base/api/v1/projects" $auth @{ name = "te-conn-$suffix.eraj" }
            $erajId = ($c.Content | ConvertFrom-Json).drive_object_id
            Write-Step "POST /api/v1/projects (.eraj) -> $erajId" (($c.StatusCode -eq 200) -and [bool]$erajId)
            if ($erajId) {
                $g = Get-Json "GET" "$Base/api/v1/projects/$erajId" $auth
                Write-Step "GET /api/v1/projects/$erajId" ($g.StatusCode -eq 200)
                $t = Get-Json "POST" "$Base/api/v1/projects/$erajId/tasks" $auth @{
                    title = "te-task-$suffix"
                    board = "backlog"
                }
                Write-Step "POST /api/v1/projects/$erajId/tasks (PutVersion flush)" ($t.StatusCode -eq 200)
            }
        } catch {
            Write-Step "Projects API - $($_.Exception.Message)" $false
        }
    }

    if ($aiOk) {
        try {
            $r = Get-Json "POST" "$Base/api/v1/docs-ai/summarize" $auth @{ text = "ERA Office Tech Eval connectivity check." }
            Write-Step "POST /api/v1/docs-ai/summarize" ($r.StatusCode -eq 200)
        } catch {
            Write-Step "Office AI API - $($_.Exception.Message)" $false
        }
    }
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$outDir = Join-Path (Split-Path $PSScriptRoot -Parent) "reports"
if (-not (Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir | Out-Null }
$outFile = Join-Path $outDir "office-te-connectivity-$stamp.log"
$script:log | Set-Content -Path $outFile -Encoding UTF8
Write-Host ""
Write-Host "Log: $outFile" -ForegroundColor Cyan

if ($script:failed -gt 0) {
    Write-Host "TE connectivity FAILED ($($script:failed))" -ForegroundColor Red
    exit 1
}
Write-Host "TE connectivity PASS" -ForegroundColor Green
exit 0
