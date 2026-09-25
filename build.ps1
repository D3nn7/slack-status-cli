# Cross-compiles the CLI for all supported platforms into .\dist.
# Usage: .\build.ps1
param(
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"
$binary = "slack-status"
$ldflags = "-s -w -X main.version=$Version"
$platforms = @(
    @("linux", "amd64"),
    @("linux", "arm64"),
    @("darwin", "amd64"),
    @("darwin", "arm64"),
    @("windows", "amd64"),
    @("windows", "arm64")
)

New-Item -ItemType Directory -Force -Path "dist" | Out-Null
foreach ($p in $platforms) {
    $os = $p[0]
    $arch = $p[1]
    $ext = if ($os -eq "windows") { ".exe" } else { "" }
    $out = "dist\$binary-$os-$arch$ext"
    Write-Host "-> $out"
    $env:GOOS = $os
    $env:GOARCH = $arch
    $env:CGO_ENABLED = "0"
    go build -trimpath -ldflags $ldflags -o $out .
}
Remove-Item Env:\GOOS, Env:\GOARCH, Env:\CGO_ENABLED -ErrorAction SilentlyContinue
Write-Host "Builds written to .\dist\"
