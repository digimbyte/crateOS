param(
    [Parameter(Position=0)]
    [ValidateSet("all","build","build-x86","build-rpi","build-rpi0","deb","deb-x86","deb-rpi","deb-rpi0","iso","image-x86","image-rpi","image-rpi0","qcow2","rpi","clean","help","check")]
    [string]$Target = "build",
    
    [Parameter()]
    [ValidateSet("x86", "rpi", "rpi0")]
    [string]$Platform = "x86"
)

$ErrorActionPreference = "Stop"
$WarningPreference = "Continue"

# Determine GOOS/GOARCH based on platform
$GOOS = "linux"
switch ($Platform) {
    "x86"  { $GOARCH = "amd64"; $VersionDefault = "0.1.0+noble1" }
    "rpi"  { $GOARCH = "arm64"; $VersionDefault = "0.1.0+rpi1" }
    "rpi0" { $GOARCH = "arm64"; $VersionDefault = "0.1.0+rpi0-1" }
}

$Version  = if ($env:VERSION) { $env:VERSION } else { $VersionDefault }
$Dist     = Join-Path $PSScriptRoot "dist"
$BinDir   = Join-Path $Dist "bin"
$Commands = @("crateos", "crateos-agent", "crateos-policy")
function Test-RebootRequired {
    <#
    .SYNOPSIS
    Detects if Windows requires a reboot (checks pending reboot registry keys).
    Returns $true if reboot is needed, $false otherwise.
    #>
    # Pending reboot can be surfaced through these registry markers.
    if (Test-Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending") {
        return $true
    }
    if (Test-Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired") {
        return $true
    }

    try {
        $sessionManager = Get-ItemProperty "HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager" -ErrorAction SilentlyContinue
        if ($sessionManager -and $sessionManager.PendingFileRenameOperations) {
            return $true
        }
    } catch {}

    return $false
}

