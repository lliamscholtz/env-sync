#Requires -Version 5.0

[CmdletBinding()]
param(
    [switch]$Force,
    [switch]$SkipTilt
)

Write-Host "🔧 Installing env-sync dependencies..." -ForegroundColor Green

# Check if running as administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Install package manager if needed
function Install-PackageManager {
    # Try winget first (Windows 10 1809+)
    if (Get-Command winget -ErrorAction SilentlyContinue) {
        return "winget"
    }

    # Try chocolatey
    if (Get-Command choco -ErrorAction SilentlyContinue) {
        return "choco"
    }

    # Install chocolatey
    Write-Host "📦 Installing Chocolatey package manager..." -ForegroundColor Yellow
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

    return "choco"
}

# Install Azure CLI
function Install-AzureCLI {
    $packageManager = Install-PackageManager

    Write-Host "📦 Installing Azure CLI using $packageManager..." -ForegroundColor Yellow

    switch ($packageManager) {
        "winget" {
            winget install Microsoft.AzureCLI
        }
        "choco" {
            choco install azure-cli -y
        }
        default {
            throw "No suitable package manager found"
        }
    }
}

# Install Tilt
function Install-Tilt {
    $packageManager = Install-PackageManager

    Write-Host "📦 Installing Tilt using $packageManager..." -ForegroundColor Yellow

    switch ($packageManager) {
        "winget" {
            # Tilt may not be available in winget, fall back to direct download
            Write-Host "⬇️ Downloading Tilt directly..." -ForegroundColor Yellow
            $url = "https://github.com/tilt-dev/tilt/releases/latest/download/tilt.windows.x86_64.exe"
            $output = "$env:USERPROFILE\bin\tilt.exe"
            New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\bin"
            Invoke-WebRequest -Uri $url -OutFile $output
            # Add to PATH if not already there
            $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
            if ($userPath -notlike "*$env:USERPROFILE\bin*") {
                [Environment]::SetEnvironmentVariable("PATH", "$userPath;$env:USERPROFILE\bin", "User")
            }
        }
        "choco" {
            choco install tilt -y
        }
    }
}

# Main installation logic
try {
    # Check Azure CLI
    if (-not (Get-Command az -ErrorAction SilentlyContinue) -or $Force) {
        Install-AzureCLI
    } else {
        Write-Host "✅ Azure CLI already installed" -ForegroundColor Green
    }

    # Check Tilt (optional)
    if (-not $SkipTilt -and (-not (Get-command tilt -ErrorAction SilentlyContinue) -or $Force)) {
        $installTilt = Read-Host "📦 Install Tilt for development workflow integration? (y/N)"
        if ($installTilt -eq "y" -or $installTilt -eq "Y") {
            Install-Tilt
        }
    } elseif (-not $SkipTilt) {
        Write-Host "✅ Tilt already installed" -ForegroundColor Green
    }

    Write-Host "🎉 Dependency installation complete!" -ForegroundColor Green
    Write-Host "💡 Next steps:" -ForegroundColor Cyan
    Write-Host "   1. Restart your terminal to refresh PATH" -ForegroundColor Cyan
    Write-Host "   2. Run 'az login' to authenticate with Azure" -ForegroundColor Cyan
    Write-Host "   3. Run 'env-sync doctor' to verify installation" -ForegroundColor Cyan

} catch {
    Write-Host "❌ Installation failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "💡 Try running as Administrator or install dependencies manually" -ForegroundColor Yellow
    exit 1
} 