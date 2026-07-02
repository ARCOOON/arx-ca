# uninstall.ps1 — Remove arx-ca binary, WebUI assets, and PATH entry for the selected scope.
# Usage: .\uninstall.ps1 [-User] [-System]   (default: -User)

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

function Test-Administrator {
    $identity = [Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()
    return $identity.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Remove-InstallPath {
    param(
        [string]$Directory,
        [EnvironmentVariableTarget]$Target
    )

    $normalized = $Directory.TrimEnd('\')
    $current = [Environment]::GetEnvironmentVariable('Path', $Target)

    if ([string]::IsNullOrWhiteSpace($current)) {
        return
    }

    $segments = $current -split ';' | Where-Object {
        $_.Trim() -ne '' -and $_.TrimEnd('\') -ne $normalized
    }

    $newPath = ($segments -join ';').TrimEnd(';')
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

function Remove-IfExists {
    param([string]$Path)

    if (Test-Path $Path) {
        Remove-Item -Path $Path -Recurse -Force
        Write-Host "Removed $Path"
    }
}

if ($System) {
    if (-not (Test-Administrator)) {
        Write-Error 'Error: -System requires Administrator privileges. Run PowerShell as Administrator.'
        exit 1
    }
    $InstallDir = Join-Path $env:ProgramFiles 'arx-ca'
    $PathTarget = [EnvironmentVariableTarget]::Machine
    $ScopeLabel = 'system'
} else {
    $InstallDir = Join-Path $env:LOCALAPPDATA 'arx-ca'
    $PathTarget = [EnvironmentVariableTarget]::User
    $ScopeLabel = 'user'
}

Write-Host "Uninstalling arx-ca ($ScopeLabel scope)"
Write-Host "  Install directory: $InstallDir"

Remove-IfExists (Join-Path $InstallDir 'arx-ca.exe')
Remove-IfExists (Join-Path $InstallDir 'ui')

Write-Host "Removing PATH entry ($PathTarget scope)..."
Remove-InstallPath -Directory $InstallDir -Target $PathTarget
Sync-SessionPath

$preserved = $false
$serverYaml = Join-Path $InstallDir 'server.yaml'
$pkiDir = Join-Path $InstallDir '.pki'

if (Test-Path $serverYaml) {
    $preserved = $true
}
if (Test-Path $pkiDir) {
    $preserved = $true
}

if ($preserved) {
    Write-Host ''
    Write-Host "Preserved data in ${InstallDir}:"
    if (Test-Path $serverYaml) {
        Write-Host '  - server.yaml'
    }
    if (Test-Path $pkiDir) {
        Write-Host '  - .pki/'
    }
    Write-Host "Install directory retained: $InstallDir"
} elseif (Test-Path $InstallDir) {
    $remaining = Get-ChildItem -Path $InstallDir -Force -ErrorAction SilentlyContinue
    if (-not $remaining) {
        Remove-Item -Path $InstallDir -Force
        Write-Host "Removed empty install directory $InstallDir"
    } else {
        Write-Host "Install directory $InstallDir is not empty; retained remaining files."
    }
} else {
    Write-Host "Install directory not found: $InstallDir (nothing to remove)"
}

Write-Host ''
Write-Host 'Uninstall complete.'
