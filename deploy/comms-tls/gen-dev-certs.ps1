# Generate lab self-signed TLS certs for mail-core prod overlay (air-gap safe).
param(
    [string]$OutDir = $PSScriptRoot
)
$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$crt = Join-Path $OutDir "server.crt"
$key = Join-Path $OutDir "server.key"
if ((Test-Path $crt) -and (Test-Path $key)) {
    Write-Host "certs already exist: $crt"
    exit 0
}
# Prefer openssl if present; else .NET
$openssl = Get-Command openssl -ErrorAction SilentlyContinue
if ($openssl) {
    & openssl req -x509 -newkey rsa:2048 -nodes -keyout $key -out $crt -days 825 -subj "/CN=mail.mail.gov.az"
    Write-Host "wrote openssl certs"
    exit 0
}
# Fallback: empty marker so compose mount works; cargo TLS e2e generates its own
"# ERA lab placeholder — run openssl or gen-dev-certs with OpenSSL installed`n" | Set-Content -Path $crt -Encoding ascii
"# replace with private key`n" | Set-Content -Path $key -Encoding ascii
Write-Host "WARNING: openssl missing — wrote placeholders; install OpenSSL for real TLS"
