# ERA Office P0 smoke — health + upload roundtrip
param(
    [string]$DriveURL = "http://127.0.0.1:8175",
    [string]$WorkspaceURL = "http://127.0.0.1:8170",
    [string]$IdentityURL = "http://127.0.0.1:8160"
)

$ErrorActionPreference = "Stop"

function Test-Health($name, $url) {
    $r = Invoke-WebRequest -Uri $url -UseBasicParsing
    if ($r.StatusCode -ne 200) { throw "$name health failed" }
    Write-Host "[PASS] $name $url" -ForegroundColor Green
}

Test-Health "drive-api" "$DriveURL/healthz"
Test-Health "workspace" "$WorkspaceURL/healthz"
try { Test-Health "identity-api" "$IdentityURL/healthz" } catch { Write-Host "[SKIP] identity-api" -ForegroundColor Yellow }

# Drive is JWT-only (or service token). Prefer workspace staging token via BFF.
$tokenBody = @{ email = "alice@mail.gov.az"; password = "1234" } | ConvertTo-Json
try {
    $tok = Invoke-RestMethod -Uri "$WorkspaceURL/oauth2/staging/token" -Method POST -ContentType "application/json" -Body $tokenBody
    $jwt = $tok.access_token
} catch {
    # Fallback: direct identity
    $tok = Invoke-RestMethod -Uri "$IdentityURL/oauth2/staging/token" -Method POST -ContentType "application/json" -Body $tokenBody
    $jwt = $tok.access_token
}
if (-not $jwt) { throw "staging token missing" }

$boundary = [guid]::NewGuid().ToString()
$body = @"
--$boundary
Content-Disposition: form-data; name="file"; filename="smoke.txt"
Content-Type: text/plain

smoke
--$boundary--
"@
$headers = @{
    "Authorization" = "Bearer $jwt"
    "Content-Type"  = "multipart/form-data; boundary=$boundary"
}
$upload = Invoke-WebRequest -Uri "$WorkspaceURL/api/v1/drive/objects" -Method POST -Headers $headers -Body $body -UseBasicParsing
if ($upload.StatusCode -ne 200) { throw "upload failed" }
Write-Host "[PASS] upload roundtrip (JWT via workspace)" -ForegroundColor Green
