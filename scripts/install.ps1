# install.ps1 — Install arx and WebUI assets from the latest GitHub release.
# Usage: .\install.ps1 [-User] [-System]   (default: -User)

#Requires -Version 5.1

[CmdletBinding(DefaultParameterSetName = 'User')]
param(
    [Parameter(ParameterSetName = 'System')]
    [switch]$System,

    [Parameter(ParameterSetName = 'User')]
    [switch]$User
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$Repo = 'ARCOOON/arx-ca'
$GitHubApi = "https://api.github.com/repos/$Repo/releases/latest"

function Test-Administrator {
    $identity = [Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()
    return $identity.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Add-InstallPath {
    param(
        [string]$Directory,
        [EnvironmentVariableTarget]$Target
    )

    $normalized = $Directory.TrimEnd('\')
    $current = [Environment]::GetEnvironmentVariable('Path', $Target)

    if ([string]::IsNullOrWhiteSpace($current)) {
        [Environment]::SetEnvironmentVariable('Path', $normalized, $Target)
        return
    }

    $segments = $current -split ';' | Where-Object {
        $_.Trim() -ne '' -and $_.TrimEnd('\') -ne $normalized
    }

    $newPath = ($segments + $normalized) -join ';'
    [Environment]::SetEnvironmentVariable('Path', $newPath, $Target)
}

function Sync-SessionPath {
    $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')

    if ([string]::IsNullOrWhiteSpace($machinePath)) {
        $env:Path = $userPath
    } elseif ([string]::IsNullOrWhiteSpace($userPath)) {
        $env:Path = $machinePath
    } else {
        $env:Path = "$machinePath;$userPath"
    }
}

function Preserve-UserData {
    param([string]$InstallDir)

    if (Test-Path (Join-Path $InstallDir 'server.yaml')) {
        Write-Host 'Preserving existing server.yaml'
    }
    if (Test-Path (Join-Path $InstallDir '.pki')) {
        Write-Host 'Preserving existing .pki/ directory'
    }
}

if ($System) {
    if (-not (Test-Administrator)) {
        Write-Error 'Error: -System requires Administrator privileges. Run PowerShell as Administrator.'
        exit 1
    }
    $InstallDir = Join-Path $env:ProgramFiles 'arx'
    $PathTarget = [EnvironmentVariableTarget]::Machine
    $ScopeLabel = 'system'
} else {
    $InstallDir = Join-Path $env:LOCALAPPDATA 'arx'
    $PathTarget = [EnvironmentVariableTarget]::User
    $ScopeLabel = 'user'
}

Write-Host "Installing arx ($ScopeLabel scope)"
Write-Host "  Install directory: $InstallDir"

try {
    $release = Invoke-RestMethod -Uri $GitHubApi -UseBasicParsing
} catch {
    Write-Error "Failed to fetch latest release from GitHub API: $_"
    exit 1
}

$tag = $release.tag_name
if ([string]::IsNullOrWhiteSpace($tag)) {
    Write-Error 'Could not parse latest release tag from GitHub API response.'
    exit 1
}

Write-Host "  Release tag:       $tag"

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("arx-install-" + [guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    $BaseUrl = "https://github.com/$Repo/releases/download/$tag"
    $BinaryAsset = 'arx-windows-amd64.exe'
    $BinaryDownload = Join-Path $TempDir $BinaryAsset
    $WebUiDownload = Join-Path $TempDir 'webui-dist.tar.gz'
    $BinaryDest = Join-Path $InstallDir 'arx.exe'
    $UiDir = Join-Path $InstallDir 'ui'

    Write-Host "Downloading $BinaryAsset..."
    Invoke-WebRequest -Uri "$BaseUrl/$BinaryAsset" -OutFile $BinaryDownload -UseBasicParsing

    Write-Host 'Downloading webui-dist.tar.gz...'
    Invoke-WebRequest -Uri "$BaseUrl/webui-dist.tar.gz" -OutFile $WebUiDownload -UseBasicParsing

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Preserve-UserData -InstallDir $InstallDir

    Write-Host "Installing binary to $BinaryDest..."
    Move-Item -Path $BinaryDownload -Destination $BinaryDest -Force

    Write-Host "Extracting WebUI assets to $UiDir..."
    New-Item -ItemType Directory -Path $UiDir -Force | Out-Null

    if (Test-Path $UiDir) {
        Get-ChildItem -Path $UiDir -Force | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
    }

    & tar -xzf $WebUiDownload -C $UiDir
    if ($LASTEXITCODE -ne 0) {
        throw "tar extraction failed with exit code $LASTEXITCODE"
    }

    Write-Host "Updating PATH ($PathTarget scope)..."
    Add-InstallPath -Directory $InstallDir -Target $PathTarget
    Sync-SessionPath

    Write-Host ''
    Write-Host 'Installation complete.'
    Write-Host "  Version:  $tag"
    Write-Host "  Binary:   $BinaryDest"
    Write-Host "  WebUI:    $UiDir\"
    Write-Host "  Command:  arx (via PATH)"
    Write-Host ''
    Write-Host 'Next steps:'
    Write-Host "  arx server config init --config $(Join-Path $InstallDir 'server.yaml')"
    Write-Host "  arx server start --config $(Join-Path $InstallDir 'server.yaml')"
} finally {
    if (Test-Path $TempDir) {
        Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
