param(
    [ValidateSet("auto", "iso", "qcow2")]
    [string]$Artifact = "auto",

    [int]$MemoryMB = 4096,

    [int]$CpuCount = 2,

    [int]$DiskGB = 24,

    [ValidateSet("tcg", "whpx")]
    [string]$Accel = "tcg",

    [switch]$NoUefi
)

$ErrorActionPreference = "Stop"

$script:RepoRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$script:DistDir = Join-Path $script:RepoRoot "dist"
$script:VmDir = Join-Path $script:DistDir "vm"

function Write-Info {
    param([string]$Message)
    Write-Host "[info] $Message" -ForegroundColor Cyan
}

function Write-Ok {
    param([string]$Message)
    Write-Host "[ok] $Message" -ForegroundColor Green
}

function Write-WarnLine {
    param([string]$Message)
    Write-Host "[warn] $Message" -ForegroundColor Yellow
}

function Resolve-QemuCommand {
    $candidates = @(
        "qemu-system-x86_64",
        "C:\Program Files\qemu\qemu-system-x86_64.exe"
    )

    foreach ($candidate in $candidates) {
        $cmd = Get-Command $candidate -ErrorAction SilentlyContinue
        if ($cmd) {
            return $cmd.Source
        }

        if (Test-Path -LiteralPath $candidate) {
            return $candidate
        }
    }

    throw "QEMU was not found. Install QEMU for Windows, then re-run scripts/test.ps1."
}

function Resolve-QemuImgCommand {
    $candidates = @(
        "qemu-img",
        "C:\Program Files\qemu\qemu-img.exe"
    )

    foreach ($candidate in $candidates) {
        $cmd = Get-Command $candidate -ErrorAction SilentlyContinue
        if ($cmd) {
            return $cmd.Source
        }

        if (Test-Path -LiteralPath $candidate) {
            return $candidate
        }
    }

    throw "qemu-img was not found. Install QEMU for Windows, then re-run scripts/test.ps1."
}

function Get-LatestArtifact {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Patterns
    )

    $matches = foreach ($pattern in $Patterns) {
        Get-ChildItem -LiteralPath $script:DistDir -Recurse -File -Filter $pattern -ErrorAction SilentlyContinue
    }

    return $matches |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 1
}

function Resolve-Artifact {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ArtifactKind
    )

    switch ($ArtifactKind) {
        "iso" {
            $artifact = Get-LatestArtifact -Patterns @("crateos-*.iso")
            if (-not $artifact) {
                throw "No CrateOS ISO found in dist/. Expected something like dist/crateos-0.1.0+noble1.iso."
            }
            return @{
                Kind = "iso"
                Path = $artifact.FullName
                Name = $artifact.Name
            }
        }
        "qcow2" {
            $artifact = Get-LatestArtifact -Patterns @("crateos-*.qcow2")
            if (-not $artifact) {
                throw "No CrateOS qcow2 found in dist/. Expected something like dist/crateos-<version>.qcow2."
            }
            return @{
                Kind = "qcow2"
                Path = $artifact.FullName
                Name = $artifact.Name
            }
        }
        "auto" {
            $iso = Get-LatestArtifact -Patterns @("crateos-*.iso")
            if ($iso) {
                return @{
                    Kind = "iso"
                    Path = $iso.FullName
                    Name = $iso.Name
                }
            }

            $qcow2 = Get-LatestArtifact -Patterns @("crateos-*.qcow2")
            if ($qcow2) {
                return @{
                    Kind = "qcow2"
                    Path = $qcow2.FullName
                    Name = $qcow2.Name
                }
            }

            throw "No CrateOS ISO or qcow2 artifact found in dist/."
        }
    }
}

function Resolve-FirmwareArgs {
    if ($NoUefi) {
        return @()
    }

    $firmwareCandidates = @(
        "C:\Program Files\qemu\share\edk2-x86_64-code.fd",
        "C:\Program Files\qemu\share\edk2-x86_64-code.fd.bin",
        "C:\Program Files\qemu\share\edk2\ovmf\OVMF_CODE.fd",
        "C:\Program Files\qemu\share\OVMF\OVMF_CODE.fd"
    )

    foreach ($candidate in $firmwareCandidates) {
        if (Test-Path -LiteralPath $candidate) {
            return @("-drive", "if=pflash,format=raw,readonly=on,file=$candidate")
        }
    }

    Write-WarnLine "UEFI firmware was not found in the QEMU install; falling back to legacy BIOS."
    return @()
}

function Ensure-IsoDisk {
    $diskPath = Join-Path $script:VmDir "crateos-test.qcow2"
    if (-not (Test-Path -LiteralPath $diskPath)) {
        $qemuImg = Resolve-QemuImgCommand
        Write-Info "Creating VM disk: $diskPath ($DiskGB GB)"
        & $qemuImg create -f qcow2 $diskPath "$($DiskGB)G" | Out-Null
    }
    return $diskPath
}

if (-not (Test-Path -LiteralPath $script:DistDir)) {
    throw "dist/ does not exist yet."
}

New-Item -ItemType Directory -Path $script:VmDir -Force | Out-Null

$qemu = Resolve-QemuCommand
$artifact = Resolve-Artifact -ArtifactKind $Artifact
$firmwareArgs = Resolve-FirmwareArgs

Write-Ok "Using artifact: $($artifact.Name)"
Write-Info "VM files directory: $script:VmDir"

$qemuArgs = @(
    "-m", $MemoryMB,
    "-smp", $CpuCount,
    "-name", "CrateOS Test",
    "-serial", "mon:stdio",
    "-display", "default"
)

if ($Accel -eq "whpx") {
    $qemuArgs += @("-accel", "whpx")
} else {
    $qemuArgs += @("-accel", "tcg")
}

$qemuArgs += $firmwareArgs

switch ($artifact.Kind) {
    "iso" {
        $diskPath = Ensure-IsoDisk
        $qemuArgs += @(
            "-boot", "d",
            "-drive", "file=$diskPath,if=virtio,format=qcow2",
            "-cdrom", $artifact.Path
        )
    }
    "qcow2" {
        $qemuArgs += @(
            "-drive", "file=$($artifact.Path),if=virtio,format=qcow2"
        )
    }
}

Write-Info "Launching QEMU"
& $qemu @qemuArgs
