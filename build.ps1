param(
    [Parameter(Position=0)]
    [ValidateSet("prompt","all","deb","deb-x86","deb-rpi","deb-rpi0","image","iso","image-x86","image-rpi","image-rpi0","qcow2","rpi","clean","help","check")]
    [string]$Target = "prompt",
    
    [Parameter()]
    [ValidateSet("x86", "rpi", "rpi0")]
    [string]$Platform = "x86",

    [Parameter()]
    [ValidateSet("iso","img","qcow2")]
    [string]$Format
)

$ErrorActionPreference = "Stop"
$WarningPreference = "Continue"

# Determine GOOS/GOARCH based on platform
$GOOS = "linux"

function Set-PlatformState {
    param(
        [Parameter(Mandatory=$true)]
        [ValidateSet("x86", "rpi", "rpi0")]
        [string]$PlatformName
    )

    $script:Platform = $PlatformName
    $script:GOOS = "linux"

    switch ($PlatformName) {
        "x86"  { $script:GOARCH = "amd64"; $script:VersionDefault = "0.1.0+noble1" }
        "rpi"  { $script:GOARCH = "arm64"; $script:VersionDefault = "0.1.0+rpi1" }
        "rpi0" { $script:GOARCH = "arm64"; $script:VersionDefault = "0.1.0+rpi0-1" }
    }

    $script:Version = if ($env:VERSION) { $env:VERSION } else { $script:VersionDefault }
}

function Get-DefaultImageFormat {
    param(
        [Parameter(Mandatory=$true)]
        [ValidateSet("x86", "rpi", "rpi0")]
        [string]$PlatformName
    )

    switch ($PlatformName) {
        "x86"  { return "iso" }
        "rpi"  { return "img" }
        "rpi0" { return "img" }
    }
}

function Resolve-ImageMakeTarget {
    param(
        [Parameter(Mandatory=$true)]
        [ValidateSet("x86", "rpi", "rpi0")]
        [string]$PlatformName,

        [Parameter()]
        [string]$ImageFormat
    )

    $resolvedFormat = if ([string]::IsNullOrWhiteSpace($ImageFormat)) { Get-DefaultImageFormat -PlatformName $PlatformName } else { $ImageFormat }
    switch ("${PlatformName}:$resolvedFormat") {
        "x86:iso"  { return @{ MakeTarget = "image-x86"; Format = "iso"; Label = "ISO image" } }
        "x86:qcow2" { return @{ MakeTarget = "qcow2"; Format = "qcow2"; Label = "QCOW2 image" } }
        "rpi:img"  { return @{ MakeTarget = "image-rpi"; Format = "img"; Label = "Raspberry Pi image" } }
        "rpi0:img" { return @{ MakeTarget = "image-rpi0"; Format = "img"; Label = "Raspberry Pi Zero 2 image" } }
        default {
            Write-Error "Unsupported image format '$resolvedFormat' for platform '$PlatformName'"
        }
    }
}

Set-PlatformState -PlatformName $Platform
$Dist     = Join-Path $PSScriptRoot "dist"
$BinDir   = Join-Path $Dist "bin"
$Commands = @("crateos", "crateos-agent", "crateos-policy")
$script:UbuntuDistro = $null

function Write-BlankLine {
    Write-Host ""
}

function Write-Banner {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Title
    )

    $width = 64
    $safeTitle = " $Title "
    if ($safeTitle.Length -gt ($width - 2)) {
        $safeTitle = $safeTitle.Substring(0, $width - 5) + "... "
    }

    $padding = [Math]::Max(0, $width - 2 - $safeTitle.Length)
    $left = [Math]::Floor($padding / 2)
    $right = $padding - $left

    Write-BlankLine
    Write-Host ("╔" + ("═" * ($width - 2)) + "╗") -ForegroundColor DarkCyan
    Write-Host ("║" + (" " * $left) + $safeTitle + (" " * $right) + "║") -ForegroundColor DarkCyan
    Write-Host ("╚" + ("═" * ($width - 2)) + "╝") -ForegroundColor DarkCyan
    Write-BlankLine
}

function Write-Section {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Title
    )

    Write-Host $Title -ForegroundColor Cyan
}