function Test-Prerequisites {
    <#
    .SYNOPSIS
    Validates all build prerequisites before attempting any build operations.
    Returns $true if all checks pass, $false otherwise.
    #>
    param(
        [ValidateSet("all", "windows", "wsl")]
        [string]$Level = "all"
    )
    
    $allPass = $true
    
    # Windows prerequisites
    if ($Level -in @("all", "windows")) {
        Write-Host "Checking Windows prerequisites..."
        
        if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
            Write-Error "Go is NOT installed. Install from https://go.dev/dl/"
            $allPass = $false
        } else {
            $goVersion = & go version
            Write-Host "  ✓ $goVersion"
        }
        
        if (-not (Get-Command powershell -ErrorAction SilentlyContinue)) {
            Write-Error "PowerShell is NOT in PATH (unexpected)"
            $allPass = $false
        } else {
            Write-Host "  ✓ PowerShell $($PSVersionTable.PSVersion)"
        }
    }
    
    # WSL prerequisites
    if ($Level -in @("all", "wsl")) {
        Write-Host "Checking WSL prerequisites..."
        
        if (-not (Get-Command wsl.exe -ErrorAction SilentlyContinue)) {
            Write-Error "WSL2 is NOT installed. Install from https://aka.ms/wsl"
            $allPass = $false
            return $false
        }
        Write-Host "  ✓ WSL2 installed"
        
        # Check Ubuntu distro is installed (auto-install if missing)
        $distros = & wsl.exe --list --quiet 2>&1
        if ($LASTEXITCODE -ne 0 -or -not $distros) {
            Write-Host "  ⚠ Cannot list WSL distros. Attempting Ubuntu install..."
            & wsl.exe --install Ubuntu --no-launch 2>&1 | Out-Null
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed to install Ubuntu automatically. Install manually: wsl --install -d Ubuntu"
                $allPass = $false
                return $false
            }
            Write-Host "  ✓ Ubuntu installed"
            if (Test-RebootRequired) {
                Write-Warning "REBOOT IS REQUIRED: Windows features were installed and require a reboot before Ubuntu will appear in WSL distros."
                Write-Warning "Please reboot and then re-run: .\build.ps1 $Target"
                $allPass = $false
                return $false
            }
            Start-Sleep -Seconds 2
            $distros = & wsl.exe --list --quiet 2>&1
            if ($LASTEXITCODE -ne 0 -or -not $distros) {
                Write-Error "Ubuntu install completed, but WSL distros still unavailable."
                $allPass = $false
                return $false
            }
        }
        
        $ubuntuFound = $false
        $ubuntuDistro = $null
        $distroArray = @($distros | Where-Object { $_ -and $_ -ne "" })
        foreach ($distro in $distroArray) {
            if ($distro -match "Ubuntu|ubuntu") {
                Write-Host "  ✓ Found Ubuntu distro: $distro"
                $ubuntuFound = $true
                $ubuntuDistro = $distro.Trim()
                break
            }
        }
        
        if (-not $ubuntuFound) {
            Write-Host "  ⚠ No Ubuntu distro found. Attempting Ubuntu install..."
            & wsl.exe --install Ubuntu --no-launch 2>&1 | Out-Null
            if ($LASTEXITCODE -ne 0) {
                Write-Error "No Ubuntu distro found and automatic install failed."
                Write-Error "Available distros: $($distroArray -join ', ')"
                Write-Error "Install manually: wsl --install -d Ubuntu"
                $allPass = $false
                return $false
            }
            Write-Host "  ✓ Ubuntu installed"
            if (Test-RebootRequired) {
                Write-Warning "REBOOT IS REQUIRED: Windows features were installed and require a reboot before Ubuntu will appear in WSL distros."
                Write-Warning "Please reboot and then re-run: .\build.ps1 $Target"
                $allPass = $false
                return $false
            }
            Start-Sleep -Seconds 2
            $distros = & wsl.exe --list --quiet 2>&1
            $distroArray = @($distros | Where-Object { $_ -and $_ -ne "" })
            $ubuntuFound = $false
            foreach ($distro in $distroArray) {
                if ($distro -match "Ubuntu|ubuntu") {
                    Write-Host "  ✓ Found Ubuntu distro: $distro"
                    $ubuntuFound = $true
                    $ubuntuDistro = $distro.Trim()
                    break
                }
            }
            if (-not $ubuntuFound) {
                if (Test-RebootRequired) {
                    Write-Warning "REBOOT IS REQUIRED: Ubuntu install was attempted, but Windows has pending reboot state."
                    Write-Warning "Please reboot and then re-run: .\build.ps1 $Target"
                } else {
                    Write-Error "Ubuntu installation was attempted, but Ubuntu is still not listed in WSL distros."
                }
                $allPass = $false
                return $false
            }
        }
        
        # Check WSL can start using Ubuntu distro
        $null = & wsl.exe -d $ubuntuDistro --exec /bin/sh -lc "exit 0" 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Host "  ⚠ WSL2 VM failed to start. Attempting recovery (wsl --shutdown)..."
            & wsl.exe --shutdown 2>&1 | Out-Null
            Start-Sleep -Seconds 2
            
            # Retry WSL startup
            $null = & wsl.exe -d $ubuntuDistro --exec /bin/sh -lc "exit 0" 2>&1
            if ($LASTEXITCODE -ne 0) {
                Write-Error "WSL2 VM still cannot start after recovery attempt."
                $allPass = $false
                return $false
            } else {
                Write-Host "  ✓ WSL2 VM recovered and started successfully"
            }
        } else {
            Write-Host "  ✓ WSL2 VM starts successfully"
        }
        
        # Check required build tools in WSL
        $requiredTools = @("wget", "7z", "xorriso", "dpkg", "make")
        $missingTools = @()
        foreach ($tool in $requiredTools) {
            $null = & wsl.exe -d $ubuntuDistro --exec /bin/sh -lc "command -v $tool" 2>&1
            if ($LASTEXITCODE -ne 0) {
                $missingTools += $tool
            }
        }
        
        if ($missingTools.Count -gt 0) {
            Write-Error "Missing build tools in WSL: $($missingTools -join ', ')"
            Write-Error "Install with: wsl -d Ubuntu -- sudo apt-get install -y $($missingTools -join ' ')"
            $allPass = $false
        } else {
            Write-Host "  ✓ All required build tools present (wget, 7z, xorriso, dpkg, make)"
        }
    }
    
    return $allPass
}

