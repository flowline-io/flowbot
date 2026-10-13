#Requires -Version 5.1
<#
.SYNOPSIS
    Install flowbot CLI from GitHub releases (Windows).

.DESCRIPTION
    Downloads the flowbot CLI binary for the current Windows architecture,
    optionally verifies the SHA256 checksum, installs to a user-writable
    directory, and ensures that directory is on the user PATH.

.PARAMETER Version
    Release tag to install (default: latest).

.PARAMETER NoVerify
    Skip checksum verification.

.PARAMETER Help
    Show this help message.

.EXAMPLE
    irm https://raw.githubusercontent.com/flowline-io/flowbot/master/scripts/install.ps1 | iex

.EXAMPLE
    .\install.ps1 -Version v0.40

.EXAMPLE
    .\install.ps1 --version v0.40 --no-verify
#>
[CmdletBinding()]
param(
	[string]$Version = "latest",
	[switch]$NoVerify,
	[switch]$Help,
	[Parameter(ValueFromRemainingArguments = $true)]
	[string[]]$RemainingArgs
)

$ErrorActionPreference = "Stop"

$Repo = "flowline-io/flowbot"
$CliAsset = "flowbot-cli"
$BinaryName = "flowbot.exe"
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\flowbot"

function Write-Info {
	param([string]$Message)
	Write-Host "  [INFO]  $Message" -ForegroundColor Green
}

function Write-Warn {
	param([string]$Message)
	Write-Host "[WARN]  $Message" -ForegroundColor Yellow
}

function Write-Err {
	param([string]$Message)
	Write-Host "  [ERROR] $Message" -ForegroundColor Red
}

function Show-Usage {
	@"
Usage: install.ps1 [OPTIONS]

Install flowbot CLI from GitHub releases.

Options:
  -Version TAG    Install a specific version (default: latest)
  -NoVerify       Skip checksum verification
  -Help           Show this help message

  --version TAG   Same as -Version (bash-style)
  --no-verify     Same as -NoVerify (bash-style)
  --help          Same as -Help (bash-style)

Examples:
  irm https://raw.githubusercontent.com/flowline-io/flowbot/master/scripts/install.ps1 | iex
  .\install.ps1 -Version v0.40
  .\install.ps1 --version v0.40
"@ | Write-Host
}

function Parse-RemainingArgs {
	$i = 0
	while ($i -lt $RemainingArgs.Count) {
		switch ($RemainingArgs[$i]) {
			"--version" {
				if ($i + 1 -ge $RemainingArgs.Count) {
					Write-Err "--version requires a value"
					Show-Usage
					exit 1
				}
				$script:Version = $RemainingArgs[$i + 1]
				$i += 2
			}
			"--no-verify" {
				$script:NoVerify = $true
				$i += 1
			}
			"--help" {
				Show-Usage
				exit 0
			}
			default {
				Write-Err "Unknown option: $($RemainingArgs[$i])"
				Show-Usage
				exit 1
			}
		}
	}
}

function Get-Platform {
	$arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
		"X64" { "amd64" }
		"Arm64" { "arm64" }
		default {
			# Fallback for Windows PowerShell 5.1 without OSArchitecture nuance
			if ($env:PROCESSOR_ARCHITECTURE -eq "AMD64" -or $env:PROCESSOR_ARCHITEW6432 -eq "AMD64") {
				"amd64"
			}
			elseif ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
				"arm64"
			}
			else {
				Write-Err "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
				exit 1
			}
		}
	}

	# goreleaser currently ships windows/amd64 only; fail early on unsupported arches
	if ($arch -ne "amd64") {
		Write-Err "Unsupported architecture for Windows releases: $arch (only amd64 is published)"
		exit 1
	}

	return @{ Os = "windows"; Arch = $arch }
}

function Get-ReleaseUrl {
	param(
		[string]$AssetName,
		[string]$Tag
	)

	if ($Tag -eq "latest") {
		return "https://github.com/$Repo/releases/latest/download/$AssetName"
	}
	return "https://github.com/$Repo/releases/download/$Tag/$AssetName"
}

function Invoke-Download {
	param(
		[string]$Url,
		[string]$Destination
	)

	# Prefer curl.exe when present (matches install.sh retry behavior)
	$curl = Get-Command curl.exe -ErrorAction SilentlyContinue
	if ($curl) {
		& curl.exe -fsSL --retry 3 --retry-delay 2 -o $Destination $Url
		if ($LASTEXITCODE -ne 0) {
			throw "curl.exe failed with exit code $LASTEXITCODE"
		}
		return
	}

	$prevProgress = $ProgressPreference
	$ProgressPreference = "SilentlyContinue"
	try {
		Invoke-WebRequest -Uri $Url -OutFile $Destination -UseBasicParsing
	}
	finally {
		$ProgressPreference = $prevProgress
	}
}