function Write-Step {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Message
    )

    Write-Host "  • $Message" -ForegroundColor Gray
}

function Write-Ok {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Message
    )

    Write-Host "  ✓ $Message" -ForegroundColor Green
}

function Write-WarnLine {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Message
    )

    Write-Host "  ⚠ $Message" -ForegroundColor Yellow
}

function Write-ErrorLine {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Message
    )

    Write-Host "  ✗ $Message" -ForegroundColor Red
}

function Write-CommandLine {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Command
    )

    Write-Host "    $Command" -ForegroundColor Cyan
}
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

function Normalize-WslText {
    param(
        [AllowNull()]
        [object]$Value
    )

    if ($null -eq $Value) {
        return $null
    }

    return ($Value.ToString() -replace "`0", "").Trim()
}

function Get-WslPackageName {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Tool
    )

    switch ($Tool) {
        "7z" { return "p7zip-full" }
        "go" { return "golang-go" }
        default { return $Tool }
    }
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
        Write-Section "Windows prerequisites"
        
        if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
            Write-ErrorLine "Go is not installed on Windows."
            Write-Step "Install from: https://go.dev/dl/"
            $allPass = $false
        } else {
            $goVersion = & go version
            Write-Ok $goVersion
        }
        
        if (-not (Get-Command powershell -ErrorAction SilentlyContinue)) {
            Write-ErrorLine "PowerShell is not in PATH."
            $allPass = $false
        } else {
            Write-Ok "PowerShell $($PSVersionTable.PSVersion)"
        }
    }
    
    # WSL prerequisites
    if ($Level -in @("all", "wsl")) {
        Write-Section "WSL prerequisites"
        
        if (-not (Get-Command wsl.exe -ErrorAction SilentlyContinue)) {
            Write-ErrorLine "WSL2 is not installed."
            Write-Step "Next step:"
            Write-CommandLine "wsl --install -d Ubuntu"
            Write-Step "Then reboot Windows and re-run:"
            Write-CommandLine ".\build.ps1 $Target"
            $allPass = $false
            return $false
        }
        Write-Ok "WSL2 installed"
        
        # Check Ubuntu distro is installed (normalize output first)
        $distros = & wsl.exe --list --quiet 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Host ""
            Write-Host "  ⚠ Failed to list WSL distros." -ForegroundColor Yellow
            Write-Host "  Run this command, then reboot:" -ForegroundColor Yellow
            Write-Host "    wsl --install -d Ubuntu" -ForegroundColor Cyan
            Write-Host "  Then re-run:" -ForegroundColor Yellow
            Write-Host "    .\build.ps1 $Target" -ForegroundColor Cyan
            $allPass = $false
            return $false
        }
        
        $ubuntuFound = $false
        $ubuntuDistro = $null
        $distroArray = @(
            $distros |
                ForEach-Object { Normalize-WslText $_ } |
                Where-Object { $_ -and $_ -ne "" }
        )
        foreach ($distro in $distroArray) {
            if ($distro.StartsWith("Ubuntu", [System.StringComparison]::OrdinalIgnoreCase)) {
                Write-Ok "Found Ubuntu distro: $distro"
                $ubuntuFound = $true
                $ubuntuDistro = $distro
                $script:UbuntuDistro = $ubuntuDistro
                break
            }
        }
        
        if (-not $ubuntuFound) {
            Write-Host "  ⚠ No Ubuntu distro found. Attempting to install..."
            $installOutput = & wsl.exe --install Ubuntu --no-launch 2>&1
            $installExitCode = $LASTEXITCODE
            
            # Check if install already-exists error (tolerated for idempotency)
            if ($installExitCode -ne 0) {
                # If error is "already exists", try to detect it (may need reboot)
                if ($installOutput -match "already exists") {
                    Write-Host "  ℹ Ubuntu already exists (may not be visible yet). Reboot may be needed." -ForegroundColor Cyan
                    if (Test-RebootRequired) {
                        Write-Warning "REBOOT IS REQUIRED: Windows features installed, need reboot before Ubuntu appears."
                        Write-Warning "Please reboot and then re-run: .\build.ps1 $Target"
                        $allPass = $false
                        return $false
                    }
                } else {
                    # Real failure
                    Write-Host ""
                    Write-Host "  ⚠ Automatic Ubuntu install failed." -ForegroundColor Yellow
                    Write-Host "  Available distros: $($distroArray -join ', ')" -ForegroundColor Yellow
                    Write-Host "  Next step (run this in elevated PowerShell):" -ForegroundColor Yellow
                    Write-Host "    wsl --install -d Ubuntu" -ForegroundColor Cyan
                    Write-Host "  Then reboot and re-run:" -ForegroundColor Yellow
                    Write-Host "    .\build.ps1 $Target" -ForegroundColor Cyan
                    $allPass = $false
                    return $false
                }
            } else {
                Write-Host "  ✓ Ubuntu installed"
                if (Test-RebootRequired) {
                    Write-Warning "REBOOT IS REQUIRED: Windows features were installed. Please reboot."
                    Write-Warning "After reboot, re-run: .\build.ps1 $Target"
                    $allPass = $false
                    return $false
                }
            }
            
            # Retry detection after install attempt
            Start-Sleep -Seconds 2
            $distros = & wsl.exe --list --quiet 2>&1
            if ($LASTEXITCODE -eq 0 -and $distros) {
                $distroArray = @(
                    $distros |
                        ForEach-Object { Normalize-WslText $_ } |
                        Where-Object { $_ -and $_ -ne "" }
                )
                foreach ($distro in $distroArray) {
                    if ($distro.StartsWith("Ubuntu", [System.StringComparison]::OrdinalIgnoreCase)) {
                        Write-Host "  ✓ Found Ubuntu distro after install: $distro"
                        $ubuntuFound = $true
                        $ubuntuDistro = $distro
                        $script:UbuntuDistro = $ubuntuDistro
                        break
                    }
                }
            }
            
            if (-not $ubuntuFound) {
                Write-Host "  ⚠ Ubuntu still not found after install attempt." -ForegroundColor Yellow
                Write-Host "  Available distros: $($distroArray -join ', ')" -ForegroundColor Yellow
                if (Test-RebootRequired) {
                    Write-Warning "REBOOT IS REQUIRED: Windows has pending reboot state."
                    Write-Warning "Please reboot and then re-run: .\build.ps1 $Target"
                } else {
                    Write-Host "  Next step (run in elevated PowerShell):" -ForegroundColor Yellow
                    Write-Host "    wsl --install -d Ubuntu" -ForegroundColor Cyan
                    Write-Host "  Then reboot and re-run: .\build.ps1 $Target" -ForegroundColor Yellow
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
                Write-Host "  ✗ WSL2 VM still cannot start after recovery attempt." -ForegroundColor Red
                Write-Host "  Next steps:" -ForegroundColor Yellow
                Write-Host "    1) Run: wsl --shutdown" -ForegroundColor Cyan
                Write-Host "    2) Launch Ubuntu once manually: wsl -d $ubuntuDistro" -ForegroundColor Cyan
                Write-Host "    3) Re-run: .\build.ps1 $Target" -ForegroundColor Cyan
                $allPass = $false
                return $false
            } else {
                Write-Host "  ✓ WSL2 VM recovered and started successfully"
            }
        } else {
            Write-Host "  ✓ WSL2 VM starts successfully"
        }
        
        # Check required build tools in WSL
        $requiredTools = @("wget", "7z", "xorriso", "dpkg", "make", "go")
        $missingTools = @()
        foreach ($tool in $requiredTools) {
            $null = & wsl.exe -d $ubuntuDistro --exec /bin/sh -lc "command -v $tool" 2>&1
            if ($LASTEXITCODE -ne 0) {
                $missingTools += $tool
            }
        }
        
        if ($missingTools.Count -gt 0) {
            $missingPackages = @($missingTools | ForEach-Object { Get-WslPackageName $_ })
            $packageList = $missingPackages -join ' '
            $repairCommand = "sudo apt-get update && sudo apt-get install -y $packageList"

            Write-WarnLine "Missing build tools in WSL: $($missingTools -join ', ')"
            Write-Step "Attempting automatic repair inside $ubuntuDistro..."
            Write-Host "    $repairCommand" -ForegroundColor DarkGray

            & wsl.exe -d $ubuntuDistro --exec /bin/sh -lc $repairCommand
            $repairExitCode = $LASTEXITCODE

            if ($repairExitCode -eq 0) {
                $remainingTools = @()
                foreach ($tool in $requiredTools) {
                    $null = & wsl.exe -d $ubuntuDistro --exec /bin/sh -lc "command -v $tool" 2>&1
                    if ($LASTEXITCODE -ne 0) {
                        $remainingTools += $tool
                    }
                }

                if ($remainingTools.Count -eq 0) {
                    Write-Ok "Missing WSL build tools were installed automatically"
                } else {
                    $remainingPackages = @($remainingTools | ForEach-Object { Get-WslPackageName $_ })
                    Write-Host "  ✗ Automatic repair completed, but some tools are still missing: $($remainingTools -join ', ')" -ForegroundColor Red
                    Write-Host "  Next step:" -ForegroundColor Yellow
                    Write-Host "    wsl -d $ubuntuDistro --exec /bin/sh -lc `"sudo apt-get update && sudo apt-get install -y $($remainingPackages -join ' ')`"" -ForegroundColor Cyan
                    Write-Host "  Then re-run:" -ForegroundColor Yellow
                    Write-Host "    .\build.ps1 $Target" -ForegroundColor Cyan
                    $allPass = $false
                }
            } else {
                Write-Host "  ✗ Automatic WSL package installation failed." -ForegroundColor Red
                Write-Host "  Next step:" -ForegroundColor Yellow
                Write-Host "    wsl -d $ubuntuDistro --exec /bin/sh -lc `"sudo apt-get update && sudo apt-get install -y $packageList`"" -ForegroundColor Cyan
                Write-Host "  Then re-run:" -ForegroundColor Yellow
                Write-Host "    .\build.ps1 $Target" -ForegroundColor Cyan
                $allPass = $false
            }
        } else {
            Write-Ok "All required build tools present (wget, 7z, xorriso, dpkg, make, go)"
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
    param(
        [string]$Distro = $script:UbuntuDistro
    )

    if (-not $Distro) {
        Write-Host "  ✗ No Ubuntu distro is selected for WSL operations." -ForegroundColor Red
        Write-Host "  Re-run the prerequisite check first: .\build.ps1 check" -ForegroundColor Yellow
        return $null
    }
    # Ensure WSL can start before asking it to resolve paths.
    $null = & wsl.exe -d $Distro --exec /bin/sh -lc "exit 0" 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  ⚠ WSL2 VM failed to start for $Distro. Attempting recovery (wsl --shutdown)..." 
        & wsl.exe --shutdown 2>&1 | Out-Null
        Start-Sleep -Seconds 2
        
        # Retry WSL startup
        $null = & wsl.exe -d $Distro --exec /bin/sh -lc "exit 0" 2>$null
        if ($LASTEXITCODE -ne 0) {
            Write-Host "  ✗ WSL failed to start for $Distro even after recovery." -ForegroundColor Red
            Write-Host "  Next steps:" -ForegroundColor Yellow
            Write-Host "    1) Run: wsl --shutdown" -ForegroundColor Cyan
            Write-Host "    2) Launch Ubuntu manually: wsl -d $Distro" -ForegroundColor Cyan
            Write-Host "    3) Re-run: .\build.ps1 $Target" -ForegroundColor Cyan
            return $null
        }
    }

    # Try to get path from wslpath first
    $wslPath = & wsl.exe -d $Distro --exec wslpath -a "$PSScriptRoot" 2>$null
    if ($LASTEXITCODE -eq 0 -and -not [string]::IsNullOrWhiteSpace($wslPath)) {
        return $wslPath.Trim()
    }

    # Fallback for drives that might not be auto-mounted (e.g. network/virtual drives)
    if ($PSScriptRoot -match "^([A-Za-z]):\\(.*)") {
        $drive = $Matches[1].ToLower()
        $rest = $Matches[2].Replace('\', '/')
        $fallbackPath = "/mnt/$drive/$rest"
        
        # Verify if the fallback path exists in WSL
        & wsl.exe -d $Distro --exec /bin/sh -lc "test -d '$fallbackPath'" 2>$null
        if ($LASTEXITCODE -eq 0) {
            return $fallbackPath
        }
    }
    Write-Host "  ✗ Failed to convert repo path for WSL: $PSScriptRoot" -ForegroundColor Red
    Write-Host "  WSL distro '$Distro' cannot access this drive." -ForegroundColor Yellow
    Write-Host "  Possible causes:" -ForegroundColor Yellow
    Write-Host "    1) WSL is not fully initialized" -ForegroundColor Cyan
    Write-Host "    2) This drive is not mounted inside WSL" -ForegroundColor Cyan
    Write-Host "    3) Ubuntu is installed but not fully provisioned yet" -ForegroundColor Cyan
    Write-Host "  Next steps:" -ForegroundColor Yellow
    Write-Host "    1) Launch Ubuntu once manually: wsl -d $Distro" -ForegroundColor Cyan
    Write-Host "    2) In Ubuntu, check mounts: ls /mnt" -ForegroundColor Cyan
    Write-Host "    3) Re-run: .\build.ps1 $Target" -ForegroundColor Cyan
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

    $repoPath = Get-WslRepoPath -Distro $script:UbuntuDistro
    if (-not $repoPath) {
        return
    }

    $singleQuoteEscape = "'" + '"' + "'" + '"' + "'"
    $versionValue = $Version.Replace("'", $singleQuoteEscape)
    $repoValue = $repoPath.Replace("'", $singleQuoteEscape)
    $targetValue = $MakeTarget.Replace("'", $singleQuoteEscape)
    $linuxCommand = "cd '$repoValue' && VERSION='$versionValue' make $targetValue"

    Write-Section "WSL build delegation"
    Write-Step "Target: $MakeTarget"
    Write-Step "Repo path: $repoPath"
    & $wsl -d $script:UbuntuDistro --exec /bin/sh -lc $linuxCommand
    if ($LASTEXITCODE -ne 0) {
        Write-Error "WSL make $MakeTarget failed"
    }
}

function Invoke-Build {
    if (-not (Test-Prerequisites -Level windows)) {
        return $false
    }
    
    # Note: Full WSL/Ubuntu check happens in Invoke-All or when deb/image targets are run
    
    # Validate platform configuration
    if (-not $GOOS -or -not $GOARCH) {
        Write-Error "Platform not properly configured. GOOS=$GOOS, GOARCH=$GOARCH"
        return $false
    }

    New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
    
    Write-Banner "Building CrateOS binaries"
    Write-Step "Platform: $Platform"
    Write-Step "GOOS=$GOOS GOARCH=$GOARCH"
    Write-Step "Version: $Version"
    Write-BlankLine

    foreach ($cmd in $Commands) {
        $src = "./cmd/$cmd"
        $binaryName = if ($GOOS -eq "windows") { "$cmd.exe" } else { $cmd }
        $out = Join-Path $BinDir $binaryName
        Write-Step "Building $cmd"
        
        $env:GOOS=$GOOS
        $env:GOARCH=$GOARCH
        $env:CGO_ENABLED=0
        
        go build -trimpath `
            -ldflags "-X github.com/crateos/crateos/internal/platform.Version=$Version -X github.com/crateos/crateos/internal/platform.BuildTarget=$Platform" `
            -o $out $src
        
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to build $cmd for $Platform"
            return $false
        }
    }
    Write-BlankLine
    Write-Ok "Binaries built for $Platform in $BinDir"
    return $true
}

function Invoke-Deb {
    if (-not (Test-Prerequisites -Level all)) {
        exit 1
    }
    $target = "deb-$Platform"
    Write-Banner "Building Debian packages (package-only)"
    Write-Step "Platform: $Platform"
    Write-WarnLine "This target stops at .deb artifacts. Use image-x86 or iso to produce a bootable ISO."
    Invoke-WslMakeTarget $target
}

function Invoke-Image {
    if (-not (Test-Prerequisites -Level all)) {
        exit 1
    }
    $resolved = Resolve-ImageMakeTarget -PlatformName $Platform -ImageFormat $Format
    Write-Banner "Building $($resolved.Label)"
    Write-Step "Platform: $Platform"
    Write-Step "Format: $($resolved.Format)"
    Invoke-WslMakeTarget $resolved.MakeTarget
}

function Invoke-Qcow2 {
    $Format = "qcow2"
    Invoke-Image
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
    Write-Banner "CrateOS prerequisite check"
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
    Interactive guided build: validate prereqs → select platform(s) → build → flash instructions
    #>
    # Validate ALL prerequisites (Windows + WSL/Ubuntu) upfront
    if (-not (Test-Prerequisites -Level all)) {
        Write-Host ""
        Write-Host "✗ Prerequisites not met. Cannot proceed with multi-platform build." -ForegroundColor Red
        exit 1
    }
    
    Write-Host ""
    Write-Host "╔════════════════════════════════════════════════════════════════╗"
    Write-Host "║              CrateOS Interactive Build System                 ║"
    Write-Host "╚════════════════════════════════════════════════════════════════╝"
    Write-Host ""
    
    # Platform selection menu
    Write-Host "Select platform(s) to build:" -ForegroundColor Cyan
    Write-Host "  1) x86-64 only (Ubuntu 24.04 ISO for bootable USB/VM)"
    Write-Host "  2) Raspberry Pi 4/5 only (ARM64, bootable image)"
    Write-Host "  3) Raspberry Pi Zero 2 W only (ARM64 Lite, bootable image)"
    Write-Host "  4) All platforms (x86 + RPi4/5 + RPi Zero 2)"
    Write-Host ""
    $platformChoice = Read-Host "Enter choice (1-4)"
    
    $platformsToBuild = @()
    switch ($platformChoice) {
        "1" { $platformsToBuild = @("x86") }
        "2" { $platformsToBuild = @("rpi") }
        "3" { $platformsToBuild = @("rpi0") }
        "4" { $platformsToBuild = @("x86", "rpi", "rpi0") }
        default {
            Write-Host "Invalid choice. Exiting." -ForegroundColor Red
            exit 1
        }
    }
    
    Write-Host ""
    Write-Host "Building: $($platformsToBuild -join ', ')" -ForegroundColor Green
    Write-Host ""
    
    $buildResults = @()
    $platformCount = $platformsToBuild.Count
    $currentPlatform = 0
    
    foreach ($platform in $platformsToBuild) {
        $currentPlatform++
        Write-Host "[$currentPlatform/$platformCount] Building $($platform.ToUpper())..." -ForegroundColor Cyan
        Write-Host ""
        
        switch ($platform) {
            "x86" {
                Build-Platform "x86" "deb-x86" "image-x86" "iso"
                if ($LASTEXITCODE -eq 0) {
                    $buildResults += @{Platform="x86"; Status="Success"; Image="dist/crateos-0.1.0+noble1.iso"; FlashTool="Rufus/balena-etcher"; Device="USB drive"}
                }
            }
            "rpi" {
                Build-Platform "rpi" "deb-rpi" "image-rpi" "img"
                if ($LASTEXITCODE -eq 0) {
                    $buildResults += @{Platform="rpi"; Status="Success"; Image="dist/crateos-rpi-0.1.0+rpi1.img"; FlashTool="balena-etcher/dd"; Device="microSD card"}
                }
            }
            "rpi0" {
                Build-Platform "rpi0" "deb-rpi0" "image-rpi0" "img"
                if ($LASTEXITCODE -eq 0) {
                    $buildResults += @{Platform="rpi0"; Status="Success"; Image="dist/crateos-rpi0-0.1.0+rpi0-1.img"; FlashTool="balena-etcher/dd"; Device="microSD card"}
                }
            }
        }
        
        if ($LASTEXITCODE -ne 0) {
            Write-Host "✗ Build failed for $platform" -ForegroundColor Red
            exit 1
        }
        Write-Host ""
    }
    
    # Success summary and flashing instructions
    Write-Host "╔════════════════════════════════════════════════════════════════╗"
    Write-Host "║                  ✓ Build Complete!                             ║"
    Write-Host "╚════════════════════════════════════════════════════════════════╝"
    Write-Host ""
    Write-Host "Build Results:" -ForegroundColor Green
    Write-Host ""
    
    foreach ($result in $buildResults) {
        Write-Host "Platform: $($result.Platform.ToUpper())" -ForegroundColor Cyan
        Write-Host "  Image:      $($result.Image)"
        Write-Host "  Destination: $($result.Device)"
        Write-Host "  Flash tool: $($result.FlashTool)"
        Write-Host ""
    }
    
    # Detailed flashing instructions
    Write-Host "Next Steps - Flash to USB/microSD:" -ForegroundColor Yellow
    Write-Host ""
    
    foreach ($result in $buildResults) {
        $platform = $result.Platform
        $image = $result.Image
        $device = $result.Device
        $tool = $result.FlashTool
        
        Write-Host "┌─ $($platform.ToUpper()) ─────────────────────────────────────────┐" -ForegroundColor Yellow
        
        if ($platform -eq "x86") {
            Write-Host "│ Image: $image" -ForegroundColor Gray
            Write-Host "│ Target: $device (bootable)"
            Write-Host "│" -ForegroundColor Gray
            Write-Host "│ Option A - Using Rufus (Windows GUI):" -ForegroundColor Cyan
            Write-Host "│   1) Download Rufus from https://rufus.ie/"
            Write-Host "│   2) Insert USB drive"
            Write-Host "│   3) Run Rufus → Select ISO → Select USB → Write"
            Write-Host "│" -ForegroundColor Gray
            Write-Host "│ Option B - Using balena-etcher (cross-platform):" -ForegroundColor Cyan
            Write-Host "│   1) Download balena-etcher from https://www.balena.io/etcher/"
            Write-Host "│   2) Run balena-etcher → Select image → Select USB → Flash"
            Write-Host "│" -ForegroundColor Gray
            Write-Host "│ Option C - Using PowerShell (advanced):" -ForegroundColor Cyan
            Write-Host "│   1) Get-Volume | ? {\$_.FileSystemLabel -eq 'LABEL'} | % DriveLetter"
            Write-Host "│   2) dd if=$image of=\dev\\sdX bs=4M status=progress (in WSL)"
        } else {
            Write-Host "│ Image: $image" -ForegroundColor Gray
            Write-Host "│ Target: $device (bootable)"
            Write-Host "│" -ForegroundColor Gray
            Write-Host "│ Option A - Using balena-etcher (recommended, cross-platform):" -ForegroundColor Cyan
            Write-Host "│   1) Download balena-etcher from https://www.balena.io/etcher/"
            Write-Host "│   2) Insert microSD card"
            Write-Host "│   3) Run balena-etcher → Select image → Select card → Flash"
            Write-Host "│" -ForegroundColor Gray
            Write-Host "│ Option B - Using dd in WSL (if you're comfortable with CLI):" -ForegroundColor Cyan
            Write-Host "│   1) Open WSL terminal: wsl"
            Write-Host "│   2) Identify card: lsblk (look for your microSD)"
            Write-Host "│   3) Flash: sudo dd if=$image of=/dev/sdX bs=4M status=progress"
            Write-Host "│   4) Sync:  sudo sync"
        }
        
        Write-Host "│" -ForegroundColor Gray
        Write-Host "│ ⚠ WARNING: Verify device letter/path before flashing!" -ForegroundColor Red
        Write-Host "│ Flashing to wrong device will overwrite your data!" -ForegroundColor Red
        Write-Host "└─────────────────────────────────────────────────────────┘" -ForegroundColor Yellow
        Write-Host ""
    }
    
    Write-Host "Images are ready in: $(Resolve-Path $Dist)" -ForegroundColor Green
    Write-Host ""
    Write-Host "After flashing:" -ForegroundColor Cyan
    Write-Host "  1) Insert flashed USB/microSD into target device"
    Write-Host "  2) Boot from USB/microSD (check BIOS/firmware settings)"
    Write-Host "  3) Follow on-screen CrateOS installation prompts"
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
        [string]$ImageTarget,

        [Parameter(Mandatory=$true)]
        [string]$ImageFormat
    )
    
    # Set global platform variables
    Set-PlatformState -PlatformName $PlatformName
    
    if (-not (Invoke-Build)) { return }
    
    # Delegate to WSL for deb and image targets
    Invoke-WslMakeTarget $DebTarget
    if ($LASTEXITCODE -ne 0) { return }
    
    $script:Format = $ImageFormat
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
    Write-Host "  deb     Build .deb packages via WSL2"
    Write-Host "  image   Build final image for -Platform/-Format via WSL2"
    Write-Host "  iso     Build autoinstall ISO via WSL2"
    Write-Host "  qcow2   Build VM image via WSL2"
    Write-Host "  rpi     Build Pi image via WSL2"
    Write-Host "  clean   Remove dist/"
    Write-Host "  help    Show this message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host '  . \build.ps1 all           # Build everything (all platforms)'
    Write-Host '  . \build.ps1 check         # Verify system is ready'
    Write-Host '  . \build.ps1 deb           # Create .deb packages (requires WSL2)'
    Write-Host '  . \build.ps1 image -Platform x86 -Format qcow2'
    Write-Host '  . \build.ps1 iso           # Build bootable ISO (requires WSL2)'
    Write-Host ""
    Write-Host "Prerequisites:"
    Write-Host "  - Windows: Go 1.20+, PowerShell 5.0+"
    Write-Host "  - WSL2: Ubuntu distro installed and running"
    Write-Host "  - WSL Ubuntu: wget, p7zip-full, xorriso, dpkg, make, golang-go"
    Write-Host ""
    Write-Host 'Use ". \build.ps1 check" to diagnose issues.'
}

