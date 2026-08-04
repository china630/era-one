# B-BR — 100-mailbox synthetic EWS FindFolder harness (no real Exchange)
param(
    [string]$BridgeAPI = "http://127.0.0.1:8151",
    [int]$Mailboxes = 100
)
$ErrorActionPreference = "Stop"
$log = Join-Path $PSScriptRoot "..\reports\comms-bridge-100mb-lab.log"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null
function Log($msg) { $line = "$(Get-Date -Format o) BRIDGE-100 $msg"; Write-Host $line; Add-Content $log $line }

$soap = @'
<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <m:FindFolder xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages"/>
  </soap:Body>
</soap:Envelope>
'@

$ok = 0
for ($i = 1; $i -le $Mailboxes; $i++) {
    try {
        $resp = Invoke-WebRequest -Uri "$BridgeAPI/ews/Exchange.asmx" -Method POST -Body $soap `
            -ContentType "text/xml" -Headers @{ SOAPAction = "FindFolder"; "X-ERA-Mailbox" = "mb$i@mail.gov.az" } -UseBasicParsing
        if ($resp.Content -match "FindFolderResponse|Success|Folder") { $ok++ }
        else { Log "mb$i unexpected: $($resp.Content.Substring(0, [Math]::Min(120, $resp.Content.Length)))" }
    } catch {
        Log "mb$i fail: $($_.Exception.Message)"
    }
}
Log "FindFolder OK $ok / $Mailboxes"
if ($ok -lt [Math]::Max(1, [int]($Mailboxes * 0.9))) { throw "bridge 100mb lab failed: $ok/$Mailboxes" }
Log "BRIDGE 100mb lab PASS — see $log"
Write-Host "PASS — see $log"
