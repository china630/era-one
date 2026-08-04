# L-1 IMAP LOGIN smoke against dovecot-lab (host port 1144)
param(
    [string]$HostAddr = "127.0.0.1",
    [int]$Port = 1144
)
$ErrorActionPreference = "Stop"
$log = Join-Path $PSScriptRoot "..\reports\comms-lab-imap-smoke.log"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null
function Log($msg) { $line = "$(Get-Date -Format o) $msg"; Write-Host $line; Add-Content $log $line }

$tcp = New-Object System.Net.Sockets.TcpClient
$tcp.Connect($HostAddr, $Port)
$stream = $tcp.GetStream()
$sw = New-Object System.IO.StreamWriter($stream); $sw.AutoFlush = $true
$sr = New-Object System.IO.StreamReader($stream)
$banner = $sr.ReadLine()
Log "banner: $banner"
$sw.WriteLine("a1 LOGIN lab1@mail.gov.az lab")
$resp = $sr.ReadLine()
Log "LOGIN: $resp"
if ($resp -notmatch "OK") { throw "LOGIN failed: $resp" }
$sw.WriteLine("a2 SELECT INBOX")
while ($true) {
    $l = $sr.ReadLine()
    Log "SELECT: $l"
    if ($l -match "^a2 ") { break }
}
$sw.WriteLine("a3 LOGOUT")
$tcp.Close()
Log "L-1 IMAP LOGIN smoke PASS"
Write-Host "PASS — see $log"
