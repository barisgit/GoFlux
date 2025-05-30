# GoFlux CLI Installation Script for Windows
# Usage: iwr -useb https://raw.githubusercontent.com/barisgit/goflux/main/scripts/install.ps1 | iex

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\GoFlux\bin",
    [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"

$Repo = "barisgit/goflux"
$BinaryName = "flux.exe"

Write-Host "🚀 GoFlux CLI Installer for Windows" -ForegroundColor Blue
Write-Host ""

# Detect architecture
$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$OS = "windows"

Write-Host "Detected: $OS/$Arch" -ForegroundColor Yellow

# Get latest release version if not specified
if ($Version -eq "latest") {
    Write-Host "📡 Fetching latest release..." -ForegroundColor Yellow
    try {
        $LatestRelease = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        $Version = $LatestRelease.tag_name
    } catch {
        Write-Host "❌ Failed to get latest version: $_" -ForegroundColor Red
        exit 1
    }
}

Write-Host "Version: $Version" -ForegroundColor Green

# Construct download URL
$FileName = "flux_$($Version.TrimStart('v'))_${OS}_${Arch}.zip"
$DownloadUrl = "https://github.com/$Repo/releases/download/$Version/$FileName"

Write-Host "📥 Downloading $BinaryName $Version..." -ForegroundColor Yellow
Write-Host "URL: $DownloadUrl" -ForegroundColor Gray

# Create temporary directory
$TempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -Type Directory -Path $_ }
$ArchivePath = Join-Path $TempDir $FileName

try {
    # Download the archive
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ArchivePath

    # Extract the archive
    Write-Host "📦 Extracting archive..." -ForegroundColor Yellow
    Expand-Archive -Path $ArchivePath -DestinationPath $TempDir -Force

    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -Type Directory -Path $InstallDir -Force | Out-Null
    }

    # Copy binary to install directory
    $BinaryPath = Join-Path $TempDir "flux.exe"
    $DestPath = Join-Path $InstallDir $BinaryName
    
    Copy-Item -Path $BinaryPath -Destination $DestPath -Force

    Write-Host "✅ GoFlux CLI installed successfully!" -ForegroundColor Green
    Write-Host "📍 Location: $DestPath" -ForegroundColor Gray

    # Add to PATH if not already there
    $CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($CurrentPath -notlike "*$InstallDir*") {
        Write-Host "🔧 Adding $InstallDir to PATH..." -ForegroundColor Yellow
        [Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$InstallDir", "User")
        Write-Host "✅ Added to PATH. Restart your terminal or run:" -ForegroundColor Green
        Write-Host "   `$env:PATH += ';$InstallDir'" -ForegroundColor Gray
    }

    Write-Host ""
    Write-Host "🎯 Quick Start:" -ForegroundColor Blue
    Write-Host "  flux new my-app"
    Write-Host "  cd my-app"
    Write-Host "  flux dev"
    Write-Host ""
    Write-Host "📖 Documentation:" -ForegroundColor Blue
    Write-Host "  https://github.com/$Repo"

} catch {
    Write-Host "❌ Installation failed: $_" -ForegroundColor Red
    exit 1
} finally {
    # Cleanup
    Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host ""
Write-Host "To verify installation, restart your terminal and run: flux --version" -ForegroundColor Yellow 