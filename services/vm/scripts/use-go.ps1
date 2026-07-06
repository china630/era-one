# Подхватывает Go в текущей сессии PowerShell, если go.exe не в PATH.
# Использование:  . "d:\My Projects\era-one\services\vm\scripts\use-go.ps1"
# Затем: go version, go mod tidy, go test ./..., go run ./cmd/vm-engine

$ErrorActionPreference = "Stop"

$candidates = @(
    (Join-Path $env:ProgramFiles "Go\bin\go.exe"),
    (Join-Path ${env:ProgramFiles(x86)} "Go\bin\go.exe"),
    (Join-Path $env:LOCALAPPDATA "Programs\Go\bin\go.exe"),
    (Join-Path $env:USERPROFILE "go\bin\go.exe"),
    (Join-Path $env:USERPROFILE "sdk\go\bin\go.exe")
)

$found = $null
foreach ($p in $candidates) {
    if (Test-Path -LiteralPath $p) {
        $found = $p
        break
    }
}

if (-not $found) {
    Write-Host "Go не найден в стандартных путях." -ForegroundColor Red
    Write-Host "Установите Go: https://go.dev/dl/ (Windows MSI) или: winget install GoProgrammingLanguage.Go" -ForegroundColor Yellow
    Write-Host "После установки закройте и снова откройте терминал (или перезагрузите ПК), чтобы обновился PATH." -ForegroundColor Yellow
    return
}

$binDir = Split-Path -Parent $found
if ($env:Path -notlike "*$binDir*") {
    $env:Path = "$binDir;$env:Path"
}

Write-Host "Используется Go: $found" -ForegroundColor Green
& $found version
