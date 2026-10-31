[CmdletBinding()]
param(
    [ValidatePattern('^(latest|v?\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)$')]
    [string]$Version = "latest",
    [string]$InstallDirectory = (Join-Path $env:LOCALAPPDATA "Programs\yawa"),
    [string]$PackagePath,
    [string]$ChecksumPath,
    [switch]$SkipPathUpdate,
    [switch]$SkipShortcut
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$repository = "charleszardd/yawa"
$archiveName = "yawa-windows-amd64.zip"
$temporaryDirectory = $null

function Get-ExpectedChecksum {
    param([Parameter(Mandatory)][string]$Path)

    $checksumLine = (Get-Content -Raw -LiteralPath $Path).Trim()
    $checksum = ($checksumLine -split '\s+')[0].ToLowerInvariant()
    if ($checksum -notmatch '^[0-9a-f]{64}$') {
        throw "Invalid SHA-256 checksum in $Path"
    }
    return $checksum
}

function Add-InstallDirectoryToUserPath {
    param([Parameter(Mandatory)][string]$Directory)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = @($userPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    $normalizedDirectory = $Directory.TrimEnd('\')
    $alreadyConfigured = $pathEntries | Where-Object {
        $_.TrimEnd('\').Equals($normalizedDirectory, [StringComparison]::OrdinalIgnoreCase)
    }
    if ($alreadyConfigured) {
        return $false
    }

    $updatedPath = (@($pathEntries) + $Directory) -join ';'
    [Environment]::SetEnvironmentVariable("Path", $updatedPath, "User")
    return $true
}

function New-YawaShortcut {
    param(
        [Parameter(Mandatory)][string]$ExecutablePath,
        [Parameter(Mandatory)][string]$WorkingDirectory
    )

    $programsDirectory = [Environment]::GetFolderPath("Programs")
    $shortcutPath = Join-Path $programsDirectory "Yawa UI.lnk"
    $shell = New-Object -ComObject WScript.Shell
    $shortcut = $shell.CreateShortcut($shortcutPath)
    $shortcut.TargetPath = $ExecutablePath
    $shortcut.WorkingDirectory = $WorkingDirectory
    $shortcut.IconLocation = "$ExecutablePath,0"
    $shortcut.Description = "Customize Yawa launcher commands and hotkeys"
    $shortcut.Save()
    return $shortcutPath
}

try {
    if ([string]::IsNullOrWhiteSpace($PackagePath)) {
        $temporaryDirectory = Join-Path ([IO.Path]::GetTempPath()) ("yawa-install-" + [Guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null
        $PackagePath = Join-Path $temporaryDirectory $archiveName
        $ChecksumPath = "$PackagePath.sha256"

        if ($Version -eq "latest") {
            $releaseBaseUrl = "https://github.com/$repository/releases/latest/download"
        } else {
            $releaseBaseUrl = "https://github.com/$repository/releases/download/$Version"
        }

        Write-Output "Downloading Yawa $Version..."
        Invoke-WebRequest -UseBasicParsing -Uri "$releaseBaseUrl/$archiveName" -OutFile $PackagePath
        Invoke-WebRequest -UseBasicParsing -Uri "$releaseBaseUrl/$archiveName.sha256" -OutFile $ChecksumPath
    } else {
        $PackagePath = (Resolve-Path -LiteralPath $PackagePath).Path
        if ([string]::IsNullOrWhiteSpace($ChecksumPath)) {
            $ChecksumPath = "$PackagePath.sha256"
        }
        $ChecksumPath = (Resolve-Path -LiteralPath $ChecksumPath).Path
    }

    $expectedChecksum = Get-ExpectedChecksum -Path $ChecksumPath
    $actualChecksum = (Get-FileHash -Algorithm SHA256 -LiteralPath $PackagePath).Hash.ToLowerInvariant()
    if ($actualChecksum -ne $expectedChecksum) {
        throw "Package checksum mismatch. Expected $expectedChecksum but received $actualChecksum."
    }

    $extractionDirectory = Join-Path ([IO.Path]::GetTempPath()) ("yawa-extract-" + [Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $extractionDirectory | Out-Null
    try {
        Expand-Archive -LiteralPath $PackagePath -DestinationPath $extractionDirectory
        $requiredExecutables = @("yawa.exe", "yawa-agent.exe", "yawa-ui.exe")
        foreach ($executable in $requiredExecutables) {
            $sourcePath = Join-Path $extractionDirectory $executable
            if (-not (Test-Path -LiteralPath $sourcePath -PathType Leaf)) {
                throw "Release package is missing $executable"
            }
        }

        New-Item -ItemType Directory -Force -Path $InstallDirectory | Out-Null
        foreach ($executable in $requiredExecutables) {
            Copy-Item -Force -LiteralPath (Join-Path $extractionDirectory $executable) -Destination (Join-Path $InstallDirectory $executable)
        }
        Copy-Item -Force -LiteralPath (Join-Path $extractionDirectory "SHA256SUMS") -Destination (Join-Path $InstallDirectory "SHA256SUMS")
    } finally {
        if (Test-Path -LiteralPath $extractionDirectory) {
            Remove-Item -Recurse -Force -LiteralPath $extractionDirectory
        }
    }

    $pathUpdated = $false
    if (-not $SkipPathUpdate) {
        $pathUpdated = Add-InstallDirectoryToUserPath -Directory $InstallDirectory
    }

    $shortcutPath = $null
    if (-not $SkipShortcut) {
        $shortcutPath = New-YawaShortcut -ExecutablePath (Join-Path $InstallDirectory "yawa-ui.exe") -WorkingDirectory $InstallDirectory
    }

    Write-Output "Yawa installed to $InstallDirectory"
    if ($pathUpdated) {
        Write-Output "Added Yawa to your user PATH. Open a new terminal before running yawa."
    }
    if ($shortcutPath) {
        Write-Output "Created Start menu shortcut: $shortcutPath"
    }
    Write-Output "Run the customization UI with: yawa-ui"
    Write-Output "Start global shortcuts from the Yawa UI, then optionally enable Start with Windows."
} finally {
    if ($temporaryDirectory -and (Test-Path -LiteralPath $temporaryDirectory)) {
        Remove-Item -Recurse -Force -LiteralPath $temporaryDirectory
    }
}
