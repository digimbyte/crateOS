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
    Write-Warning "deb packaging requires Linux (WSL2 or CI). Run 'make deb' in WSL2."
}

function Invoke-Iso {
    Write-Warning "ISO build requires Linux (WSL2 or CI). Run 'make iso' in WSL2."
}

function Invoke-Qcow2 {
    Write-Warning "qcow2 build requires Linux (WSL2 or CI). Run 'make qcow2' in WSL2."
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
    Write-Host "  deb     Build .deb packages (WSL2 only)"
    Write-Host "  iso     Build autoinstall ISO (WSL2 only)"
    Write-Host "  qcow2   Build VM image (WSL2 only)"
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
