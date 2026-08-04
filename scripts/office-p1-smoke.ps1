# ERA Office P1 smoke — docs-engine health
param(
    [string]$DocsURL = "http://127.0.0.1:8142"
)

$ErrorActionPreference = "Stop"
$r = Invoke-WebRequest -Uri "$DocsURL/healthz" -UseBasicParsing
if ($r.StatusCode -ne 200) { throw "docs-engine health failed" }
Write-Host "[PASS] docs-engine $DocsURL/healthz" -ForegroundColor Green
