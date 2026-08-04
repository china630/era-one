# ERA Communications — staging pilot RT-01…RT-08 (gap-list §6.2 mapping)
param(
    [string]$MailAPI = "http://127.0.0.1:8150",
    [string]$UIMail = "http://127.0.0.1:8180",
    [string]$IdentityAPI = "http://127.0.0.1:8160",
    [string]$MailCoreSMTP = "127.0.0.1:2525",
    [string]$MailCoreIMAP = "127.0.0.1:1143",
    [string]$ClickHouse = "http://127.0.0.1:8123",
    [string]$ClickHouseUser = "era",
    [string]$ClickHousePassword = "era_dev_only",
    [switch]$UseCompose,
    [switch]$ProdProfile,
    [switch]$AllowSkipInfra
)
$ErrorActionPreference = "Stop"
$log = Join-Path $PSScriptRoot "..\reports\comms-pilot-staging.log"
New-Item -ItemType Directory -Force -Path (Split-Path $log) | Out-Null
function Log($msg) { $line = "$(Get-Date -Format o) $msg"; Write-Host $line; Add-Content $log $line }

if ($UseCompose) {
    Log "RT-01 compose up --wait$(if ($ProdProfile) { ' (prod profile)' } else { '' })"
    $compose = Join-Path $PSScriptRoot "..\deploy\docker-compose.comms.yml"
    $dev = Join-Path $PSScriptRoot "..\deploy\docker-compose.comms.dev.yml"
    $prod = Join-Path $PSScriptRoot "..\deploy\docker-compose.comms.prod.yml"
    $files = @("-f", $compose)
    if ($ProdProfile) {
        $gen = Join-Path $PSScriptRoot "..\deploy\comms-tls\gen-dev-certs.ps1"
        if (Test-Path $gen) { & $gen | ForEach-Object { Log "tls: $_" } }
        if (Test-Path $prod) { $files += @("-f", $prod) }
    } elseif (Test-Path $dev) {
        $files += @("-f", $dev)
    }
    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    docker compose @files up -d --wait 2>&1 | ForEach-Object { Log "compose: $_" }
    $composeExit = $LASTEXITCODE
    $ErrorActionPreference = $prevEap
    if ($composeExit -ne 0) {
        if ($AllowSkipInfra) {
            Log "RT-01 compose SKIP (exit $composeExit) — AllowSkipInfra"
            Log "STAGING SKIP (infra unavailable) - see $log"
            exit 0
        }
        throw "compose up failed (exit $composeExit)"
    }
}

Log "RT-01 healthz/readyz"
$hz = Invoke-WebRequest -Uri "$MailAPI/healthz" -UseBasicParsing
$rz = Invoke-WebRequest -Uri "$MailAPI/readyz" -UseBasicParsing
$ready = ($rz.Content | ConvertFrom-Json)
if (-not $ready.ready) { throw "readyz not ready: $($rz.Content)" }
Log "RT-01 readyz checks: $($rz.Content)"

Log "RT-02 mailbox provision (pre-Outlook/EWS)"
$body = @{ tenant_id="t-demo"; email="staging@mail.gov.az"; password="staging-pass"; quota_bytes=536870912 } | ConvertTo-Json
try {
    Invoke-WebRequest -Uri "$MailAPI/api/v1/mailboxes" -Method POST -Body $body -ContentType "application/json" -UseBasicParsing | Out-Null
} catch {
    if ($_.Exception.Response.StatusCode.value__ -ne 409 -and $_.ErrorDetails.Message -notmatch "exists") { throw }
    Log "RT-02 mailbox already provisioned"
}

