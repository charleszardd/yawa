[CmdletBinding()]
param(
    [string]$Version = "dev",
    [string]$OutputDirectory = ".\dist"
)

$ErrorActionPreference = "Stop"
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$resolvedOutput = Join-Path $repositoryRoot $OutputDirectory
New-Item -ItemType Directory -Force -Path $resolvedOutput | Out-Null

Push-Location $repositoryRoot
try {
    go test ./...
    go vet ./...
    $binaryPath = Join-Path $resolvedOutput "yawa.exe"
    go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $binaryPath .\cmd\yawa
    $checksum = (Get-FileHash -Algorithm SHA256 -LiteralPath $binaryPath).Hash.ToLowerInvariant()
    Set-Content -LiteralPath (Join-Path $resolvedOutput "yawa.exe.sha256") -Value "$checksum  yawa.exe"
    Write-Output "Built $binaryPath"
} finally {
    Pop-Location
}