function Invoke-TargetPrompt {
    Write-Banner "Choose build target"
    Write-Host "  1) Guided build flow" -ForegroundColor Cyan
    Write-Host "  2) Prerequisite check" -ForegroundColor Cyan
    Write-Host "  3) Full x86 ISO build (.deb -> .iso)" -ForegroundColor Cyan
    Write-Host "  4) Debian package only (x86)" -ForegroundColor Cyan
    Write-Host "  5) Debian package only (Raspberry Pi 4/5)" -ForegroundColor Cyan
    Write-Host "  6) Debian package only (Raspberry Pi Zero 2)" -ForegroundColor Cyan
    Write-Host "  7) Raspberry Pi image" -ForegroundColor Cyan
    Write-Host "  8) Raspberry Pi Zero 2 image" -ForegroundColor Cyan
    Write-Host "  9) QCOW2 image (x86)" -ForegroundColor Cyan
    Write-Host " 10) Help" -ForegroundColor Cyan
    Write-BlankLine

    $selection = Read-Host "Enter choice (1-10)"
    switch ($selection) {
        "1" { return "all" }
        "2" { return "check" }
        "3" { return "image-x86" }
        "4" { return "deb-x86" }
        "5" { return "deb-rpi" }
        "6" { return "deb-rpi0" }
        "7" { return "image-rpi" }
        "8" { return "image-rpi0" }
        "9" { return "qcow2" }
        "10" { return "help" }
        default {
            Write-ErrorLine "Invalid selection."
            return $null
        }
    }
}

