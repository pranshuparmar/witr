<div align="center">

# witr

### Why is this running?

[![Go Version](https://img.shields.io/github/go-mod/go-version/pranshuparmar/witr?style=flat-square)](https://github.com/pranshuparmar/witr/blob/main/go.mod) [![Go Report Card](https://goreportcard.com/badge/github.com/pranshuparmar/witr?style=flat-square)](https://goreportcard.com/report/github.com/pranshuparmar/witr) [![Release](https://img.shields.io/github/actions/workflow/status/pranshuparmar/witr/release.yml?style=flat-square)](https://github.com/pranshuparmar/witr/actions/workflows/release.yml) [![Platforms](https://img.shields.io/badge/platforms-linux%20%7C%20macos%20%7C%20windows%20%7C%20freebsd-blue?style=flat-square)](https://github.com/pranshuparmar/witr) <br>
[![Latest Release](https://img.shields.io/github/v/release/pranshuparmar/witr?label=Latest%20Release&style=flat-square)](https://github.com/pranshuparmar/witr/releases/latest) [![Package Managers](https://img.shields.io/badge/Package%20Managers-brew%20|%20conda%20|%20aur%20|%20winget%20|%20choco%20|%20scoop%20|%20ports%20|%20aosc%20|%20guix%20|%20uniget%20|%20brioche%20|%20aqua%20-blue?style=flat-square)](https://repology.org/project/witr/versions)

✨ *Introducing the new* [**Interactive TUI Mode**](#3-interactive-mode-tui)

<img width="1232" height="693" alt="witr_banner" src="https://github.com/user-attachments/assets/e9c19ef0-1391-4a5f-a015-f4003d3697a9" />

</div>

---

<div align="center">

[**Purpose**](#1-purpose) • [**Installation**](#2-installation) • [**TUI**](#3-interactive-mode-tui) <sup>✨</sup> • [**Flags**](#4-flags--options) • [**Examples**](#5-example-outputs) • [**Platforms**](#6-platform-support)
<br>
[**Goals**](#7-goals) • [**Core Concept**](#8-core-concept) • [**Output Behavior**](#9-output-behavior) • [**Success Criteria**](#10-success-criteria)

</div>

---

## 1. Purpose

**witr** exists to answer a single question:

> **Why is this running?**

When something is running on a system, whether it is a process, a service, or something bound to a port, there is always a cause. That cause is often indirect, non-obvious, or spread across multiple layers such as supervisors, containers, services, or shells.

Existing tools (`ps`, `top`, `lsof`, `ss`, `systemctl`, `docker ps`) expose state and metadata. They show _what_ is running, but leave the user to infer _why_ by manually correlating outputs across tools.

**witr** makes that causality explicit.

It explains **where a running thing came from**, **how it was started**, and **what chain of systems is responsible for it existing right now**, in a single, human-readable output or an **interactive TUI dashboard**.

---

## 2. Installation

witr is distributed as a single static binary for Linux, macOS, FreeBSD, and Windows.

witr is also independently packaged and maintained across multiple operating systems and ecosystems. An up-to-date overview of packaging status is available on [Repology](https://repology.org/project/witr/versions). Please note that community packages may lag GitHub releases due to independent review and validation.

> [!TIP]
> If you use a package manager (Homebrew, Conda, Winget, etc.), we recommend installing via that for easier updates. Otherwise, the install script is the quickest way to get started.

---

### 2.1 Quick Install

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
<summary><strong>Homebrew (macOS & Linux)</strong> <a href="https://formulae.brew.sh/formula/witr"><img src="https://img.shields.io/homebrew/v/witr?style=flat-square" alt="Homebrew"></a></summary>
<br>


You can install **witr** using [Homebrew](https://brew.sh/) on macOS or Linux:

```bash
brew install witr
```
</details>

<details>
<summary><strong>Conda (macOS, Linux & Windows)</strong> <a href="https://anaconda.org/conda-forge/witr"><img src="https://img.shields.io/conda/vn/conda-forge/witr?style=flat-square" alt="Conda"></a></summary>
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
<summary><strong>Arch Linux (AUR)</strong> <a href="https://aur.archlinux.org/packages/witr-bin"><img src="https://img.shields.io/aur/version/witr-bin?style=flat-square" alt="AUR"></a></summary>
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
<summary><strong>Winget (Windows)</strong> <a href="https://winstall.app/apps/PranshuParmar.witr"><img src="https://img.shields.io/winget/v/PranshuParmar.witr?style=flat-square" alt="Winget"></a></summary>
<br>


You can install **witr** via [winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/):

```powershell
winget install -e --id PranshuParmar.witr
```
</details>

<details>
<summary><strong>Chocolatey (Windows)</strong> <a href="https://community.chocolatey.org/packages/witr"><img src="https://img.shields.io/chocolatey/v/witr?style=flat-square" alt="Chocolatey"></a></summary>
<br>


You can install **witr** using [Chocolatey](https://community.chocolatey.org):

```powershell
choco install witr
```
</details>

<details>
<summary><strong>Scoop (Windows)</strong> <a href="https://scoop.sh/#/apps?q=witr"><img src="https://img.shields.io/scoop/v/witr?bucket=main&style=flat-square" alt="Scoop"></a></summary>
<br>


You can install **witr** using [Scoop](https://scoop.sh):

```powershell
scoop install main/witr
```
</details>


<details>
<summary><strong>FreeBSD Ports</strong> <a href="https://www.freshports.org/sysutils/witr/"><img src="https://repology.org/badge/version-for-repo/freebsd/witr.svg?style=flat-square" alt="FreeBSD Port"></a></summary>
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
<summary><strong>AOSC OS</strong> <a href="https://packages.aosc.io/packages/witr"><img src="https://repology.org/badge/version-for-repo/aosc/witr.svg?style=flat-square" alt="AOSC OS"></a></summary>
<br>


You can install **witr** from the [AOSC OS repository](https://packages.aosc.io/packages/witr):

```bash
oma install witr
```
</details>

<details>
<summary><strong>GNU Guix</strong> <a href="https://packages.guix.gnu.org/packages/witr/"><img src="https://repology.org/badge/version-for-repo/gnuguix/witr.svg?style=flat-square" alt="GNU Guix"></a></summary>
<br>


You can install **witr** from the [GNU Guix repository](https://packages.guix.gnu.org/packages/witr/):

```bash
guix install witr
```
</details>

<details>
<summary><strong>Uniget (Linux)</strong> <a href="https://github.com/uniget-org/tools/tree/main/tools/witr"><img src="https://img.shields.io/badge/dynamic/yaml?url=https%3A%2F%2Fraw.githubusercontent.com%2Funiget-org%2Ftools%2Fmain%2Ftools%2Fwitr%2Fmanifest.yaml&query=%24.version&label=uniget&style=flat-square&color=blue" alt="Uniget"></a></summary>
<br>

You can install **witr** using [uniget](https://uniget.dev/):

```bash
uniget install witr
```
</details>

<details>
<summary><strong>Brioche (Linux)</strong> <a href="https://github.com/brioche-dev/brioche-packages/tree/main/packages/witr"><img src="https://img.shields.io/static/v1?label=brioche&message=v0.2.7&color=blue&style=flat-square" alt="Brioche"></a></summary>
<br>

You can install **witr** using [brioche](https://brioche.dev/):

```bash
brioche install -r witr
```
</details>

<details>
<summary><strong>Aqua (macOS, Linux & Windows)</strong> <a href="https://github.com/aquaproj/aqua-registry/blob/main/pkgs/pranshuparmar/witr"><img src="https://img.shields.io/badge/dynamic/yaml?url=https%3A%2F%2Fraw.githubusercontent.com%2Faquaproj%2Faqua-registry%2Fmain%2Fpkgs%2Fpranshuparmar%2Fwitr%2Fpkg.yaml&query=%24.packages%5B0%5D.name&label=aqua&style=flat-square&color=blue" alt="Aqua"></a></summary>
<br>

You can install **witr** using [aqua](https://aquaproj.github.io/):

```bash
# Add package
aqua g -i pranshuparmar/witr

# Install package
aqua i pranshuparmar/witr
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

### 2.4 Run Without Installation

<details>
<summary><strong>Nix Flake</strong></summary>
<br>

If you use Nix, you can build **witr** from source and run without installation:

```bash
nix run github:pranshuparmar/witr -- --help
```

</details>

<details>
<summary><strong>Pixi</strong></summary>
<br>

If you use [pixi](https://pixi.prefix.dev/latest/), you can run without installation on Linux or macOS:

```bash
pixi exec witr --help
```
</details>

---

### 2.5 Other Operations

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

If you installed via a package manager (Homebrew, Conda, etc.), please use the respective uninstall command (e.g., `brew uninstall witr`).

To completely remove script/manual installation of **witr**:

**Unix (Linux, macOS, FreeBSD)**

```bash
sudo rm -f /usr/local/bin/witr
sudo rm -f /usr/local/share/man/man1/witr.1
```

**Windows**

```powershell
Remove-Item -Recurse -Force "$env:LocalAppData\witr"
```
</details>

---
 
## 3. Interactive Mode (TUI)

Running `witr` without any arguments or with the `-i` flag launches the **Interactive Mode (TUI)**. This provides a real-time, terminal-based dashboard for exploring processes and ports.

### Key Features:
- **Live Process List**: Real-time view of all running processes with sorting and filtering.
- **Port View**: Explore open ports and immediately see which processes are holding them.
- **Process Details**: Deep-dive into a specific process to see its full ancestry tree, child processes, environment variables, working directory, and more.
- **Process Actions**: Send signals (Kill, Terminate, Pause, Resume) or Renice processes directly from the UI.
- **Mouse Support**: Navigate, sort columns, and click rows using your mouse.

---

## 4. Flags & Options

```
      --env           show environment variables for the process
  -x, --exact         use exact name matching (no substring search)
  -f, --file string   file path to find process for
  -h, --help          help for witr
  -i, --interactive   interactive mode (TUI)
      --json          show result as JSON
      --no-color      disable colorized output
  -p, --pid string    pid to look up
  -o, --port string   port to look up
  -s, --short         show only ancestry
  -t, --tree          show only ancestry as a tree
      --verbose       show extended process information
  -v, --version       version for witr
      --warnings      show only warnings
```

A single positional argument (without flags) is treated as a process or service name. By default, name matching uses substring matching (fuzzy search). Use `--exact` to match only processes with the exact name.

The TUI is launched if no arguments or relevant flags (`--pid`, `--port`, `--file`) are provided, or if the `--interactive` flag is explicitly used.

---

## 5. Example Outputs

### 5.1 Name Based Query

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

### 5.2 Short Output

```bash
witr --port 5000 --short
```

```
systemd (pid 1) → PM2 v5.3.1: God (pid 1481580) → python (pid 1482060)
```

---

### 5.3 Tree Output

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

Note: _Tree view includes child processes (up to 10) and highlights the target process._

---

### 5.4 Multiple Matches

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

To avoid substring matching and only find processes with an exact name, use the `--exact` flag:

```bash
witr nginx -x
```

---

### 5.5 File Based Query

```bash
witr --file /var/lib/dpkg/lock
```

Explains the process holding a file open.

---

## 6. Platform Support

- **Linux** (x86_64, arm64) - Full feature support (`/proc`).
- **macOS** (x86_64, arm64) - Uses `ps`, `lsof`, `sysctl`, `pgrep`.
- **Windows** (x86_64, arm64) - Uses `Get-CimInstance`, `tasklist`, `netstat`.
- **FreeBSD** (x86_64, arm64) - Uses `procstat`, `ps`, `lsof`.

---

### 5.1 Feature Compatibility Matrix

| Feature | Linux | macOS | Windows | FreeBSD | Notes |
|---------|:-----:|:-----:|:-------:|:-------:|-------|
| **Process Selection** |
| By Name | ✅ | ✅ | ✅ | ✅ | |
| By PID | ✅ | ✅ | ✅ | ✅ | |
| By Port | ✅ | ✅ | ✅ | ✅ | |
| By File | ✅ | ✅ | ❌ | ✅ | |
| Exact Match | ✅ | ✅ | ✅ | ✅ | |
| Full command line | ✅ | ✅ | ✅ | ✅ | |
| Process start time | ✅ | ✅ | ✅ | ✅ | |
| Working directory | ✅ | ✅ | ✅ | ✅ | |
| Environment variables | ✅ | ⚠️ | ❌ | ✅ | macOS: Partial support due to SIP restrictions. |
| **Network** |
| Listening ports | ✅ | ✅ | ✅ | ✅ | |
| Bind addresses | ✅ | ✅ | ✅ | ✅ | |
| Port → PID resolution | ✅ | ✅ | ✅ | ✅ | |
| **Service Detection** |
| Service Manager | ✅ | ✅ | ✅ | ✅ | Linux: systemd, macOS: launchd, Windows: Services, FreeBSD: rc.d |
| Service Description | ✅ | ✅ | ✅ | ✅ | Linux: `Description`, macOS: `Comment`, Windows: `Display Name`, FreeBSD: `rc` header |
| Configuration Source | ✅ | ✅ | ✅ | ✅ | Linux: Unit File, macOS: Plist, Windows: Registry Key, FreeBSD: Rc Script |
| Supervisor | ✅ | ✅ | ✅ | ✅ | |
| Containers | ✅ | ✅ | ✅ | ✅ | Docker (plus Compose mappings), Podman, K8s (Kubepods), Containerd. Colima on macOS/Linux. Jails on FreeBSD. |
| **Health & Diagnostics** |
| CPU usage detection | ✅ | ✅ | ✅ | ✅ | |
| Memory usage detection | ✅ | ✅ | ✅ | ✅ | |
| Health status detection | ✅ | ✅ | ✅ | ✅ | |
| Open Files / Handles | ✅ | ✅ | ⚠️ | ✅ | Windows: count only. |
| Deleted binary detection | ✅ | ✅ | ✅ | ✅ | Warns if executable is missing. |
| **Context** |
| Git repo/branch detection | ✅ | ✅ | ✅ | ✅ | |
| **Interactive Mode (TUI)** |
| Process Dashboard | ✅ | ✅ | ✅ | ✅ | |
| Port Dashboard | ✅ | ✅ | ✅ | ✅ | |
| Process Details | ✅ | ✅ | ✅ | ✅ | |
| Process Actions | ✅ | ✅ | ❌ | ✅ | |

**Legend:** ✅ Full support | ⚠️ Partial/limited support | ❌ Not available

---

### 5.2 Permissions Note

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
.\witr.exe [your arguments]
```

---

## 7. Goals

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

## 8. Core Concept

witr treats **everything as a process question**.

Ports, services, containers, and commands all eventually map to **PIDs**. Once a PID is identified, witr builds a causal chain explaining _why that PID exists_.

At its core, witr answers:

1. What is running?
2. How did it start?
3. What is keeping it running?
4. What context does it belong to?

---

## 9. Output Behavior

### 9.1 Output Principles

- Single screen by default (best effort)
- Deterministic ordering
- Narrative-style explanation
- Best-effort detection with explicit uncertainty

---

### 9.2 Standard Output Sections

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

## 10. Success Criteria

witr is successful if:

- A user can answer "why is this running?" within seconds
- It reduces reliance on multiple tools
- Output is understandable under stress
- Users trust it during incidents