function Get-WslCommand {
    $wsl = Get-Command wsl.exe -ErrorAction SilentlyContinue
    if (-not $wsl) {
        Write-Error "WSL is not installed or not in PATH. Install WSL2/Ubuntu first."
        return $null
    }
    return $wsl.Source
}

function Get-WslRepoPath {
    # Ensure WSL can start before asking it to resolve paths.
    $null = & wsl.exe sh -lc "exit 0" 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  ⚠ WSL2 VM failed to start. Attempting recovery (wsl --shutdown)..."
        & wsl.exe --shutdown 2>&1 | Out-Null
        Start-Sleep -Seconds 2
        
        # Retry WSL startup
        $null = & wsl.exe sh -lc "exit 0" 2>$null
        if ($LASTEXITCODE -ne 0) {
            Write-Error "WSL failed to start even after recovery. Ensure WSL2 + an Ubuntu distro are installed and running."
            return $null
        }
    }

    # Try to get path from wslpath first
    $wslPath = & wsl.exe wslpath -a "$PSScriptRoot" 2>$null
    if ($LASTEXITCODE -eq 0 -and -not [string]::IsNullOrWhiteSpace($wslPath)) {
        return $wslPath.Trim()
    }

    # Fallback for drives that might not be auto-mounted (e.g. network/virtual drives)
    if ($PSScriptRoot -match "^([A-Za-z]):\\(.*)") {
        $drive = $Matches[1].ToLower()
        $rest = $Matches[2].Replace('\', '/')
        $fallbackPath = "/mnt/$drive/$rest"
        
        # Verify if the fallback path exists in WSL
        & wsl.exe sh -lc "test -d '$fallbackPath'" 2>$null
        if ($LASTEXITCODE -eq 0) {
            return $fallbackPath
        }
    }
    Write-Error "Failed to convert repo path for WSL: $PSScriptRoot. WSL cannot access this drive. Possible causes: 1) WSL not fully initialized; 2) Drive not mounted in WSL; 3) Ubuntu distro not installed. Run 'wsl --install' and try again."
    return $null
}

function Invoke-WslMakeTarget {
    param(
        [Parameter(Mandatory=$true)]
        [string]$MakeTarget
    )

    $wsl = Get-WslCommand
    if (-not $wsl) {
        return
    }

    $repoPath = Get-WslRepoPath
    if (-not $repoPath) {
        return
    }

    $singleQuoteEscape = "'" + '"' + "'" + '"' + "'"
    $versionValue = $Version.Replace("'", $singleQuoteEscape)
    $repoValue = $repoPath.Replace("'", $singleQuoteEscape)
    $targetValue = $MakeTarget.Replace("'", $singleQuoteEscape)
    $linuxCommand = "cd '$repoValue' && VERSION='$versionValue' make $targetValue"

    Write-Host "==> delegating '$MakeTarget' to WSL in $repoPath"
    & $wsl bash -lc $linuxCommand
    if ($LASTEXITCODE -ne 0) {
        Write-Error "WSL make $MakeTarget failed"
    }
}

function Invoke-Build {
    if (-not (Test-Prerequisites -Level windows)) {
        exit 1
    }
    
    # Validate platform configuration
    if (-not $GOOS -or -not $GOARCH) {
        Write-Error "Platform not properly configured. GOOS=$GOOS, GOARCH=$GOARCH"
        exit 1
    }

    New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
    
    Write-Host ""
    Write-Host "==> Building CrateOS binaries"
    Write-Host "    Platform: $Platform"
    Write-Host "    GOOS=$GOOS GOARCH=$GOARCH"
    Write-Host "    Version: $Version"
    Write-Host ""

    foreach ($cmd in $Commands) {
        $src = "./cmd/$cmd"
        $out = Join-Path $BinDir "$cmd.exe"
        Write-Host "==> building $cmd (platform=$Platform)"
        
        $env:GOOS=$GOOS
        $env:GOARCH=$GOARCH
        $env:CGO_ENABLED=0
        
        go build -trimpath `
            -ldflags "-X github.com/crateos/crateos/internal/platform.Version=$Version -X github.com/crateos/crateos/internal/platform.BuildTarget=$Platform" `
            -o $out $src
        
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to build $cmd for $Platform"
            exit 1
        }
    }
    Write-Host ""
    Write-Host "==> all binaries built for $Platform in $BinDir"
    exit 0
}