if ($Target -eq "prompt") {
    $Target = Invoke-TargetPrompt
    if (-not $Target) {
        exit 1
    }
}

switch ($Target) {
    "all"           { Invoke-All }
    "check"         { Invoke-Check }
    "deb"           { Invoke-Deb }
    "deb-x86"       { Set-PlatformState -PlatformName "x86"; Invoke-Deb }
    "deb-rpi"       { Set-PlatformState -PlatformName "rpi"; Invoke-Deb }
    "deb-rpi0"      { Set-PlatformState -PlatformName "rpi0"; Invoke-Deb }
    "image"         { Invoke-Image }
    "iso"           { Set-PlatformState -PlatformName "x86"; $Format = "iso"; Invoke-Image }
    "image-x86"     { Set-PlatformState -PlatformName "x86"; $Format = "iso"; Invoke-Image }
    "image-rpi"     { Set-PlatformState -PlatformName "rpi"; $Format = "img"; Invoke-Image }
    "image-rpi0"    { Set-PlatformState -PlatformName "rpi0"; $Format = "img"; Invoke-Image }
    "qcow2"         { Invoke-Qcow2 }
    "rpi"           { Set-PlatformState -PlatformName "rpi"; $Format = "img"; Invoke-Image }
    "clean"         { Invoke-Clean }
    "help"          { Invoke-Help }
}
