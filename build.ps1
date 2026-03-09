param(
    [Parameter(Position=0)]
    [ValidateSet("build","deb","iso","qcow2","rpi","clean","help")]
    [string]$Target = "build"
)

$ErrorActionPreference = "Stop"

$Version  = if ($env:VERSION) { $env:VERSION } else { "0.1.0-dev" }
$Dist     = Join-Path $PSScriptRoot "dist"
$BinDir   = Join-Path $Dist "bin"
$Commands = @("crateos", "crateos-agent", "crateos-policy")

function Get-WslCommand {
    $wsl = Get-Command wsl.exe -ErrorAction SilentlyContinue
    if (-not $wsl) {
        Write-Error "WSL is not installed or not in PATH. Install WSL2/Ubuntu first."
        return $null
    }
    return $wsl.Source
}

function Get-WslRepoPath {
    $wslPath = & wsl.exe wslpath -a "$PSScriptRoot"
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($wslPath)) {
        Write-Error "Failed to convert repo path for WSL: $PSScriptRoot"
        return $null
    }
    return $wslPath.Trim()
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
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Error "Go is not installed or not in PATH. Install Go first: https://go.dev/dl/"
        return
    }

    New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

    foreach ($cmd in $Commands) {
        $src = "./cmd/$cmd"
        $out = Join-Path $BinDir "$cmd.exe"
        Write-Host "==> building $cmd -> $out"
        go build -trimpath -o $out $src
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to build $cmd"
            return
        }
    }
    Write-Host "==> all binaries built in $BinDir"
}

function Invoke-Deb {
    Invoke-WslMakeTarget "deb"
}

function Invoke-Iso {
    Invoke-WslMakeTarget "iso"
}

function Invoke-Qcow2 {
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

function Invoke-Help {
    Write-Host "CrateOS build script"
    Write-Host ""
    Write-Host "Usage: .\build.ps1 <target>"
    Write-Host ""
    Write-Host "Targets:"
    Write-Host "  build   Compile Go binaries (default)"
    Write-Host "  deb     Build .deb packages via WSL2"
    Write-Host "  iso     Build autoinstall ISO via WSL2"
    Write-Host "  qcow2   Build VM image via WSL2"
    Write-Host "  rpi     Build Pi image (stub)"
    Write-Host "  clean   Remove dist/"
    Write-Host "  help    Show this message"
}

switch ($Target) {
    "build"  { Invoke-Build }
    "deb"    { Invoke-Deb }
    "iso"    { Invoke-Iso }
    "qcow2"  { Invoke-Qcow2 }
    "rpi"    { Invoke-Rpi }
    "clean"  { Invoke-Clean }
    "help"   { Invoke-Help }
}
