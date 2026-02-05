# vibe auracle Windows Installer
# Usage: iex (irm https://raw.githubusercontent.com/nathfavour/vibeauracle/release/install.ps1)

$ErrorActionPreference = "Stop"

$Repo = "nathfavour/vibeauracle"
$GithubUrl = "https://github.com/$Repo"

# Detect Architecture
$Arch = "amd64"
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $Arch = "arm64"
}

$BinaryName = "vibeaura-windows-$Arch.exe"

Write-Host "Detected Platform: Windows/$Arch" -ForegroundColor Cyan

# --- Source & Config Detection ---
$BuildFromSource = $false
$SourceMode = "none"
$ConfigFile = Join-Path $HOME ".vibeauracle\config.yaml"
if (Test-Path $ConfigFile) {
    $Content = Get-Content $ConfigFile -Raw
    if ($Content -match "build_from_source: true") {
        $BuildFromSource = $true
    }
}

# Detect if we are inside the source tree
if ((Test-Path "go.work") -and (Test-Path "cmd\vibeaura")) {
    $BuildFromSource = $true
    $SourceMode = "local"
}

# Detect if existing installation is a source build
$ExistingVibe = $null
if (Get-Command vibeaura -ErrorAction SilentlyContinue) {
    $ExistingVibe = (Get-Command vibeaura).Source
    $VersionOut = (& $ExistingVibe version)
    if ($VersionOut -match "Version\s*:\s*(master|main|release|dev)") {
        $BuildFromSource = $true
    }
} elseif (Test-Path "$HOME\.vibeaura\bin\vibeaura.exe") {
    $ExistingVibe = "$HOME\.vibeaura\bin\vibeaura.exe"
}

if ($BuildFromSource) {
    Write-Host "Existing configuration or environment suggests building from source." -ForegroundColor Cyan
    if (Get-Command go -ErrorAction SilentlyContinue) {
        if ($SourceMode -eq "local") {
            Write-Host "Building from current directory..." -ForegroundColor Cyan
            $Commit = (git rev-parse HEAD 2>$null)
            if (-not $Commit) { $Commit = "unknown" }
            $Date = (Get-Date -UFormat "%Y-%m-%dT%H:%M:%SZ")
            $Branch = (git rev-parse --abbrev-ref HEAD 2>$null)
            if (-not $Branch) { $Branch = "master" }
            
            go build -ldflags "-s -w -X main.Version=$Branch -X main.Commit=$Commit -X main.BuildDate=$Date" -o vibeaura.exe ./cmd/vibeaura
            # We continue with the rest of the script to install the binary we just built
        } else {
            if ($ExistingVibe) {
                Write-Host "Handing over to 'vibeaura update' for source-based update..." -ForegroundColor Cyan
                & $ExistingVibe update
                return
            } else {
                Write-Host "Installing via 'go install'..." -ForegroundColor Cyan
                $env:GOTOOLCHAIN = "local"
                go install "github.com/$Repo/cmd/vibeaura@latest"
                Write-Host "Successfully installed to $((go env GOPATH))\bin\vibeaura.exe" -ForegroundColor Green
                return
            }
        }
    } else {
        Write-Host "Warning: Source build preferred but 'go' not found. Falling back to binary release." -ForegroundColor Yellow
    }
}

if (-not (Test-Path "vibeaura.exe")) {
    # Get latest release tag
    $ReleaseInfo = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $LatestTag = $ReleaseInfo.tag_name

    if (-not $LatestTag) {
        Write-Error "Could not find latest release. Please check $GithubUrl/releases"
    }

    # Check if vibeaura is already installed and up-to-date
    if ($ExistingVibe) {
        $VersionOutput = (& $ExistingVibe version)
        $VersionLine = ($VersionOutput | Select-String "Version")
        $CommitLine = ($VersionOutput | Select-String "Commit")
        
        if ($VersionLine -and $CommitLine) {
            $LocalVersion = $VersionLine.ToString().Split(":")[1].Trim()
            $LocalCommit = $CommitLine.ToString().Split(":")[1].Trim()
            
            # Resolve the SHA of the latest tag to be sure
            $LatestSHA = $null
            if (Get-Command git -ErrorAction SilentlyContinue) {
                $Match = (git ls-remote --tags "$GithubUrl.git" | Select-String "refs/tags/$LatestTag$")
                if ($Match) {
                    $LatestSHA = $Match.ToString().Split("`t")[0].Trim()
                }
            }

            # If the local version matches the latest tag, OR the local commit matches the latest SHA, we can skip
            if (($LocalVersion -eq $LatestTag) -or ($null -ne $LatestSHA -and $LocalCommit -eq $LatestSHA)) {
                Write-Host "Vibe Auracle is already up to date ($LatestTag / $($LocalCommit.Substring(0,7)))." -ForegroundColor Green
                return
            }
        }
    }

    $DownloadUrl = "$GithubUrl/releases/download/$LatestTag/$BinaryName"

    Write-Host "Downloading $BinaryName ($LatestTag)..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $DownloadUrl -OutFile "vibeaura.exe"
}

$InstallDir = Join-Path $HOME ".vibeaura\bin"
if (-not (Test-Path $InstallDir)) {
    New-Item -Path $InstallDir -ItemType Directory | Out-Null
}

$ExePath = Join-Path $InstallDir "vibeaura.exe"

# Move the binary to the install directory
Move-Item -Path "vibeaura.exe" -Destination $ExePath -Force

# Add to Path for current session
if ($env:Path -notlike "*$InstallDir*") {
    Write-Host "Adding $InstallDir to User Path..." -ForegroundColor Yellow
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    }
    $env:Path += ";$InstallDir"
}

Write-Host "Successfully installed vibe auracle to $ExePath" -ForegroundColor Green
Write-Host "You may need to restart your terminal for changes to take effect." -ForegroundColor Yellow
& "$ExePath" --help