function Invoke-Deb {
    if (-not (Test-Prerequisites -Level all)) {
        exit 1
    }
    $target = "deb-$Platform"
    Write-Host "==> Building .deb packages for $Platform"
    Invoke-WslMakeTarget $target
}

function Invoke-Iso {
    if (-not (Test-Prerequisites -Level all)) {
        exit 1
    }
    if ($Platform -ne "x86") {
        Write-Error 'ISO images are x86-only. Use: . \build.ps1 -Platform x86 image-x86'
        exit 1
    }
    Write-Host "==> Building ISO image for x86"
    Invoke-WslMakeTarget "image-x86"
}

function Invoke-Qcow2 {
    if (-not (Test-Prerequisites -Level all)) {
        exit 1
    }
    if ($Platform -ne "x86") {
        Write-Error 'QCOW2 images are x86-only. Use: . \build.ps1 -Platform x86 qcow2'
        exit 1
    }
    Write-Host "==> Building QCOW2 image for x86"
    Invoke-WslMakeTarget "qcow2"
}

function Invoke-Rpi {
    Write-Warning "rpi target not yet implemented."
}

function Invoke-Clean {
    if (Test-Path $Dist) {
        Remove-Item -Recurse -Force $Dist
        Write-Host "==> cleaned $Dist"
    } else {
        Write-Host "==> nothing to clean"
    }
}

function Invoke-Check {
    Write-Host "CrateOS Build Prerequisite Check"
    Write-Host ""
    if (Test-Prerequisites -Level all) {
        Write-Host "✓ All prerequisites satisfied. Ready to build." -ForegroundColor Green
        exit 0
    } else {
        Write-Host "✗ Prerequisites not met. See errors above." -ForegroundColor Red
        exit 1
    }
}

function Invoke-All {
    <#
    .SYNOPSIS
    Build all platforms in sequence: x86, rpi, rpi0
    Each platform: build → deb → image
    #>
    Write-Host ""
    Write-Host "╔════════════════════════════════════════════════════════════════╗"
    Write-Host "║         CrateOS Multi-Platform Build (All Platforms)          ║"
    Write-Host "╚════════════════════════════════════════════════════════════════╝"
    Write-Host ""
    
    # x86 platform
    Write-Host "[1/3] Building x86-64 (Ubuntu 24.04)..."
    Build-Platform "x86" "deb-x86" "image-x86"
    if ($LASTEXITCODE -ne 0) { exit 1 }
    Write-Host "✓ x86 complete: crateos-0.1.0+noble1.iso" -ForegroundColor Green
    Write-Host ""
    
    # RPi platform
    Write-Host "[2/3] Building Raspberry Pi 4/5 (RPi OS)..."
    Build-Platform "rpi" "deb-rpi" "image-rpi"
    if ($LASTEXITCODE -ne 0) { exit 1 }
    Write-Host "✓ RPi complete: crateos-rpi-0.1.0+rpi1.img" -ForegroundColor Green
    Write-Host ""
    
    # RPi Zero 2 platform
    Write-Host "[3/3] Building Raspberry Pi Zero 2 W (RPi OS Lite)..."
    Build-Platform "rpi0" "deb-rpi0" "image-rpi0"
    if ($LASTEXITCODE -ne 0) { exit 1 }
    Write-Host "✓ RPi0 complete: crateos-rpi0-0.1.0+rpi0-1.img" -ForegroundColor Green
    Write-Host ""
    
    Write-Host "╔════════════════════════════════════════════════════════════════╗"
    Write-Host "║                  ✓ All platforms built successfully!           ║"
    Write-Host "╚════════════════════════════════════════════════════════════════╝"
    Write-Host ""
    Write-Host "Images ready in dist/:"
    Write-Host "  • crateos-0.1.0+noble1.iso         (x86, Ubuntu)"
    Write-Host "  • crateos-rpi-0.1.0+rpi1.img       (RPi 4/5)"
    Write-Host "  • crateos-rpi0-0.1.0+rpi0-1.img    (RPi Zero 2)"
    Write-Host ""
}

