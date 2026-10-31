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
    Push-Location (Join-Path $repositoryRoot "cmd\yawa-ui\frontend")
    try {
        npm ci
        if ($LASTEXITCODE -ne 0) { throw "npm ci failed with exit code $LASTEXITCODE" }
        npm run build
        if ($LASTEXITCODE -ne 0) { throw "frontend build failed with exit code $LASTEXITCODE" }
    } finally {
        Pop-Location
    }
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "Go tests failed with exit code $LASTEXITCODE" }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "Go vet failed with exit code $LASTEXITCODE" }
    $binaryPath = Join-Path $resolvedOutput "yawa.exe"
    go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $binaryPath .\cmd\yawa
    if ($LASTEXITCODE -ne 0) { throw "CLI build failed with exit code $LASTEXITCODE" }
    $agentBinaryPath = Join-Path $resolvedOutput "yawa-agent.exe"
    go build -trimpath -ldflags "-s -w -H windowsgui" -o $agentBinaryPath .\cmd\yawa-agent
    if ($LASTEXITCODE -ne 0) { throw "background agent build failed with exit code $LASTEXITCODE" }
    $uiBinaryPath = Join-Path $resolvedOutput "yawa-ui.exe"
    go build -tags production -trimpath -ldflags "-s -w -H windowsgui" -o $uiBinaryPath .\cmd\yawa-ui
    if ($LASTEXITCODE -ne 0) { throw "UI build failed with exit code $LASTEXITCODE" }
    $checksums = @($binaryPath, $agentBinaryPath, $uiBinaryPath) | ForEach-Object {
        $checksum = (Get-FileHash -Algorithm SHA256 -LiteralPath $_).Hash.ToLowerInvariant()
        "$checksum  $(Split-Path -Leaf $_)"
    }
    Set-Content -LiteralPath (Join-Path $resolvedOutput "SHA256SUMS") -Value $checksums
    Write-Output "Built $binaryPath"
    Write-Output "Built $agentBinaryPath"
    Write-Output "Built $uiBinaryPath"
} finally {
    Pop-Location
}
