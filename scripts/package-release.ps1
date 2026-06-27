param(
    [Parameter(Mandatory = $true)]
    [string]$Version,
    [string]$GOOS = $(go env GOOS),
    [string]$GOARCH = $(go env GOARCH),
    [string]$OutDir = "dist",
    [string]$CC = "",
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

$pluginID = "model-fallback-router"
$extension = switch ($GOOS) {
    "windows" { ".dll"; break }
    "darwin" { ".dylib"; break }
    default { ".so" }
}

$libraryName = "$pluginID$extension"
$archiveName = "${pluginID}_${Version}_${GOOS}_${GOARCH}.zip"
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$outDirPath = (Resolve-Path -LiteralPath $OutDir).Path
$packageDir = Join-Path $outDirPath "package-$GOOS-$GOARCH"
New-Item -ItemType Directory -Force -Path $packageDir | Out-Null
$libraryPath = Join-Path $packageDir $libraryName
$archivePath = Join-Path $outDirPath $archiveName

$oldCGO = $env:CGO_ENABLED
$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
$oldCC = $env:CC
try {
    $env:CGO_ENABLED = "1"
    $env:GOOS = $GOOS
    $env:GOARCH = $GOARCH
    if ($CC.Trim() -ne "") {
        $env:CC = $CC
    }

    if (-not $SkipTests) {
        go test ./...
    }

    go build -trimpath -buildmode=c-shared -ldflags="-s -w" -o $libraryPath .

    Push-Location $packageDir
    try {
        Compress-Archive -LiteralPath $libraryName -DestinationPath $archivePath -Force
    }
    finally {
        Pop-Location
    }

    Write-Host "Built $archivePath with $libraryName at archive root"
}
finally {
    $env:CGO_ENABLED = $oldCGO
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
    $env:CC = $oldCC
}
