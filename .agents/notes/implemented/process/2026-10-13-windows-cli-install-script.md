# Agent Note: Windows CLI install script

Status: implemented

## Problem

`scripts/install.sh` installs the flowbot CLI from GitHub releases on Linux and macOS. Windows users had no first-party equivalent and would otherwise download assets by hand or rely on incompatible bash-oriented instructions.

## Decision

Ship `scripts/install.ps1` as a PowerShell peer of `install.sh`:

- Same release source (`flowline-io/flowbot`), asset naming (`flowbot-cli_windows_amd64.exe`), and checksum file (`flowbot_checksums.txt`).
- Same options: version pin and optional checksum skip (PowerShell `-Version` / `-NoVerify`, plus bash-style `--version` / `--no-verify`).
- Install to `%LOCALAPPDATA%\Programs\flowbot\flowbot.exe` (user-writable; no elevation) and append that directory to the user `Path` when missing.
- Prefer `curl.exe` for downloads with retries; fall back to `Invoke-WebRequest`.
- Verify with `Get-FileHash -Algorithm SHA256`. Refuse non-amd64 early because goreleaser currently publishes Windows amd64 only.

## Alternatives considered

- **Batch (`.bat` / `.cmd`) only** — weaker HTTP, hashing, and PATH APIs; PowerShell is available on supported Windows versions and matches the bash script’s structure more closely.
- **Default install under Program Files** — would need elevation for a one-file CLI; user-local Programs under LocalAppData avoids UAC for the common case.
- **Extend `install.sh` via Git Bash / MSYS** — already partially detects Windows in `install.sh`, but Git Bash is not a default Windows PATH and `/usr/local/bin` is not the Windows convention.

## Consequences

- Windows install one-liner: `irm https://raw.githubusercontent.com/flowline-io/flowbot/master/scripts/install.ps1 | iex`.
- Docs may later link the PowerShell path next to the bash path; this note does not require a docs edit in the same change.
- When goreleaser adds `windows/arm64`, remove or relax the amd64-only guard in `install.ps1`.

## Verification

- Run `.\scripts\install.ps1 -Help` and confirm usage text.
- On a Windows amd64 host: `.\scripts\install.ps1` installs `flowbot.exe`, prints `flowbot version`, and leaves the install dir on the user PATH.
- `.\scripts\install.ps1 -NoVerify` skips checksum download/compare.
- `.\scripts\install.ps1 -Version <tag>` fetches that release asset URL shape.
