<div align="center">

# witr

### Why is this running?

[![Go Version](https://img.shields.io/github/go-mod/go-version/pranshuparmar/witr?style=flat-square)](https://github.com/pranshuparmar/witr/blob/main/go.mod) [![Go Report Card](https://goreportcard.com/badge/github.com/pranshuparmar/witr?style=flat-square)](https://goreportcard.com/report/github.com/pranshuparmar/witr) [![Build Status](https://img.shields.io/github/actions/workflow/status/pranshuparmar/witr/pr-check.yml?branch=main&style=flat-square&label=build)](https://github.com/pranshuparmar/witr/actions/workflows/pr-check.yml) [![Platforms](https://img.shields.io/badge/platforms-linux%20%7C%20macos%20%7C%20windows%20%7C%20freebsd-blue?style=flat-square)](https://github.com/pranshuparmar/witr) <br>
[![Latest Release](https://img.shields.io/github/v/release/pranshuparmar/witr?label=Latest%20Release&style=flat-square)](https://github.com/pranshuparmar/witr/releases/latest) [![Homebrew](https://img.shields.io/homebrew/v/witr?style=flat-square)](https://formulae.brew.sh/formula/witr) [![Conda](https://img.shields.io/conda/vn/conda-forge/witr?style=flat-square)](https://anaconda.org/conda-forge/witr) [![AUR](https://img.shields.io/aur/version/witr-bin?style=flat-square)](https://aur.archlinux.org/packages/witr-bin) <br>
[![FreeBSD Port](https://repology.org/badge/version-for-repo/freebsd/witr.svg?style=flat-square)](https://www.freshports.org/sysutils/witr/) [![AOSC OS](https://repology.org/badge/version-for-repo/aosc/witr.svg?style=flat-square)](https://packages.aosc.io/packages/witr) [![GNU Guix package](https://repology.org/badge/version-for-repo/gnuguix/witr.svg?style=flat-square)](https://packages.guix.gnu.org/packages/witr/)

<img width="1232" height="693" alt="witr_banner" src="https://github.com/user-attachments/assets/e9c19ef0-1391-4a5f-a015-f4003d3697a9" />

</div>

---

<div align="center">

[**Purpose**](#1-purpose) • [**Installation**](#2-installation) • [**Goals**](#3-goals) • [**Core Concept**](#4-core-concept) • [**Supported Targets**](#5-supported-targets)
<br>
[**Output Behavior**](#6-output-behavior) • [**Flags**](#7-flags--options) • [**Examples**](#8-example-outputs) • [**Platforms**](#9-platform-support) • [**Success Criteria**](#10-success-criteria)

</div>

---

## 1. Purpose

**witr** exists to answer a single question:

> **Why is this running?**

When something is running on a system—whether it is a process, a service, or something bound to a port—there is always a cause. That cause is often indirect, non-obvious, or spread across multiple layers such as supervisors, containers, services, or shells.

Existing tools (`ps`, `top`, `lsof`, `ss`, `systemctl`, `docker ps`) expose state and metadata. They show _what_ is running, but leave the user to infer _why_ by manually correlating outputs across tools.

**witr** makes that causality explicit.

It explains **where a running thing came from**, **how it was started**, and **what chain of systems is responsible for it existing right now**, in a single, human-readable output.

---

## 2. Installation

witr is distributed as a single static binary for Linux, macOS, FreeBSD, and Windows.

witr is also independently packaged and maintained across multiple operating systems and ecosystems. An up-to-date overview of packaging status is available on [Repology](https://repology.org/project/witr/versions). Please note that community packages may lag GitHub releases due to independent review and validation.

> [!TIP]
> If you use a package manager (Homebrew, Conda, etc.), we recommend installing via that for easier updates. Otherwise, the install script is the fastest way to get started.

---

### 2.1 Script Installation

The easiest way to install **witr** is via the install script.

#### Unix (Linux, macOS & FreeBSD)

```bash
curl -fsSL https://raw.githubusercontent.com/pranshuparmar/witr/main/install.sh | bash
```

<details>
<summary>Script Details</summary>

The script will:
- Detect your operating system (`linux`, `darwin` or `freebsd`)
- Detect your CPU architecture (`amd64` or `arm64`)
- Download the latest released binary and man page
- Install it to `/usr/local/bin/witr`
- Install the man page to `/usr/local/share/man/man1/witr.1`
- Pass INSTALL_PREFIX to override default install path

</details>

#### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/pranshuparmar/witr/main/install.ps1 | iex
```

<details>
<summary>Script Details</summary>

The script will:
- Download the latest release (zip) and verify checksum.
- Extract `witr.exe` to `%LocalAppData%\witr\bin`.
- Add the bin directory to your User `PATH`.

</details>

---

### 2.2 Package Managers

<details>
<summary><strong>Homebrew (macOS & Linux)</strong></summary>
<br>

You can install **witr** using [Homebrew](https://brew.sh/) on macOS or Linux:

```bash
brew install witr
```

See the [Homebrew Formula page](https://formulae.brew.sh/formula/witr#default) for more details.
</details>

<details>
<summary><strong>Conda (macOS, Linux & Windows)</strong></summary>
<br>

You can install **witr** using [conda](https://docs.conda.io/en/latest/), [mamba](https://mamba.readthedocs.io/en/latest/), or [pixi](https://pixi.prefix.dev/latest/) on macOS, Linux, and Windows:

```bash
conda install -c conda-forge witr
# alternatively using mamba
mamba install -c conda-forge witr
# alternatively using pixi
pixi global install witr
```
</details>

<details>
<summary><strong>Arch Linux (AUR)</strong></summary>
<br>

On Arch Linux and derivatives, install from the [AUR package](https://aur.archlinux.org/packages/witr-bin):

```bash
yay -S witr-bin
# alternatively using paru
paru -S witr-bin
# or use your preferred AUR helper
```
</details>

<details>
<summary><strong>FreeBSD Ports</strong></summary>
<br>

You can install **witr** on FreeBSD from the [FreshPorts port](https://www.freshports.org/sysutils/witr/):

```bash
pkg install witr
# or
pkg install sysutils/witr
```

Or build from Ports:

```bash
cd /usr/ports/sysutils/witr/
make install clean
```
</details>

<details>
<summary><strong>AOSC OS</strong></summary>
<br>

You can install **witr** from the [AOSC OS repository](https://packages.aosc.io/packages/witr):

```bash
oma install witr
```
</details>

<details>
<summary><strong>GNU Guix</strong></summary>
<br>

You can install **witr** from the [GNU Guix repository](https://packages.guix.gnu.org/packages/witr/):

```bash
guix install witr
```
</details>

<details>
<summary><strong>Prebuilt Packages (deb, rpm, apk)</strong></summary>
<br>

**witr** provides native packages for major Linux distributions. You can download the latest `.deb`, `.rpm`, or `.apk` package from the [GitHub releases page](https://github.com/pranshuparmar/witr/releases/latest).

- Generic download command using `curl`:
  ```bash
  # Replace <package name with the actual package that you need>
  curl -LO https://github.com/pranshuparmar/witr/releases/latest/download/<package-name>
  ```

- **Debian/Ubuntu (.deb):**
  ```bash
  sudo dpkg -i ./witr-*.deb
  # Or, using apt for dependency resolution:
  sudo apt install ./witr-*.deb
  ```
- **Fedora/RHEL/CentOS (.rpm):**
  ```bash
  sudo rpm -i ./witr-*.rpm
  ```
- **Alpine Linux (.apk):**
  ```bash
  sudo apk add --allow-untrusted ./witr-*.apk
  ```
</details>

---

### 2.3 Source & Manual Installation

<details>
<summary><strong>Go (cross-platform)</strong></summary>
<br>

You can install the latest version directly from source:

```bash
go install github.com/pranshuparmar/witr/cmd/witr@latest
```

This will place the `witr` binary in your `$GOPATH/bin` or `$HOME/go/bin` directory. Make sure this directory is in your `PATH`.
</details>

<details>
<summary><strong>Manual Installation</strong></summary>
<br>

If you prefer manual installation, follow these simple steps for your platform:

**Unix (Linux, macOS, FreeBSD)**

```bash
# 1. Determine OS and Architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
[ "$ARCH" = "x86_64" ] && ARCH="amd64"
[ "$ARCH" = "aarch64" ] && ARCH="arm64"

# 2. Download the binary
curl -fsSL "https://github.com/pranshuparmar/witr/releases/latest/download/witr-${OS}-${ARCH}" -o witr

# 3. Verify checksum (Optional)
curl -fsSL "https://github.com/pranshuparmar/witr/releases/latest/download/SHA256SUMS" -o SHA256SUMS
grep "witr-${OS}-${ARCH}" SHA256SUMS | (sha256sum -c - 2>/dev/null || shasum -a 256 -c - 2>/dev/null)
rm SHA256SUMS

# 4. Rename and install
chmod +x witr
sudo mkdir -p /usr/local/bin
sudo mv witr /usr/local/bin/witr

# 5. Install man page (Optional)
sudo mkdir -p /usr/local/share/man/man1
sudo curl -fsSL https://github.com/pranshuparmar/witr/releases/latest/download/witr.1 -o /usr/local/share/man/man1/witr.1
```

**Windows (PowerShell)**

```powershell
# 1. Determine Architecture
if ($env:PROCESSOR_ARCHITECTURE -eq "AMD64") {
    $ZipName = "witr-windows-amd64.zip"
} elseif ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $ZipName = "witr-windows-arm64.zip"
} else {
    Write-Error "Unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)"
    exit 1
}

# 2. Download the zip
Invoke-WebRequest -Uri "https://github.com/pranshuparmar/witr/releases/latest/download/$ZipName" -OutFile "witr.zip"
# 3. Extract the binary
Expand-Archive -Path "witr.zip" -DestinationPath "." -Force

# 4. Verify checksum (Optional)
Invoke-WebRequest -Uri "https://github.com/pranshuparmar/witr/releases/latest/download/SHA256SUMS" -OutFile "SHA256SUMS"
$hash = Get-FileHash -Algorithm SHA256 .\witr.zip
$expected = Select-String -Path .\SHA256SUMS -Pattern $ZipName
if ($expected -and $hash.Hash.ToLower() -eq $expected.Line.Split(' ')[0]) { Write-Host "Checksum OK" } else { Write-Host "Checksum Mismatch" }

# 5. Install to local bin directory
$InstallDir = "$env:LocalAppData\witr\bin"
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Move-Item .\witr.exe $InstallDir\witr.exe -Force

# 6. Add to User Path (Persistent)
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path += ";$InstallDir"
    Write-Host "Added to Path. You may need to restart PowerShell."
}

# 7. Cleanup
Remove-Item witr.zip
Remove-Item SHA256SUMS
```
</details>

---

### 2.4 Other Operations

<details>
<summary><strong>Verify Installation</strong></summary>
<br>

```bash
witr --version
man witr
```
</details>

<details>
<summary><strong>Uninstallation</strong></summary>
<br>

To completely remove **witr**:

**Unix (Linux, macOS, FreeBSD)**

```bash
sudo rm -f /usr/local/bin/witr
sudo rm -f /usr/local/share/man/man1/witr.1
```

If you installed via a package manager (Homebrew, Conda, etc.), please use the respective uninstall command (e.g., `brew uninstall witr`).

**Windows**

```powershell
Remove-Item -Recurse -Force "$env:LocalAppData\witr"
```
</details>

<details>
<summary><strong>Run Without Installation</strong></summary>
<br>

**Nix Flake**

If you use Nix, you can build **witr** from source and run without installation:

```bash
nix run github:pranshuparmar/witr -- --help
```

**Pixi**

If you use [pixi](https://pixi.prefix.dev/latest/), you can run without installation on Linux or macOS:

```bash
pixi exec witr --help
```
</details>

---

## 3. Goals

### Primary goals

- Explain **why a process exists**, not just that it exists
- Reduce time‑to‑understanding during debugging and outages
- Work with zero configuration
- Be safe, read‑only, and non‑destructive
- Prefer clarity over completeness

### Non‑goals

- Not a monitoring tool
- Not a performance profiler
- Not a replacement for systemd/docker tooling
- Not a remediation or auto‑fix tool

---

## 4. Core Concept

witr treats **everything as a process question**.

Ports, services, containers, and commands all eventually map to **PIDs**. Once a PID is identified, witr builds a causal chain explaining _why that PID exists_.

At its core, witr answers:

1. What is running?
2. How did it start?
3. What is keeping it running?
4. What context does it belong to?

---

## 5. Supported Targets

witr supports multiple entry points that converge to PID analysis.

---

### 5.1 Name (process or service)

```bash
witr node
witr nginx
```

A single positional argument (without flags) is treated as a process or service name. If multiple matches are found, witr will prompt for disambiguation by PID.

---

### 5.2 PID

```bash
witr --pid 14233
```

Explains why a specific process exists.

---

### 5.3 Port

```bash
witr --port 5000
```

Explains the process(es) listening on a port.

---

## 6. Output Behavior

### 6.1 Output Principles

- Single screen by default (best effort)
- Deterministic ordering
- Narrative-style explanation
- Best-effort detection with explicit uncertainty

---

### 6.2 Standard Output Sections

#### Target

What the user asked about.

#### Process

Executable, PID, user, command, start time and restart count.

#### Why It Exists

A causal ancestry chain showing how the process came to exist.
This is the core value of witr.

#### Source

The primary system responsible for starting or supervising the process (best effort).

Examples:

- systemd unit (Linux)
- launchd service (macOS)
- docker container
- pm2
- cron
- interactive shell

Only **one primary source** is selected.

#### Context (best effort)

- Working directory
- Git repository name and branch
- Container name / image (docker, podman, kubernetes, colima, containerd)
- Public vs private bind

#### Warnings

Non‑blocking observations such as:

- Process is running as root
- Process is listening on a public interface (0.0.0.0 / ::)
- Restarted multiple times (warning only if above threshold)
- Process is using high memory (>1GB RSS)
- Process has been running for over 90 days

---

## 7. Flags & Options

```
--pid <n>         Explain a specific PID
--port <n>        Explain port usage
--short           One-line summary
--tree            Show ancestry tree with child processes
--json            Output result as JSON
--warnings        Show only warnings
--no-color        Disable colorized output
--env             Show only environment variables for the process
--help            Show this help message
--verbose         Show extended process information
```

A single positional argument (without flags) is treated as a process or service name.

---

## 8. Example Outputs

### 8.1 Name Based Query

```bash
witr node
```

```
Target      : node

Process     : node (pid 14233)
User        : pm2
Command     : node index.js
Started     : 2 days ago (Mon 2025-02-02 11:42:10 +05:30)
Restarts    : 1

Why It Exists :
  systemd (pid 1) → pm2 (pid 5034) → node (pid 14233)

Source      : pm2

Working Dir : /opt/apps/expense-manager
Git Repo    : expense-manager (main)
Listening   : 127.0.0.1:5001
```

---

### 8.2 Short Output

```bash
witr --port 5000 --short
```

```
systemd (pid 1) → PM2 v5.3.1: God (pid 1481580) → python (pid 1482060)
```

---

### 8.3 Tree Output

```bash
witr --pid 143895 --tree
```

```
systemd (pid 1)
  └─ init-systemd(Ub (pid 2)
    └─ SessionLeader (pid 143858)
      └─ Relay(143860) (pid 143859)
        └─ bash (pid 143860)
          └─ sh (pid 143886)
            └─ node (pid 143895)
              ├─ node (pid 143930)
              ├─ node (pid 144189)
              └─ node (pid 144234)
```

_Note: Tree view now includes child processes (up to 10) and highlights the target process._

---

### 8.4 Multiple Matches

```bash
witr ng
```

```
Multiple matching processes found:

[1] nginx (pid 2311)
    nginx -g daemon off;
[2] nginx (pid 24891)
    nginx -g daemon off;
[3] ngrok (pid 14233)
    ngrok http 5000

Re-run with:
  witr --pid <pid>
```

---

## 9. Platform Support

- **Linux** (x86_64, arm64) - Full feature support (`/proc`).
- **macOS** (x86_64, arm64) - Uses `ps`, `lsof`, `sysctl`, `pgrep`.
- **Windows** (x86_64, arm64) - Uses `Get-CimInstance`, `tasklist`, `netstat`.
- **FreeBSD** (x86_64, arm64) - Uses `procstat`, `ps`, `lsof`.

---

### 9.1 Feature Compatibility Matrix

| Feature | Linux | macOS | Windows | FreeBSD | Notes |
|---------|:-----:|:-----:|:-------:|:-------:|-------|
| **Process Inspection** |
| Basic process info (PID, PPID, user, command) | ✅ | ✅ | ✅ | ✅ | |
| Full command line | ✅ | ✅ | ✅ | ✅ | |
| Process start time | ✅ | ✅ | ✅ | ✅ | |
| Working directory | ✅ | ✅ | ❌ | ✅ | Windows: hard to get without injection |
| Environment variables | ✅ | ⚠️ | ❌ | ✅ | Windows: not supported. macOS: partial. |
| **Network** |
| Listening ports | ✅ | ✅ | ✅ | ✅ | |
| Bind addresses | ✅ | ✅ | ✅ | ✅ | |
| Port → PID resolution | ✅ | ✅ | ✅ | ✅ | |
| **Service Detection** |
| systemd | ✅ | ❌ | ❌ | ❌ | Linux only |
| launchd | ❌ | ✅ | ❌ | ❌ | macOS only |
| rc.d | ❌ | ❌ | ❌ | ✅ | FreeBSD only |
| Supervisor | ✅ | ✅ | ✅ | ✅ | |
| Containers | ✅ | ⚠️ | ❌ | ✅ | Windows/macOS: Docker detects VM context. FreeBSD: Jails. |
| **Health & Diagnostics** |
| CPU usage detection | ✅ | ✅ | ✅ | ✅ | |
| Memory usage detection | ✅ | ✅ | ✅ | ✅ | |
| Health status detection | ✅ | ✅ | ✅ | ✅ | Windows checks process Status (WMI). |
| Open Files / Handles | ✅ | ✅ | ✅ | ✅ | Verbose mode only. |
| **Context** |
| Git repo/branch detection | ✅ | ✅ | ❌ | ✅ | Requires working directory |

**Legend:** ✅ Full support | ⚠️ Partial/limited support | ❌ Not available

---

### 9.2 Permissions Note

#### Linux/FreeBSD

witr inspects system directories which may require elevated permissions.

If you are not seeing the expected information, try running witr with sudo:

```bash
sudo witr [your arguments]
```

#### macOS

On macOS, witr uses `ps`, `lsof`, and `launchctl` to gather process information. Some operations may require elevated permissions:

```bash
sudo witr [your arguments]
```

Note: Due to macOS System Integrity Protection (SIP), some system process details may not be accessible even with sudo.

#### Windows

On Windows, witr uses `Get-CimInstance`, `tasklist`, and `netstat`. To see details for processes owned by other users or system services, you must run the terminal as **Administrator**.

```powershell
# Run in Administrator PowerShell
./witr.exe [your arguments]
```

---

## 10. Success Criteria

witr is successful if:

- A user can answer "why is this running?" within seconds
- It reduces reliance on multiple tools
- Output is understandable under stress
- Users trust it during incidents

---
