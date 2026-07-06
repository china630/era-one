#!/usr/bin/env pwsh
# Dev mTLS certs for ingest-gateway + era-agent (GA-1 S5-8)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Out = Join-Path $Root "deploy\tls"
New-Item -ItemType Directory -Force -Path $Out | Out-Null

function Require-OpenSSL {
    if (-not (Get-Command openssl -ErrorAction SilentlyContinue)) {
        Write-Error "openssl not found in PATH"
    }
}

Require-OpenSSL
Push-Location $Out
try {
    if (-not (Test-Path "ca.key")) {
        openssl genrsa -out ca.key 4096 2>$null
        openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 `
            -subj "/CN=ERA-XDR-Dev-CA" -out ca.crt 2>$null
    }
    if (-not (Test-Path "server.key")) {
        openssl genrsa -out server.key 2048 2>$null
        openssl req -new -key server.key -subj "/CN=era-ingest" -out server.csr 2>$null
        @"
subjectAltName = DNS:localhost,DNS:era-ingest,DNS:ingest-gateway,IP:127.0.0.1
"@ | Set-Content -Encoding ascii server-ext.cnf
        openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial `
            -out server.crt -days 825 -sha256 -extfile server-ext.cnf 2>$null
    }
    if (-not (Test-Path "agent.key")) {
        openssl genrsa -out agent.key 2048 2>$null
        openssl req -new -key agent.key -subj "/CN=era-agent" -out agent.csr 2>$null
        openssl x509 -req -in agent.csr -CA ca.crt -CAkey ca.key -CAcreateserial `
            -out agent.crt -days 825 -sha256 2>$null
    }
    Write-Host "TLS dev certs ready in $Out" -ForegroundColor Green
} finally {
    Pop-Location
}