Log "RT-03 Thunderbird SMTP AUTH deliver"
$smtp = New-Object System.Net.Sockets.TcpClient($MailCoreSMTP.Split(":")[0], [int]$MailCoreSMTP.Split(":")[1])
$sw = New-Object System.IO.StreamWriter($smtp.GetStream())
$sw.AutoFlush = $true
$sr = New-Object System.IO.StreamReader($smtp.GetStream())
$sw.WriteLine("") | Out-Null; $sr.ReadLine() | Out-Null
$sw.WriteLine("EHLO staging") | Out-Null
while ($true) { $l = $sr.ReadLine(); if ($l -match "^250 [^\-]") { break } }
$auth = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes([char]0 + "staging@mail.gov.az" + [char]0 + "staging-pass"))
$sw.WriteLine("AUTH PLAIN $auth") | Out-Null; $sr.ReadLine() | Out-Null
$sw.WriteLine("MAIL FROM:<staging@mail.gov.az>") | Out-Null; $sr.ReadLine() | Out-Null
$sw.WriteLine("RCPT TO:<staging@mail.gov.az>") | Out-Null; $sr.ReadLine() | Out-Null
$sw.WriteLine("DATA") | Out-Null; $sr.ReadLine() | Out-Null
$sw.WriteLine("Subject: staging-smtp`r") | Out-Null
$sw.WriteLine("`r") | Out-Null
$sw.WriteLine("SMTP staging body`r") | Out-Null
$sw.WriteLine(".") | Out-Null; $smtpResp = $sr.ReadLine()
if ($smtpResp -notmatch "^250") { throw "SMTP deliver failed: $smtpResp" }
$sw.WriteLine("QUIT") | Out-Null
$smtp.Close()

Log "RT-03 Thunderbird IMAP FETCH"
$imap = New-Object System.Net.Sockets.TcpClient($MailCoreIMAP.Split(":")[0], [int]$MailCoreIMAP.Split(":")[1])
$iw = New-Object System.IO.StreamWriter($imap.GetStream())
$iw.AutoFlush = $true
$ir = New-Object System.IO.StreamReader($imap.GetStream())
$ir.ReadLine() | Out-Null
$iw.WriteLine('a1 LOGIN "staging@mail.gov.az" "staging-pass"') | Out-Null; $ir.ReadLine() | Out-Null
$iw.WriteLine('a2 SELECT "INBOX"') | Out-Null
while ($true) { $l = $ir.ReadLine(); if ($l -match "^a2 OK") { break } }
$iw.WriteLine("a3 FETCH 1 BODY[]") | Out-Null
$imapBody = ""
while ($true) { $l = $ir.ReadLine(); $imapBody += "$l`n"; if ($l -match "^a3 OK") { break } }
if ($imapBody -notmatch "SMTP staging body") { throw "IMAP FETCH missing body" }
$imap.Close()

Log "RT-02 Outlook EWS CreateItem smoke"
$soap = '<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><CreateItem><Items><Message><Subject>Staging</Subject><Body>Hi</Body></Message></Items></CreateItem></soap:Body></soap:Envelope>'
Invoke-WebRequest -Uri "$MailAPI/ews/Exchange.asmx" -Method POST -Body $soap -ContentType "text/xml" -Headers @{ SOAPAction = "CreateItem" } -UseBasicParsing | Out-Null

Log "RT-04 CalDAV PUT"
$ical = "BEGIN:VCALENDAR`r`nBEGIN:VEVENT`r`nUID:staging-evt`r`nSUMMARY:Pilot`r`nEND:VEVENT`r`nEND:VCALENDAR"
Invoke-WebRequest -Uri "$MailAPI/caldav/staging@mail.gov.az/staging-evt.ics" -Method PUT -Body $ical -ContentType "text/calendar" -UseBasicParsing | Out-Null

Log "RT-04b CardDAV PUT"
$vcf = "BEGIN:VCARD`r`nVERSION:3.0`r`nFN:Staging Contact`r`nEND:VCARD"
Invoke-WebRequest -Uri "$MailAPI/carddav/staging@mail.gov.az/contact-1.vcf" -Method PUT -Body $vcf -ContentType "text/vcard" -UseBasicParsing | Out-Null

