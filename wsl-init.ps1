#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Initialize WSL2 and verify it's ready for CrateOS builds

.DESCRIPTION
    Checks WSL2 installation, starts the VM, verifies Ubuntu distro exists,
    and confirms build tools are available.

.EXAMPLE
    .\wsl-init.ps1
    # Outputs status of WSL, Ubuntu, and required build tools
#>

param(
    [switch]$Quiet  # Suppress non-error output
)

$ErrorActionPreference = "Continue"
$warningCount = 0
$errorCount = 0

function Write-Status {
    param([string]$Message, [ValidateSet("OK", "WARN", "ERROR")]$Type = "OK")
    
    if ($Quiet -and $Type -eq "OK") { return }
    
    $color = @{"OK" = "Green"; "WARN" = "Yellow"; "ERROR" = "Red"}[$Type]
    Write-Host "[" -NoNewline
    Write-Host $Type -ForegroundColor $color -NoNewline
    Write-Host "] $Message"
    
    if ($Type -eq "WARN") { $script:warningCount++ }
    if ($Type -eq "ERROR") { $script:errorCount++ }
}

Write-Host "==> CrateOS WSL2 Initialization Check"
Write-Host ""

# 1. Check if wsl.exe exists
Write-Host "1. Checking WSL2 installation..."
if (Get-Command wsl.exe -ErrorAction SilentlyContinue) {
    Write-Status "wsl.exe found" OK
} else {
    Write-Status "wsl.exe NOT found - install WSL2: https://aka.ms/wsl" ERROR
    exit 1
}

# 2. Check WSL version
Write-Host "2. Checking WSL version..."
$wslVersion = & wsl.exe --version 2>&1 | Select-String "WSL version"
if ($wslVersion) {
    Write-Status "$wslVersion" OK
} else {
    Write-Status "Could not detect WSL version (may be WSL1)" WARN
}

# 3. Try to start WSL and list distros
Write-Host "3. Checking installed distros..."
$distros = & wsl.exe --list --quiet 2>&1
if ($LASTEXITCODE -eq 0 -and $distros) {
    $distroArray = $distros | Where-Object { $_ -and $_ -ne "" }
    if ($distroArray.Count -gt 0) {
        foreach ($distro in $distroArray) {
            Write-Status "Found distro: $distro" OK
        }
    } else {
        Write-Status "No distros installed - install Ubuntu: wsl --install -d Ubuntu" ERROR
        exit 1
    }
} else {
    Write-Status "Could not list distros - WSL may need initialization" WARN
}

# 4. Verify WSL can start
Write-Host "4. Testing WSL VM startup..."
$null = & wsl.exe sh -lc "exit 0" 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Status "WSL started successfully" OK
} else {
    Write-Status "WSL failed to start" ERROR
    Write-Host "  Try: wsl --shutdown (then retry)" -ForegroundColor Yellow
    exit 1
}

# 5. Check P: drive mount in WSL
Write-Host "5. Checking drive access in WSL..."
$wslPath = & wsl.exe wslpath -a "P:\" 2>&1
if ($LASTEXITCODE -eq 0 -and $wslPath) {
    Write-Status "P: drive mounted as: $($wslPath.Trim())" OK
} else {
    Write-Status "P: drive not accessible in WSL" WARN
    Write-Host "  If P: is a network drive, it may need to be mounted in WSL" -ForegroundColor Yellow
}

# 6. Check required build tools in WSL
Write-Host "6. Checking WSL build tools..."
$tools = @("wget", "7z", "xorriso", "dpkg")
foreach ($tool in $tools) {
    $null = & wsl.exe sh -lc "command -v $tool" 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Status "$tool installed" OK
    } else {
        Write-Status "$tool NOT found" WARN
    }
}

# 7. Check Go in Windows
Write-Host "7. Checking Go installation..."
if (Get-Command go -ErrorAction SilentlyContinue) {
    $goVersion = & go version
    Write-Status "$goVersion" OK
} else {
    Write-Status "Go NOT installed - install from https://go.dev/dl/" ERROR
    exit 1
}

# Summary
Write-Host ""
Write-Host "==> Summary"
if ($errorCount -eq 0 -and $warningCount -eq 0) {
    Write-Host "✓ All checks passed. Ready to build!" -ForegroundColor Green
    exit 0
} elseif ($errorCount -eq 0) {
    Write-Host "⚠ $warningCount warning(s). Build may work but check above." -ForegroundColor Yellow
    exit 0
} else {
    Write-Host "✗ $errorCount error(s) found. Fix above before building." -ForegroundColor Red
    exit 1
}
