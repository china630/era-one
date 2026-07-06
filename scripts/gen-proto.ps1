# ERA XDR — генерация Go/Rust стабов из proto/era/v1/
# Refs: S1-1, ADR-0001, ADR-0003
#
# Требования: protoc, protoc-gen-go, protoc-gen-go-grpc (Go plugins),
#             cargo + tonic-build (Rust codegen при cargo build -p era-proto).
#
# Использование:
#   .\scripts\gen-proto.ps1
#   .\scripts\gen-proto.ps1 -RegisterApicurio   # + регистрация в dev-registry

param(
    [switch]$RegisterApicurio
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

function Resolve-ProtocInclude {
    $cmd = Get-Command protoc -ErrorAction SilentlyContinue
    if (-not $cmd) { throw "protoc не найден в PATH" }

    $binDir = Split-Path $cmd.Source -Parent
    $candidates = @(
        (Join-Path $binDir "include"),
        (Join-Path (Split-Path $binDir -Parent) "include"),
        "C:\Program Files\protobuf\include",
        "C:\ProgramData\chocolatey\lib\protobuf\tools\include"
    )
    $wingetPackages = Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Packages"
    if (Test-Path $wingetPackages) {
        Get-ChildItem $wingetPackages -Directory -Filter "Google.Protobuf*" -ErrorAction SilentlyContinue |
            ForEach-Object { $candidates += (Join-Path $_.FullName "include") }
    }
    foreach ($p in $candidates) {
        if (Test-Path (Join-Path $p "google\protobuf\timestamp.proto")) {
            return $p
        }
    }
    throw "Не найден include-path google/protobuf (установите protoc с well-known types)"
}

Write-Host "==> ERA XDR proto codegen (S1-1)" -ForegroundColor Cyan

$protoInclude = Resolve-ProtocInclude
$protoPath = Join-Path $Root "proto"
$goOut = Join-Path $Root "gen\go"

$protos = @(
    "era/v1/envelope.proto",
    "era/v1/ingest.proto"
)

Write-Host "    protoc include: $protoInclude"
Write-Host "    Go output:      $goOut"

$protocArgs = @(
    "--proto_path=$protoPath",
    "--proto_path=$protoInclude",
    "--go_out=$goOut",
    "--go_opt=module=era/contracts/gen",
    "--go-grpc_out=$goOut",
    "--go-grpc_opt=module=era/contracts/gen"
)
foreach ($p in $protos) { $protocArgs += (Join-Path $protoPath $p) }

& protoc @protocArgs
if ($LASTEXITCODE -ne 0) { throw "protoc Go codegen failed" }

Write-Host "    Go stubs OK" -ForegroundColor Green

Push-Location $goOut
go mod tidy
go test ./...
if ($LASTEXITCODE -ne 0) { throw "go test gen/go failed" }
Pop-Location

Write-Host "    Rust codegen (era-proto)..." -ForegroundColor Cyan
cargo test -p era-proto --quiet
if ($LASTEXITCODE -ne 0) { throw "cargo test era-proto failed" }
Write-Host "    Rust stubs OK" -ForegroundColor Green

if ($RegisterApicurio) {
    $registry = "http://localhost:8085"
    Write-Host "    Apicurio register @ $registry ..." -ForegroundColor Cyan
    foreach ($artifact in @(
            @{ Group = "era-one"; Id = "envelope"; File = "era/v1/envelope.proto" },
            @{ Group = "era-one"; Id = "ingest"; File = "era/v1/ingest.proto" }
        )) {
        $bodyPath = Join-Path $protoPath $artifact.File
        $uri = "$registry/apis/registry/v2/groups/$($artifact.Group)/artifacts?artifactId=$($artifact.Id)"
        try {
            Invoke-RestMethod -Method Post -Uri $uri -ContentType "application/x-protobuf" `
                -InFile $bodyPath | Out-Null
            Write-Host "      registered $($artifact.Id)" -ForegroundColor Green
        } catch {
            # dev-registry in-memory: повторная регистрация может конфликтовать — не фatal
            Write-Warning "      $($artifact.Id): $($_.Exception.Message)"
        }
    }
}

Write-Host "`nDone. Generated:" -ForegroundColor Cyan
Write-Host "  gen/go/era/v1/*.pb.go"
Write-Host "  crates/era-proto (build-time via tonic-build)"
