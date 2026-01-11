$ErrorActionPreference = "Stop"

$Repo = "pranshuparmar/witr"
$InstallDir = Join-Path $env:LOCALAPPDATA "witr\bin"
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")

Write-Host "Installing witr..."

# 1. Get Latest Tag
Write-Host "Fetching latest version..."
$LatestUrl = "https://api.github.com/repos/$Repo/releases/latest"
try {
    $LatestJson = Invoke-RestMethod -Uri $LatestUrl
    $LatestTag = $LatestJson.tag_name
} catch {
    Write-Error "Failed to fetch latest release version. check internet connection."
    exit 1
}
Write-Host "Detected latest version: $LatestTag"

# 2. Setup Paths
if ($env:PROCESSOR_ARCHITECTURE -eq "AMD64") {
    $ZipName = "witr-windows-amd64.zip"
} elseif ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $ZipName = "witr-windows-arm64.zip"
} else {
    Write-Error "Unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)"
    exit 1
}
Write-Host "Detected Architecture: $($env:PROCESSOR_ARCHITECTURE)"
$DownloadUrl = "https://github.com/$Repo/releases/download/$LatestTag/$ZipName"
$ChecksumUrl = "https://github.com/$Repo/releases/download/$LatestTag/SHA256SUMS"
$TempDir = [System.IO.Path]::GetTempPath()
$ZipPath = Join-Path $TempDir $ZipName
$ChecksumPath = Join-Path $TempDir "SHA256SUMS"

# 3. Download
Write-Host "Downloading $DownloadUrl..."
Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath
Invoke-WebRequest -Uri $ChecksumUrl -OutFile $ChecksumPath

# 4. Verify Checksum
Write-Host "Verifying checksum..."
$Hash = Get-FileHash -Algorithm SHA256 $ZipPath
$ExpectedLine = Select-String -Path $ChecksumPath -Pattern $ZipName
if (-not $ExpectedLine) {
    Write-Error "Checksum verification failed: could not find $ZipName in SHA256SUMS"
    Remove-Item $ZipPath, $ChecksumPath -ErrorAction SilentlyContinue
    exit 1
}
$ExpectedHash = $ExpectedLine.Line.Split(' ')[0].Trim()

if ($Hash.Hash.ToLower() -ne $ExpectedHash.ToLower()) {
    Write-Error "Checksum mismatch!`nExpected: $ExpectedHash`nActual:   $($Hash.Hash)"
    Remove-Item $ZipPath, $ChecksumPath -ErrorAction SilentlyContinue
    exit 1
}
Write-Host "Checksum verified."

# 5. Extract/Install
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

Write-Host "Extracting to $InstallDir..."
Expand-Archive -Path $ZipPath -DestinationPath $InstallDir -Force

# Cleanup downloads
Remove-Item $ZipPath, $ChecksumPath -ErrorAction SilentlyContinue

# 6. Update PATH
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host "Adding $InstallDir to User Path..."
    $NewPath = "$UserPath;$InstallDir" 
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    $env:Path += ";$InstallDir"
    Write-Host "Path updated. You may need to restart your shell."
} else {
    Write-Host "Install directory already in Path."
}

Write-Host "`nwitr ($LatestTag) installed successfully!"
Write-Host "Try running: witr --help"
