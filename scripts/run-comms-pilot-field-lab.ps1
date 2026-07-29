# ERA Communications — internal field lab (Phase 2, not customer RT-09)
param(
    [string]$MailAPI = "http://127.0.0.1:8150",
    [string]$MailBridge = "http://127.0.0.1:8151",
    [string]$MigrationAPI = "http://127.0.0.1:8350",
    [string]$ClickHouse = "http://127.0.0.1:8123",
    [string]$ClickHouseUser = "era",
    [string]$ClickHousePassword = "era_dev_only",
    [switch]$UseCompose
)
$ErrorActionPreference = "Stop"
$log = Join-Path $PSScriptRoot "..\reports\comms-pilot-field-lab.log"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null
function Log($msg) { $line = "$(Get-Date -Format o) FIELD-LAB $msg"; Write-Host $line; Add-Content $log $line }

if ($UseCompose) {
    Log "compose partner profile up --wait"
    $base = Join-Path $PSScriptRoot "..\deploy\docker-compose.comms.yml"
    $partner = Join-Path $PSScriptRoot "..\deploy\docker-compose.comms.partner.yml"
    $dev = Join-Path $PSScriptRoot "..\deploy\docker-compose.comms.dev.yml"
    $envFile = Join-Path $PSScriptRoot "..\deploy\comms-partner.env"
    $args = @("-f", $base, "-f", $dev, "-f", $partner, "up", "-d", "--wait")
    if (Test-Path $envFile) { $args = @("-f", $base, "-f", $dev, "-f", $partner, "--env-file", $envFile, "up", "-d", "--wait") }
    docker compose @args
    if ($LASTEXITCODE -ne 0) { throw "compose up failed" }
}

Log "A2 bridge healthz"
$bh = Invoke-WebRequest -Uri "$MailBridge/healthz" -UseBasicParsing
if ($bh.StatusCode -ne 200) { throw "bridge healthz failed" }

Log "A2 migration healthz"
$mh = Invoke-WebRequest -Uri "$MigrationAPI/healthz" -UseBasicParsing
if ($mh.StatusCode -ne 200) { throw "migration healthz failed" }

Log "A4 bridge autodiscover smoke"
$ad = Invoke-WebRequest -Uri "$MailBridge/autodiscover/autodiscover.xml?email=pilot@mail.lab.local" -UseBasicParsing
if ($ad.Content -notmatch "/ews/Exchange.asmx") { throw "autodiscover missing EwsUrl to bridge" }
if ($ad.Content -notmatch "<SSL>on</SSL>") { throw "autodiscover SSL off" }

Log "A4 bridge EWS FindFolder smoke"
$soap = '<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><FindFolder/></soap:Body></soap:Envelope>'
try {
    $ews = Invoke-WebRequest -Uri "$MailBridge/ews/Exchange.asmx" -Method POST -Body $soap -ContentType "text/xml" -Headers @{ SOAPAction = "FindFolder" } -UseBasicParsing
    Log "A4 EWS status $($ews.StatusCode) (502 expected without lab upstream)"
} catch {
    Log "A4 EWS upstream not configured (expected in stub lab): $($_.Exception.Message)"
}

Log "A3 Mail Server baseline — manual Outlook VM checklist (see docs/Comms-Internal-Field-Lab.md §Outlook)"
Log "A5 Thunderbird/CG optional — manual if CG lab available"

Log "CH bridge audit probe"
$chQ = "SELECT count() FROM era_comms.mail_audit WHERE action LIKE 'BRIDGE_%' FORMAT TabSeparated"
$chAuth = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("${ClickHouseUser}:${ClickHousePassword}"))
try {
    $chResp = Invoke-WebRequest -Uri "$ClickHouse/?query=$([uri]::EscapeDataString($chQ))" -Headers @{ Authorization = "Basic $chAuth" } -UseBasicParsing
    Log "BRIDGE audit rows: $($chResp.Content.Trim())"
} catch {
    Log "CH audit skip: $($_.Exception.Message)"
}

Log "FIELD-LAB PASS (automated probes; complete A3/A4 manual Outlook steps for full AC-BR-1/2)"