Log "RT-05 webmail OIDC machine-flow"
$tokBody = @{ email="staging@mail.gov.az"; password="staging-pass" } | ConvertTo-Json
try {
    $tokResp = Invoke-WebRequest -Uri "$IdentityAPI/oauth2/staging/token" -Method POST -Body $tokBody -ContentType "application/json" -UseBasicParsing
} catch {
    throw "RT-05 OIDC staging token failed (is ERA_IDENTITY_DEV=1?): $_"
}
$tok = ($tokResp.Content | ConvertFrom-Json).access_token
if (-not $tok) { throw "no access_token in OIDC response" }
$hdr = @{ Authorization = "Bearer $tok" }
$sendBody = @{ from="staging@mail.gov.az"; to="staging@mail.gov.az"; subject="oidc-staging"; body="OIDC staging hello" } | ConvertTo-Json
$sendResp = Invoke-WebRequest -Uri "$UIMail/mail/api/send" -Method POST -Body $sendBody -ContentType "application/json" -Headers $hdr -UseBasicParsing
$sentID = ($sendResp.Content | ConvertFrom-Json).id
$inbox = Invoke-WebRequest -Uri "$UIMail/mail/api/messages" -Headers $hdr -UseBasicParsing
$ids = @(($inbox.Content | ConvertFrom-Json).messages | ForEach-Object { $_.ID })
if ($ids -notcontains $sentID) { throw "RT-05 inbox missing sent message id=$sentID" }
Log "RT-05 OIDC send+list PASS"

Log "RT-06 ActiveSync Provision"
Invoke-WebRequest -Uri "$MailAPI/Microsoft-Server-ActiveSync?Cmd=Provision" -Method POST -Body "" -UseBasicParsing | Out-Null

Log "RT-06 wait for async SMTP audit webhook"
Start-Sleep -Seconds 2

Log "RT-06 ClickHouse audit count"
$chQ = "SELECT count() FROM era_comms.mail_audit FORMAT TabSeparated"
$chAuth = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("${ClickHouseUser}:${ClickHousePassword}"))
$chHdr = @{ Authorization = "Basic $chAuth" }
$chResp = Invoke-WebRequest -Uri "$ClickHouse/?query=$([uri]::EscapeDataString($chQ))" -Headers $chHdr -UseBasicParsing
$chCount = $chResp.Content.Trim()
Log "RT-06 CH audit rows: $chCount"
# L-2: no synthetic fallback — REST/SMTP audit path must populate CH
if ([int]$chCount -le 0) { throw "RT-06 mail_audit count must be > 0 (AC-C7; no fallback)" }

Log "RT-07 policy deny oversized"
$huge = "x" * (26 * 1024 * 1024)
$sendBody = '{"from":"staging@mail.gov.az","to":"staging@mail.gov.az","subject":"big","body":"' + $huge + '"}'
$rt07Hdr = @{ "Content-Type" = "application/json" }
if ($tok) { $rt07Hdr["Authorization"] = "Bearer $tok" }
try {
    Invoke-WebRequest -Uri "$MailAPI/api/v1/mail/send" -Method POST -Body $sendBody -Headers $rt07Hdr -UseBasicParsing | Out-Null
    throw "expected 413 for oversized send"
} catch {
    if ($_.Exception.Response.StatusCode.value__ -ne 413) { throw }
    Log "RT-07 policy deny PASS (413)"
}

Log "RT-08 autodiscover TLS honesty"
$ad = Invoke-WebRequest -Uri "$MailAPI/autodiscover/autodiscover.xml?email=staging@mail.gov.az" -UseBasicParsing
if ($ProdProfile) {
    if ($ad.Content -notmatch "<SSL>on</SSL>") {
        throw "RT-08 ProdProfile requires <SSL>on</SSL>, got: $($ad.Content.Substring(0, [Math]::Min(200, $ad.Content.Length)))"
    }
    Log "RT-08 autodiscover SSL on PASS (ProdProfile)"
} elseif ($ad.Content -match "<SSL>on</SSL>") {
    Log "RT-08 autodiscover SSL on PASS"
} elseif ($ad.Content -match "<SSL>off</SSL>") {
    Log "RT-08 autodiscover SSL off PASS (lab honesty — ERA_MAIL_TLS unset; set ERA_MAIL_TLS=1 for SSL on)"
} else {
    throw "autodiscover missing SSL element"
}

Log "STAGING PASS - see $log"