function Build-Platform {
    <#
    .SYNOPSIS
    Helper function to build a single platform with its associated targets.
    Sets up platform globals, then delegates to the appropriate functions.
    #>
    param(
        [Parameter(Mandatory=$true)]
        [string]$PlatformName,
        
        [Parameter(Mandatory=$true)]
        [string]$DebTarget,
        
        [Parameter(Mandatory=$true)]
        [string]$ImageTarget
    )
    
    # Set global platform variables
    $script:Platform = $PlatformName
    switch ($PlatformName) {
        "x86"  { $script:GOARCH = "amd64"; $script:VersionDefault = "0.1.0+noble1" }
        "rpi"  { $script:GOARCH = "arm64"; $script:VersionDefault = "0.1.0+rpi1" }
        "rpi0" { $script:GOARCH = "arm64"; $script:VersionDefault = "0.1.0+rpi0-1" }
    }
    
    Invoke-Build
    if ($LASTEXITCODE -ne 0) { return }
    
    # Delegate to WSL for deb and image targets
    Invoke-WslMakeTarget $DebTarget
    if ($LASTEXITCODE -ne 0) { return }
    
    Invoke-WslMakeTarget $ImageTarget
    if ($LASTEXITCODE -ne 0) { return }
}

function Invoke-Help {
    Write-Host "CrateOS build script"
    Write-Host ""
    Write-Host "Usage: .\build.ps1 <target>"
    Write-Host ""
    Write-Host "Targets:"
    Write-Host "  all     Build all platforms (x86, rpi, rpi0)"
    Write-Host "  check   Verify all prerequisites are installed"
    Write-Host "  build   Compile Go binaries (default)"
    Write-Host "  deb     Build .deb packages via WSL2"
    Write-Host "  iso     Build autoinstall ISO via WSL2"
    Write-Host "  qcow2   Build VM image via WSL2"
    Write-Host "  rpi     Build Pi image (stub)"
    Write-Host "  clean   Remove dist/"
    Write-Host "  help    Show this message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host '  . \build.ps1 all           # Build everything (all platforms)'
    Write-Host '  . \build.ps1 check         # Verify system is ready'
    Write-Host '  . \build.ps1 build         # Compile binaries'
    Write-Host '  . \build.ps1 deb           # Create .deb packages (requires WSL2)'
    Write-Host '  . \build.ps1 iso           # Build bootable ISO (requires WSL2)'
    Write-Host ""
    Write-Host "Prerequisites:"
    Write-Host "  - Windows: Go 1.20+, PowerShell 5.0+"
    Write-Host "  - WSL2: Ubuntu distro installed and running"
    Write-Host "  - WSL Ubuntu: wget, p7zip-full, xorriso, dpkg, make"
    Write-Host ""
    Write-Host 'Use ". \build.ps1 check" to diagnose issues.'
}

switch ($Target) {
    "all"           { Invoke-All }
    "check"         { Invoke-Check }
    "build"         { Invoke-Build }
    "build-x86"     { $Platform="x86"; Invoke-Build }
    "build-rpi"     { $Platform="rpi"; Invoke-Build }
    "build-rpi0"    { $Platform="rpi0"; Invoke-Build }
    "deb"           { Invoke-Deb }
    "deb-x86"       { $Platform="x86"; Invoke-Deb }
    "deb-rpi"       { $Platform="rpi"; Invoke-Deb }
    "deb-rpi0"      { $Platform="rpi0"; Invoke-Deb }
    "iso"           { Invoke-Iso }
    "image-x86"     { $Platform="x86"; Invoke-Iso }
    "image-rpi"     { $Platform="rpi"; Invoke-WslMakeTarget "image-rpi" }
    "image-rpi0"    { $Platform="rpi0"; Invoke-WslMakeTarget "image-rpi0" }
    "qcow2"         { Invoke-Qcow2 }
    "rpi"           { $Platform="rpi"; Invoke-WslMakeTarget "image-rpi" }
    "clean"         { Invoke-Clean }
    "help"          { Invoke-Help }
}
