[CmdletBinding()]
param(
    [string]$BuildDirectory = ".\dist",
    [string]$OutputDirectory = ".\release"
)

$ErrorActionPreference = "Stop"
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$resolvedBuildDirectory = Join-Path $repositoryRoot $BuildDirectory
$resolvedOutputDirectory = Join-Path $repositoryRoot $OutputDirectory
$archiveName = "yawa-windows-amd64.zip"
$archivePath = Join-Path $resolvedOutputDirectory $archiveName

$packageFiles = @(
    (Join-Path $resolvedBuildDirectory "yawa.exe"),
    (Join-Path $resolvedBuildDirectory "yawa-agent.exe"),
    (Join-Path $resolvedBuildDirectory "yawa-ui.exe"),
    (Join-Path $resolvedBuildDirectory "SHA256SUMS")
)

foreach ($packageFile in $packageFiles) {
    if (-not (Test-Path -LiteralPath $packageFile -PathType Leaf)) {
        throw "Release input is missing: $packageFile"
    }
}

New-Item -ItemType Directory -Force -Path $resolvedOutputDirectory | Out-Null
if (Test-Path -LiteralPath $archivePath) {
    Remove-Item -Force -LiteralPath $archivePath
}

Compress-Archive -LiteralPath $packageFiles -DestinationPath $archivePath -CompressionLevel Optimal
$archiveChecksum = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
Set-Content -LiteralPath "$archivePath.sha256" -Value "$archiveChecksum  $archiveName" -Encoding Ascii
Copy-Item -Force -LiteralPath (Join-Path $PSScriptRoot "install.ps1") -Destination (Join-Path $resolvedOutputDirectory "install.ps1")

Write-Output "Packaged $archivePath"
Write-Output "Checksum $archivePath.sha256"
