param(
    [string]$GOOS = $(go env GOOS),
    [string]$GOARCH = $(go env GOARCH),
    [string]$OutDir = "dist",
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

$extension = switch ($GOOS) {
    "windows" { ".dll"; break }
    "darwin" { ".dylib"; break }
    default { ".so" }
}

$artifact = "model-fallback-router-$GOOS-$GOARCH$extension"
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$oldCGO = $env:CGO_ENABLED
$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
try {
    $env:CGO_ENABLED = "1"
    $env:GOOS = $GOOS
    $env:GOARCH = $GOARCH

    if (-not $SkipTests) {
        go test ./...
    }

    go build -trimpath -buildmode=c-shared -ldflags="-s -w" -o (Join-Path $OutDir $artifact) .
    Write-Host "Built $OutDir/$artifact"
}
finally {
    $env:CGO_ENABLED = $oldCGO
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
}
