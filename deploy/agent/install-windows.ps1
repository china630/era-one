# ERA XDR — install Windows service (GA-1 sketch)
# Run elevated PowerShell from repo root after building era-agent.exe
param(
  [string]$Binary = ".\target\release\era-agent.exe"
)
$ErrorActionPreference = "Stop"
if (-not (Test-Path $Binary)) { throw "Build agent first: cargo build --release -p era-agent" }
$dest = "C:\Program Files\ERA XDR\era-agent.exe"
New-Item -ItemType Directory -Force -Path "C:\Program Files\ERA XDR" | Out-Null
Copy-Item $Binary $dest -Force
$envBlock = @"
ERA_PRODUCTION=1
ERA_GATEWAY_ADDR=http://127.0.0.1:50051
ERA_CONTROL_PLANE_URL=http://127.0.0.1:8090
"@
Set-Content -Path "C:\Program Files\ERA XDR\agent.env" -Value $envBlock
sc.exe create ERA-XDR-Agent binPath= "`"$dest`"" start= auto DisplayName= "ERA XDR Agent"
Write-Host "Service ERA-XDR-Agent created. Set Sysmon/ERA_SYSMON_JSONL and start service."
