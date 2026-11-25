<div align="center">

# witr

### Why is this running?
Trace any process, port, container, or file back to the exact chain that started it —<br>
one command, machine-readable JSON, or an [interactive TUI](#4-interactive-mode-tui).

[![Latest Release](https://img.shields.io/github/v/release/pranshuparmar/witr?label=Latest%20Release&style=flat-square)](https://github.com/pranshuparmar/witr/releases/latest) [![Platforms](https://img.shields.io/badge/platforms-linux%20%7C%20macos%20%7C%20windows%20%7C%20freebsd-blue?style=flat-square)](#8-platform-support) <br> [![Package Managers](https://img.shields.io/badge/Package%20Managers-brew%20|%20conda%20|%20aur%20|%20winget%20|%20npm%20|%20ports%20|%20...%20-blue?style=flat-square)](https://repology.org/project/witr/versions)

<a href="https://trendshift.io/repositories/18714" target="_blank"><img src="https://trendshift.io/api/badge/repositories/18714" alt="pranshuparmar/witr on Trendshift" width="250" height="55" /></a>
<a href="https://www.producthunt.com/products/witr?embed=true&amp;utm_source=badge-featured&amp;utm_medium=badge&amp;utm_campaign=badge-witr" target="_blank" rel="noopener noreferrer"><img alt="witr - Why is this running? Trace process, port, container or file. | Product Hunt" width="250" height="54" src="https://api.producthunt.com/widgets/embed-image/v1/featured.svg?post_id=1211309&amp;theme=light&amp;t=1785480480150"></a>

### 🎮 [**Try witr in your browser →**](https://pranshuparmar.github.io/witr/)
*Investigate a simulated Linux box — a guided tutorial and free-play sandbox, no install required.*

<a href="https://pranshuparmar.github.io/witr/"><img width="1232" alt="witr's interactive TUI and CLI answering why a node process is running — the same systemd → PM2 → node chain in both" src="https://github.com/user-attachments/assets/dbe271ad-25e5-425b-b414-392d0c4eee37" /></a>

</div>

---

<div align="center">

[**Purpose**](#1-purpose) • [**Installation**](#2-installation) • [**AI Agents**](#3-using-witr-with-ai-coding-agents) • [**TUI**](#4-interactive-mode-tui) • [**Examples**](#5-example-outputs)
<br>
[**Flags**](#6-flags--options) • [**Output Behavior**](#7-output-behavior) • [**Platforms**](#8-platform-support) • [**Sponsors**](#9-sponsors)

</div>

---

## 1. Purpose

**witr** exists to answer a single question:

> **Why is this running?**

When something is running on a system, whether it is a process, a service, or something bound to a port, there is always a cause. That cause is often indirect, non-obvious, or spread across multiple layers such as supervisors, containers, services, or shells.

Existing tools (`ps`, `top`, `lsof`, `ss`, `systemctl`, `docker ps`) expose state and metadata. They show _what_ is running, but leave the user to infer _why_ by manually correlating outputs across tools.

**witr** makes that causality explicit. Ports, services, containers and files all map back to a **PID**, and for that PID witr answers:

1. What is running?
2. How did it start?
3. What is keeping it running?
4. What context does it belong to?

The answer comes as a single human-readable report, as JSON for scripts and AI agents, or in an **interactive TUI dashboard**.

```bash
witr node                 # why is this node process running?
witr --port 8080          # what is holding port 8080, and why?
witr --pid 1234 --tree    # its ancestry and children as a tree
witr --container redis    # a container, across Docker, Podman, Kubernetes and more
witr --port 8080 --json   # the same answer for scripts and AI agents
witr                      # browse everything in the interactive TUI
```

> 📖 Curious how witr came to be? [Read the story](https://medium.com/@pranshu.parmar/witr-why-is-this-running-a9a97cbedd18).

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
- Detect your CPU architecture (`amd64`, `arm64` or `loong64` on Linux)
- Download the latest released binary and man page
- Install it to `/usr/local/bin/witr`
- Install the man page to `/usr/local/share/man/man1/witr.1`

To install somewhere else, such as `~/.local` (no sudo needed), set `INSTALL_PREFIX` on `bash`:

```bash
curl -fsSL https://raw.githubusercontent.com/pranshuparmar/witr/main/install.sh | INSTALL_PREFIX="$HOME/.local" bash
```

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
<summary><strong>APT (Debian, Ubuntu & Derivatives)</strong> <a href="https://packages.debian.org/sid/witr"><img src="https://repology.org/badge/version-for-repo/debian_unstable/witr.svg?style=flat-square" alt="Debian"></a></summary>
<br>


You can install **witr** from the official Debian and Ubuntu repositories (Ubuntu 26.04+, Debian sid and later), as well as derivative distributions like Kali Linux, Devuan, and Raspbian:

```bash
sudo apt install witr
```

> Note: The apt-shipped version may lag the latest GitHub release. For the newest features, use the install script or another installation method.
</details>

<details>
<summary><strong>Homebrew (macOS & Linux)</strong> <a href="https://formulae.brew.sh/formula/witr"><img src="https://img.shields.io/homebrew/v/witr?style=flat-square" alt="Homebrew"></a></summary>
<br>


You can install **witr** using [Homebrew](https://brew.sh/) on macOS or Linux:

```bash
brew install witr
```
</details>

<details>
<summary><strong>MacPorts (macOS)</strong> <a href="https://ports.macports.org/port/witr/"><img src="https://repology.org/badge/version-for-repo/macports/witr.svg?style=flat-square" alt="MacPorts"></a></summary>
<br>


You can install **witr** using [MacPorts](https://www.macports.org/) on macOS:

```bash
sudo port install witr
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
<summary><strong>NPM (Cross-platform)</strong> <a href="https://www.npmjs.com/package/@pranshuparmar/witr"><img src="https://img.shields.io/npm/v/@pranshuparmar/witr?label=npm&color=blue&style=flat-square" alt="NPM"></a></summary>
<br>

You can install **witr** using [npm](https://www.npmjs.com/package/@pranshuparmar/witr):

```bash
npm install -g @pranshuparmar/witr
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
<summary><strong>Brioche (Linux)</strong> <a href="https://github.com/brioche-dev/brioche-packages/tree/main/packages/witr"><img src="https://img.shields.io/badge/dynamic/regex?url=https%3A%2F%2Fraw.githubusercontent.com%2Fbrioche-dev%2Fbrioche-packages%2Fmain%2Fpackages%2Fwitr%2Fproject.bri&search=version%3A%20%22(%5B%5E%22%5D%2B)%22&replace=v%241&label=brioche&color=blue&style=flat-square" alt="Brioche"></a></summary>
<br>

You can install **witr** using [brioche](https://brioche.dev/):

```bash
brioche install -r witr
```
</details>

<details>
<summary><strong>Mise (macOS, Linux & Windows)</strong> <a href="https://github.com/pranshuparmar/witr/releases/latest"><img src="https://img.shields.io/github/v/release/pranshuparmar/witr?label=mise&style=flat-square" alt="Mise"></a></summary>
<br>

You can install **witr** using [mise](https://mise.jdx.dev/):

```bash
mise use github:pranshuparmar/witr
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
[ "$ARCH" = "loongarch64" ] && ARCH="loong64"

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
    throw "Unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)"
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
<summary><strong>Shell Completions</strong></summary>
<br>

`witr` supports tab completion for all flags. To enable it, add the appropriate line to your shell configuration:

**Bash**
```bash
echo 'eval "$(witr completion bash)"' >> ~/.bashrc
source ~/.bashrc
```

**Zsh**
```zsh
echo 'eval "$(witr completion zsh)"' >> ~/.zshrc
source ~/.zshrc
```

**Fish**
```fish
witr completion fish | source
# To make it permanent:
witr completion fish > ~/.config/fish/completions/witr.fish
```

**PowerShell**
```powershell
witr completion powershell | Out-String | Invoke-Expression
# To make it permanent, add the above line to your $PROFILE
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

# Remove the install directory from your User PATH
$InstallDir = "$env:LocalAppData\witr\bin"
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
[Environment]::SetEnvironmentVariable("Path", (($UserPath -split ';') | Where-Object { $_ -and $_ -ne $InstallDir }) -join ';', "User")
```
</details>

---

## 3. Using witr with AI Coding Agents

AI coding agents (Claude Code, Codex, Cursor and others) regularly run into ports that are already in use, leftover dev servers and confusing containers, and work around them by chaining `lsof`, `ps`, `netstat` and `docker ps`. witr answers the same questions in one command, and two things make it easy for an agent to use:

- **`--json`** prints the full report (the process, its ancestry chain, the source that started it and any warnings), and a failed lookup as `{Target, Error}`. Every shape is listed in [7.3 JSON Output](#73-json-output).
- **Exit codes** say what happened without parsing any text (see [7.1 Exit Codes](#71-exit-codes)). Exit code `1` means the process was found and has warnings; it is not a failure.

Add a short note like this to your project's agent instructions (`AGENTS.md`, `CLAUDE.md` or similar):

```markdown
## Process and port debugging

Use `witr` instead of chaining lsof/ps/netstat/docker commands:

- Port already in use: `witr --port <PORT> --json`
- Unknown or stuck process: `witr <name> --json`, or `witr --pid <PID> --tree`
- Container: `witr --container <name or ID> --json`

Exit codes: 0 found, 1 found with warnings (not a failure), 2 not found,
3 permission denied (retry with sudo), 4 ambiguous name or bad input
(re-run with --pid), 5 internal error, 6 found but what started it can't
be traced. Run `witr --help` for all options.
```

### Official agent skill

For fuller guidance, install the official skill: it tells an agent which command fits the situation, how to read the JSON and exit codes, and to ask before stopping anything witr finds. The `witr` binary still needs to be installed.

- **Claude Code:** run `/plugin marketplace add pranshuparmar/witr`, then `/plugin install witr@witr`.
- **Other agents that read Agent Skills, or a manual install:** copy [`plugins/witr/skills/witr`](plugins/witr/skills/witr) into the agent's skills folder (for Claude Code, `~/.claude/skills/`).

---

## 4. Interactive Mode (TUI)

Running `witr` without any arguments or with the `-i` flag launches the **Interactive Mode (TUI)**. This provides a real-time, terminal-based dashboard with four tabs for exploring processes, ports, containers, and file locks. Combine `-i` with a target to open the TUI already focused on it: `witr -i -p 1234` selects that PID, `witr -i nginx` pre-fills the process filter, and `witr -i -o 5432` / `-c web` / `-f /path` open the Ports / Containers / Locks tab with the filter set.

### Key Features:
- **Processes Tab**: Live, sortable, filterable list of all running processes with a side panel showing the ancestry tree of the highlighted process.
- **Ports Tab**: Open/listening ports with the owning processes attached in a side panel. Toggle between LISTEN-only and ALL with `a`. Enter on an owner opens it, or the container a port is published for; an owner hidden from your user shows the user it runs as (run with sudo to see the process).
- **Containers Tab**: All running containers across Docker, Podman, nerdctl, K8s/crictl, Incus, LXC, LXD, and FreeBSD jails in one list - name, image, status, ports, command, plus a per-container detail view with mounts, networks, and compose project metadata.
- **Locks Tab**: System-wide file locks (POSIX/FLOCK on Linux, from `lsof` on macOS and `fstat` on FreeBSD). Press `a` to switch into "all open files" mode, where locked entries are merged with every interesting open fd; type into `/` to search across the merged set.
- **Process Details**: Deep-dive into a process to see its full ancestry tree, child processes, environment variables, working directory, sockets, file context, and more.
- **Process Actions**: Send signals (Kill, Terminate, Pause, Resume) or Renice processes directly from the UI (Unix only). Press `a` on a process in the list or in its detail view; the confirmation names the exact process before anything is sent.
- **Mouse Support**: Navigate, sort columns, and click rows using your mouse.
- **Adaptive Theme**: Colors adapt automatically to light and dark terminal backgrounds.
- **Auto-Refresh**: The process, port, container, and lock lists refresh automatically on an adaptive cadence (starts at 3 seconds, backing off under load).

---

## 5. Example Outputs

> 💡 Prefer learning by doing? These examples come from the simulated box in the [interactive browser tutorial](https://pranshuparmar.github.io/witr/), where you can run them yourself — no install required.

### 5.1 Name Based Query

```bash
witr node
```

```
Target      : node

Process     : node (pid 14233)
User        : deploy
Command     : node /srv/expense-manager/index.js
Started     : 2 days ago (Fri 2026-10-02 10:28:38 +00:00)
Restarts    : 3

Why It Exists :
  systemd (pid 1) → PM2 v5.3.1: God (pid 5034) → node (pid 14233)

Source      : pm2-deploy.service (systemd)
Description : PM2 process manager for user deploy (app: expense-manager)
Unit File   : /etc/systemd/system/pm2-deploy.service

Working Dir : /srv/expense-manager
Git Repo    : expense-manager (main)
Sockets     : 127.0.0.1:5000 (TCP | LISTENING, 2 connections)
              127.0.0.1:51744 → 127.0.0.1:5432 (TCP | ESTABLISHED)
```

---

### 5.2 Short Output

```bash
witr --port 5000 --short
```

```
systemd (pid 1) → PM2 v5.3.1: God (pid 5034) → node (pid 14233)
```

---

### 5.3 Tree Output

```bash
witr --pid 40141 --tree
```

```
systemd (pid 1)
  └─ sshd (pid 711)
    └─ sshd (pid 40122)
      └─ sshd (pid 40140)
        └─ bash (pid 40141)
          ├─ python3 (pid 8123)
          └─ ngrok (pid 14290)
```

Note: _Tree view includes child processes (up to 10) and highlights the target process._

---

### 5.4 Multiple Matches

```bash
witr ssh
```

```
Multiple matching processes found:

[1] sshd (pid 711)
    sshd: /usr/sbin/sshd -D [listener] 0 of 10-100 startups
[2] ssh (pid 27311)
    ssh -l backup 10.0.4.20 rsync --server -logDtpre.iLsfxCIvu --partial . /backups/uploads/
[3] sshd (pid 40122)
    sshd: deploy [priv]
[4] sshd (pid 40140)
    sshd: deploy@pts/0

Re-run with:
  witr --pid <pid>
```

Names match as substrings, so `ssh` also finds every `sshd`. To match only processes named exactly `ssh`, use the `--exact` flag:

```bash
witr ssh -x
```

---

### 5.5 File Based Query

```bash
witr --file /var/lib/dpkg/lock
```

```
Target      : unattended-upgr

Process     : unattended-upgr (pid 33871)
User        : root
Command     : /usr/bin/python3 /usr/bin/unattended-upgrade --download-only
Started     : 4 min ago (Sun 2026-10-04 10:24:38 +00:00)

Why It Exists :
  systemd (pid 1) → unattended-upgr (pid 33871)

Source      : apt-daily-upgrade.service (systemd)
Description : Daily apt upgrade and clean activities
Unit File   : /lib/systemd/system/apt-daily-upgrade.service
```

---

### 5.6 Container Based Query

```bash
witr --container api
```

```
Target      : gunicorn

Process     : gunicorn (pid 3141)
User        : root
Container   : docker: expense-manager-api-1 (id 7c41e0b9a2d3)
Image       : expense-manager-api:1.4.2
Published   : 0.0.0.0:8081->8080/tcp
Compose File: /srv/expense-manager/docker-compose.yml
Compose Dir : /srv/expense-manager
Command     : /usr/local/bin/python /usr/local/bin/gunicorn app:app --bind 0.0.0.0:8080 --workers 2
Started     : 5 days ago (Tue 2026-09-29 10:28:38 +00:00)
Restarts    : 1 (policy: unless-stopped)

Why It Exists :
  systemd (pid 1) → containerd-shim (pid 3120) → gunicorn (pid 3141)

Source      : docker (container)

Working Dir : /app

Warnings    :
  • Process is running as root
  • Container has no healthcheck configured
```

Looks up a container by name, ID, image, command, or compose project/service across every detected runtime (Docker, Podman, nerdctl, K8s/crictl, Incus, LXC, LXD, FreeBSD jails). The output includes the container's image and, for Compose services, the Compose file and project directory on the host. Pass `--verbose` to include mounts.

---

### 5.7 Multiple Inputs

```bash
witr node --port 5432 --pid 8123
```

```
----- [name: node] -----
Target      : node

Process     : node (pid 14233)
...

----- [port: 5432] -----
Target      : postgres

Process     : postgres (pid 2010)
...

----- [pid: 8123] -----
Target      : python3

Process     : python3 (pid 8123)
...
```

All target flags are repeatable and can be mixed. Results appear in the order you typed them. All output modes (`--short`, `--tree`, `--json`, `--env`, `--warnings`, `--verbose`) work with multiple inputs.

---

### 5.8 Warnings, Verbose and JSON

Warnings close the standard report (see 5.6). `--warnings` shows only them:

```bash
witr --pid 8123 --warnings
```

```
Process     : python3 (pid 8123)
Command     : python3 -m http.server 8000
Warnings    :
  • Process is listening on a public interface
```

<details>
<summary><strong><code>--verbose</code></strong>: adds memory, I/O, file descriptors and threads to the standard report</summary>
<br>

```bash
witr node --verbose
```

```
...                       (the report from 5.1)

Memory:
  Virtual  : 1.1 GB
  Resident : 179.0 MB
  Shared   : 40.0 MB

I/O Statistics:
  Read  : 15.0 MB (2140 ops)
  Write : 4.0 MB (512 ops)

File Descriptors: 34/1024

Threads: 11
```
</details>

<details>
<summary><strong><code>--json</code></strong>: the same report for scripts and AI agents (trimmed)</summary>
<br>

```bash
witr --port 5000 --json
```

```
{
  "Target": { "Type": "port", "Value": "5000" },
  "ResolvedTarget": "node",
  "Process": {
    "PID": 14233,
    "PPID": 5034,
    "Command": "node",
    "Cmdline": "node /srv/expense-manager/index.js",
    "StartedAt": "2026-10-02T10:28:38Z",
    "User": "deploy",
    "WorkingDir": "/srv/expense-manager",
    "GitRepo": "expense-manager",
    "GitBranch": "main",
    "Sockets": [
      { "Port": 5000, "Address": "127.0.0.1", "State": "LISTEN", "Protocol": "TCP" },
      ...
    ],
    ...
  },
  "RestartCount": 3,
  "Ancestry": [
    { "PID": 1, "Command": "systemd", ... },
    { "PID": 5034, "Command": "PM2 v5.3.1: God", ... },
    { "PID": 14233, "Command": "node", ... }
  ],
  "Source": {
    "Type": "systemd",
    "Name": "pm2-deploy.service",
    "Description": "PM2 process manager for user deploy (app: expense-manager)",
    "UnitFile": "/etc/systemd/system/pm2-deploy.service",
    "Details": { "NRestarts": "3" }
  },
  "Warnings": null,
  ...
}
```

Every JSON shape is listed in [7.3 JSON Output](#73-json-output).
</details>

---

## 6. Flags & Options

```
  -c, --container strings container(s) to look up (repeatable)
      --env              show environment variables for the process
  -x, --exact            use exact name matching (no substring search)
  -f, --file strings     file(s) held open by a process (repeatable)
  -h, --help             help for witr
  -i, --interactive      interactive mode (TUI)
      --json             show result as JSON
      --no-color         disable colorized output
  -p, --pid strings      pid(s) to look up (repeatable)
  -o, --port strings     port(s) to look up (repeatable)
  -s, --short            show only ancestry
  -t, --tree             show only ancestry as a tree
      --verbose          show extended process information
  -v, --version          version for witr
      --warnings         show only warnings
```

Positional arguments (without flags) are treated as process or service names. Multiple names can be passed. By default, name matching uses substring matching (fuzzy search). Use `--exact` to match only processes with the exact name.

All target flags (`--pid`, `--port`, `--file`, `--container`) are repeatable and can be mixed with each other and with positional name arguments. When multiple targets are provided, results are shown sequentially with labeled dividers. All output modes (standard, short, tree, JSON, env, warnings, verbose) work with multiple inputs.

The `--container` flag searches across Docker, Podman, nerdctl, K8s/crictl, Incus, LXC, LXD, and FreeBSD jails, and matches against container name, image, command, and compose project/service labels, or a container ID (full, short, or a prefix of at least 4 characters).

The TUI is launched if no arguments or relevant flags (`--pid`, `--port`, `--file`, `--container`) are provided, or if the `--interactive` flag is explicitly used. It needs a terminal: run without one (from a script, a pipe or CI) and witr exits with code 4 and asks for a target instead. An output mode with no target (`witr --json`, `--short`, `--tree`, `--warnings`, `--verbose` or `--env`) also exits with code 4 and asks for a target, rather than opening the TUI.

---

## 7. Output Behavior

### 7.1 Exit Codes

witr returns meaningful exit codes for use in scripts, CI pipelines, and monitoring:

| Code | Meaning |
|------|---------|
| 0 | Clean: process found, no warnings |
| 1 | Warnings: process found but has one or more warnings |
| 2 | Not found: no matching process or service (also a port no process on this system holds, when witr runs as root or on Windows) |
| 3 | Permission denied: insufficient privileges (retry with sudo) |
| 4 | Invalid input: bad arguments or ambiguous match |
| 5 | Internal error: an unexpected failure occurred |
| 6 | Cause unknown: process found, but what started it can't be traced (its warnings say why). Not used on Windows, where this is routine |

With several targets, the most severe result wins. Cause unknown ranks above warnings but below every failure (2 to 5).

#### Example Usage:

```bash
witr nginx --short
case $? in
  0) echo "All clear" ;;
  1) echo "Warnings detected" ;;
  2) echo "Process not running" ;;
  3) echo "Need elevated privileges" ;;
  4) echo "Invalid input or ambiguous match" ;;
  5) echo "Internal error" ;;
  6) echo "Cannot tell what started it" ;;
esac
```

---

### 7.2 Standard Output Sections

#### Target

What the user asked about.

#### Process

Executable, PID, user, command, start time and restart count. The restart count comes from the managing system: a systemd unit, or the container runtime (Docker, Podman, nerdctl and Kubernetes), which also shows the container's restart policy, e.g. `Restarts : 12 (policy: unless-stopped)`.

On Windows the user carries the process's integrity level when it isn't the normal Medium: `(elevated)` for a process run as administrator, `(system integrity)`, `(low integrity)` or `(untrusted integrity)` for sandboxed ones. On Linux, a `Security` line shows the AppArmor profile or SELinux context confining the process; unconfined processes have none.

#### Why It Exists

A causal ancestry chain showing how the process came to exist.
This is the core value of witr.

When the process that started it has exited, the chain marks the break with `? (original parent exited)` (or `? (parent pid N exited)` when that parent's PID no longer exists or now belongs to an unrelated process) instead of crediting whatever adopted it.

#### Source

The primary system responsible for starting or supervising the process (best effort).

Examples:

- systemd unit with schedule info for timer-triggered services (Linux)
- launchd service with schedule/trigger details (macOS)
- SSH session (with remote IP and terminal)
- docker container
- pm2
- cron
- interactive shell (detects tmux/screen sessions)
- Snap/Flatpak sandbox (Linux)

Only **one primary source** is selected. If the original parent has exited and nothing about the process itself (its container, service unit, login session or app scope, or launchd job) explains it, the source is reported as `unknown` rather than guessed. On systems without such a service manager it stays `init`, with a note that init only adopted it.

#### Context (best effort)

- Working directory
- Git repository name and branch
- Container name / image (docker, podman, kubernetes, colima, containerd)
- Sockets: listeners first, each with the number of connections it accepted, then other sockets; a connection shows `local → remote`
- Public vs private bind

#### Warnings

Non‑blocking observations:

- Process is a zombie (defunct) or stopped (T state)
- Process is using high CPU (>2h total) or high memory (>1GB RSS)
- Process is listening on a public interface (0.0.0.0 / ::)
- Process is running as root
- Dangerous Linux capabilities on a non-root process (CAP_SYS_ADMIN, CAP_SYS_PTRACE, etc.)
- Service or container has restarted more than 5 times
- No known supervisor or service manager detected
- Original parent process has exited, so what started the process can't be traced (shown instead of the one above)
- Process has been running for over 90 days
- Process is running from a suspicious working directory (`/`, `/tmp` or `/var/tmp`)
- Container has no healthcheck configured
- Service name and process name do not match
- Process is running from a deleted binary
- Library injection indicators (`LD_PRELOAD`, `DYLD_*`)

---

### 7.3 JSON Output

`--json` is meant for scripts and tools, and its output is a contract:

- **Stable:** field names, what they mean, and their types. Releases may add fields, so ignore any you don't recognise.
- **Not part of the contract:** field order, whitespace, and the wording of human-readable text (`Error`, `Note`, `Description` and warning messages). Match on fields and exit codes, not on sentences.
- **Empty values** may be `null`, an empty list, or left out.
- **Breaking changes** (renaming, removing or retyping a field) only ship in a release whose notes call them out.

| Command | Output |
|---------|--------|
| `witr <target> --json` | The full report: `Target`, `Process`, `Ancestry`, `Source`, `Warnings` and the rest |
| `--short --json` | A list of `{PID, Command}` from the top of the chain to the process, with `PPID` and `ParentExited` where the process that started it has exited |
| `--tree --json` | `{Ancestry, Children}`, each a list of the same entries |
| `--warnings --json` | `{PID, Process, Command, Warnings}` |
| `--env --json` | `{PID, Process, Command, Env}` |
| A container whose processes aren't visible | `{Target, Runtime, ContainerID, ContainerName, Image, …, Note}` |
| A failed lookup | `{Target, Error}`, plus `Matches` when the target was ambiguous |
| Several targets | A list of the above, one entry per target, in the order given |

A test pins every field of every shape, so an accidental rename fails the build.

---

## 8. Platform Support

- **Linux** (x86_64, arm64, loong64) - Full feature support (`/proc`).
- **macOS** (x86_64, arm64) - Uses `ps`, `lsof`, `netstat`, `sysctl`, `launchctl`.
- **Windows** (x86_64, arm64) - Native Win32 APIs (ToolHelp32, PSAPI, Service Control Manager). No PowerShell or WMI dependency.
- **FreeBSD** (x86_64, arm64) - Uses `procstat`, `ps`, `sockstat`, `fstat`.

---

### 8.1 Feature Compatibility Matrix

| Feature | Linux | macOS | Windows | FreeBSD | Notes |
|---------|:-----:|:-----:|:-------:|:-------:|-------|
| **Process Selection** |
| By Name | ✅ | ✅ | ✅ | ✅ | |
| By PID | ✅ | ✅ | ✅ | ✅ | |
| By Port | ✅ | ✅ | ✅ | ✅ | |
| By File | ✅ | ✅ | ✅ | ✅ | |
| By Container | ✅ | ✅ | ✅ | ✅ | Requires the runtime CLI on PATH (docker/podman/nerdctl/crictl/incus/lxc/lxc-ls/jls). |
| Multiple/mixed inputs | ✅ | ✅ | ✅ | ✅ | Repeatable flags, mixed types. |
| Exact Match | ✅ | ✅ | ✅ | ✅ | |
| Full command line | ✅ | ✅ | ✅ | ✅ | |
| Process start time | ✅ | ✅ | ✅ | ✅ | |
| Working directory | ✅ | ✅ | ✅ | ✅ | |
| Environment variables | ✅ | ⚠️ | ⚠️ | ✅ | macOS: SIP restrictions; Windows: protected processes inaccessible. |
| **Network** |
| Listening ports | ✅ | ✅ | ✅ | ✅ | |
| Connections (remote end) | ✅ | ✅ | ✅ | ✅ | Connections show their remote end; connections a listener accepted are counted on its row. |
| Bind addresses | ✅ | ✅ | ✅ | ✅ | |
| Port → PID resolution | ✅ | ✅ | ✅ | ✅ | |
| Port → Container fallback | ✅ | ✅ | ✅ | ✅ | A port held by PID 1 (socket activation), a port forwarder (`docker-proxy`, Docker Desktop, `rootlessport`, `pasta`, RootlessKit) or no process at all is traced to the service or container behind it, in the CLI and the TUI's Ports tab. |
| **Service Detection** |
| Service Manager | ✅ | ✅ | ✅ | ✅ | Linux: systemd, macOS: launchd, Windows: Services, FreeBSD: rc.d |
| Service Description | ✅ | ✅ | ✅ | ✅ | Linux: `Description`, macOS: `Comment`, Windows: `Display Name`, FreeBSD: `rc` header |
| Configuration Source | ✅ | ✅ | ✅ | ✅ | Linux: Unit File, macOS: Plist, Windows: Registry Key, FreeBSD: Rc Script |
| Supervisor | ✅ | ✅ | ✅ | ✅ | |
| Containers | ✅ | ✅ | ✅ | ✅ | Docker (plus compose mappings), Podman, nerdctl, K8s (Kubepods/crictl), Containerd. Colima on macOS/Linux. Incus/LXC/LXD on Linux. Jails on FreeBSD. |
| SSH session detection | ✅ | ✅ | ✅ | ✅ | Detects remote IP and terminal. |
| tmux/screen detection | ✅ | ✅ | ❌ | ✅ | Shows session name in source. |
| Schedule detection | ✅ | ✅ | ❌ | ❌ | Linux: systemd timers, macOS: launchd intervals/calendar. |
| Restart count | ✅ | ⚠️ | ⚠️ | ❌ | systemd restarts on Linux; container restart count and policy wherever Docker, Podman, nerdctl or Kubernetes runs. |
| Exited parent detection | ✅ | ✅ | ⚠️ | ✅ | Marks where the process that started it exited. Windows shows the break but doesn't treat it as a finding (no exit code 6): its launchers routinely exit. |
| Snap/Flatpak detection | ✅ | ❌ | ❌ | ❌ | |
| **Health & Diagnostics** |
| CPU usage detection | ✅ | ✅ | ✅ | ✅ | |
| Memory usage detection | ✅ | ✅ | ✅ | ✅ | |
| Health status detection | ✅ | ✅ | ✅ | ✅ | |
| Open Files / Handles | ✅ | ✅ | ⚠️ | ✅ | Windows: count only. |
| File Locks | ✅ | ✅ | ❌ | ✅ | Linux: `/proc/locks`; macOS/FreeBSD: derived from `lsof`/`fstat`. |
| Deleted binary detection | ✅ | ✅ | ✅ | ✅ | Warns if executable is missing. |
| Capability warnings | ✅ | ❌ | ❌ | ❌ | Warns about dangerous capabilities on non-root processes. |
| Security context | ✅ | ❌ | ✅ | ❌ | Linux: AppArmor profile or SELinux context; Windows: integrity level (elevated, system, low). |
| **Context** |
| Git repo/branch detection | ✅ | ✅ | ✅ | ✅ | |
| **Interactive Mode (TUI)** |
| Processes Tab | ✅ | ✅ | ✅ | ✅ | |
| Ports Tab | ✅ | ✅ | ✅ | ✅ | |
| Containers Tab | ✅ | ✅ | ✅ | ✅ | |
| Locks Tab | ✅ | ✅ | ❌ | ✅ | Toggle (`a`) shows all open files. |
| Process Details | ✅ | ✅ | ✅ | ✅ | |
| Process Actions | ✅ | ✅ | ❌ | ✅ | |

**Legend:** ✅ Full support | ⚠️ Partial/limited support | ❌ Not available

---

### 8.2 Permissions Note

#### Linux/FreeBSD

witr inspects system directories which may require elevated permissions.

If you are not seeing the expected information, try running witr with sudo:

```bash
sudo witr [your arguments]
```

Without root, a port held by another user's process still shows which user it belongs to (`it belongs to postgres`), but not the process. With root, a port that no process on the system holds is reported as such (exit code 2): on WSL that's usually a process in another distro, since all distros share one network.

#### macOS

Some operations may require elevated permissions:

```bash
sudo witr [your arguments]
```

Note: Due to macOS System Integrity Protection (SIP), some system process details may not be accessible even with sudo.

#### Windows

On Windows, witr talks directly to Win32 APIs (ToolHelp32, PSAPI, Service Control Manager) rather than spawning PowerShell or WMI, so startup is fast and there's no `Get-CimInstance` hang. To see details for processes owned by other users or system services, you must run the terminal as **Administrator**.

```powershell
# Run in Administrator PowerShell
.\witr.exe [your arguments]
```

---

## 9. Sponsors

Special thanks to the people who supported **witr** ❤️

<p>
  <a href="https://github.com/timcolson" title="Tim Colson">
    <img src="https://images.weserv.nl/?url=github.com/timcolson.png&mask=circle&w=80&h=80" width="80">
  </a>
  <a href="https://github.com/R-Bose" title="Rijurekh Bose">
    <img src="https://images.weserv.nl/?url=github.com/R-Bose.png&mask=circle&w=80&h=80" width="80">
  </a>
</p>