function Test-Checksum {
	param(
		[string]$BinaryPath,
		[string]$AssetName,
		[string]$TempDir
	)

	$checksumUrl = Get-ReleaseUrl -AssetName "flowbot_checksums.txt" -Tag $Version
	$checksumFile = Join-Path $TempDir "checksums.txt"

	Write-Info "Downloading checksums..."
	try {
		Invoke-Download -Url $checksumUrl -Destination $checksumFile
	}
	catch {
		Write-Warn "Could not download checksum file, skipping verification"
		return
	}

	$expected = $null
	foreach ($line in Get-Content -Path $checksumFile) {
		$parts = $line -split '\s+', 2
		if ($parts.Count -eq 2 -and $parts[1] -eq $AssetName) {
			$expected = $parts[0]
			break
		}
	}

	if (-not $expected) {
		Write-Warn "No checksum entry found for $AssetName, skipping verification"
		return
	}

	$actual = (Get-FileHash -Path $BinaryPath -Algorithm SHA256).Hash.ToLowerInvariant()
	$expected = $expected.ToLowerInvariant()

	if ($expected -ne $actual) {
		Write-Err "Checksum verification failed"
		Write-Err "  expected: $expected"
		Write-Err "  actual:   $actual"
		exit 1
	}

	Write-Info "Checksum verified"
}

function Install-Binary {
	param(
		[string]$Source
	)

	if (-not (Test-Path -LiteralPath $InstallDir)) {
		New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
	}

	$dest = Join-Path $InstallDir $BinaryName
	Copy-Item -LiteralPath $Source -Destination $dest -Force
	return $dest
}

function Ensure-UserPath {
	param(
		[string]$Dir
	)

	$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
	if (-not $userPath) {
		$userPath = ""
	}

	$entries = $userPath -split ";" | Where-Object { $_ -ne "" }
	$already = $entries | Where-Object { $_.TrimEnd("\") -ieq $Dir.TrimEnd("\") }
	if ($already) {
		return
	}

	$newPath = if ($userPath.TrimEnd(";")) {
		"$($userPath.TrimEnd(';'));$Dir"
	}
	else {
		$Dir
	}

	[Environment]::SetEnvironmentVariable("Path", $newPath, "User")

	# Update current session so `flowbot` works immediately when possible
	if ($env:Path -notlike "*$Dir*") {
		$env:Path = "$env:Path;$Dir"
	}

	Write-Info "Added $Dir to user PATH"
	Write-Warn "Open a new terminal for PATH changes to take effect in other sessions"
}

function Main {
	Parse-RemainingArgs

	if ($Help) {
		Show-Usage
		exit 0
	}

	$platform = Get-Platform
	$assetName = "${CliAsset}_$($platform.Os)_$($platform.Arch).exe"
	$downloadUrl = Get-ReleaseUrl -AssetName $assetName -Tag $Version

	$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("flowbot-install-" + [guid]::NewGuid().ToString("N"))
	New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

	try {
		$binaryPath = Join-Path $tempDir $BinaryName

		Write-Info "Downloading flowbot CLI $Version ($($platform.Os)/$($platform.Arch))..."
		try {
			Invoke-Download -Url $downloadUrl -Destination $binaryPath
		}
		catch {
			Write-Err "Failed to download from $downloadUrl"
			Write-Err "Check that the version exists and your platform is supported"
			Write-Err $_.Exception.Message
			exit 1
		}

		if (-not $NoVerify) {
			Test-Checksum -BinaryPath $binaryPath -AssetName $assetName -TempDir $tempDir
		}
		else {
			Write-Info "Checksum verification skipped (-NoVerify / --no-verify)"
		}

		$installed = Install-Binary -Source $binaryPath
		Ensure-UserPath -Dir $InstallDir

		Write-Info "Flowbot CLI installed to $installed"

		try {
			& $installed version
			Write-Info "Installation successful"
		}
		catch {
			Write-Warn "Installation completed, but running 'flowbot version' failed"
		}

		$cmd = Get-Command flowbot -ErrorAction SilentlyContinue
		if (-not $cmd) {
			Write-Warn "$InstallDir may not be visible in this session yet"
			Write-Warn "Restart the terminal, or run: `$env:Path += ';$InstallDir'"
		}
	}
	finally {
		if (Test-Path -LiteralPath $tempDir) {
			Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
		}
	}
}

Main
