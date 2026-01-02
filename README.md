# witr (why-is-this-running)

<img width="631" height="445" alt="witr" src="https://github.com/user-attachments/assets/e51cace3-0070-4200-9d1f-c4c9fbc81b8d" />

---

## Table of Contents

- [1. Purpose](#1-purpose)
- [2. Goals](#2-goals)
- [3. Core Concept](#3-core-concept)
- [4. Supported Targets](#4-supported-targets)
  - [4.1 Name (process or service)](#41-name-process-or-service)
  - [4.2 PID](#42-pid)
  - [4.3 Port](#43-port)
- [5. Output Behavior](#5-output-behavior)
  - [5.1 Output Principles](#51-output-principles)
  - [5.2 Standard Output Sections](#52-standard-output-sections)
- [6. Flags & Options](#6-flags--options)
- [7. Example Outputs](#7-example-outputs)
  - [7.1 Name Based Query](#71-name-based-query)
  - [7.2 Short Output](#72-short-output)
  - [7.3 Tree Output](#73-tree-output)
  - [7.4 Multiple Matches](#74-multiple-matches)
- [8. Installation](#8-installation)
  - [8.1 Homebrew (macOS & Linux)](#81-homebrew-macos--linux)
  - [8.2 Arch Linux (AUR)](#82-arch-linux-aur)
  - [8.3 Script Installation (Recommended)](#83-script-installation-recommended)
  - [8.4 Manual Installation](#84-manual-installation)
  - [8.5 Verify Installation](#85-verify-installation)
  - [8.6 Uninstallation](#86-uninstallation)
  - [8.7 Nix Flake](#87-nix-flake)
- [9. Platform Support](#9-platform-support)
  - [9.1 Feature Compatibility Matrix](#91-feature-compatibility-matrix)
  - [9.2 Permissions Note](#92-permissions-note)
- [10. Success Criteria](#10-success-criteria)
- [11. AI Assistance Disclaimer](#11-ai-assistance-disclaimer)

---

## 1. Purpose

**witr** exists to answer a single question:

> **Why is this running?**

When something is running on a system—whether it is a process, a service, or something bound to a port—there is always a cause. That cause is often indirect, non-obvious, or spread across multiple layers such as supervisors, containers, services, or shells.

Existing tools (`ps`, `top`, `lsof`, `ss`, `systemctl`, `docker ps`) expose state and metadata. They show _what_ is running, but leave the user to infer _why_ by manually correlating outputs across tools.

**witr** makes that causality explicit.

It explains **where a running thing came from**, **how it was started**, and **what chain of systems is responsible for it existing right now**, in a single, human-readable output.

---

## 2. Goals

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

## 3. Core Concept

witr treats **everything as a process question**.

Ports, services, containers, and commands all eventually map to **PIDs**. Once a PID is identified, witr builds a causal chain explaining _why that PID exists_.

At its core, witr answers:

1. What is running?
2. How did it start?
3. What is keeping it running?
4. What context does it belong to?

---

## 4. Supported Targets

witr supports multiple entry points that converge to PID analysis.

---

### 4.1 Name (process or service)

```bash
witr node
witr nginx
```

A single positional argument (without flags) is treated as a process or service name. If multiple matches are found, witr will prompt for disambiguation by PID.

---

### 4.2 PID

```bash
witr --pid 14233
```

Explains why a specific process exists.

---

### 4.3 Port

```bash
witr --port 5000
```

Explains the process(es) listening on a port.

---

## 5. Output Behavior

### 5.1 Output Principles

- Single screen by default (best effort)
- Deterministic ordering
- Narrative-style explanation
- Best-effort detection with explicit uncertainty

---

### 5.2 Standard Output Sections

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

## 6. Flags & Options

```
--pid <n>         Explain a specific PID
--port <n>        Explain port usage
--short           One-line summary
--tree            Show full process ancestry tree
--json            Output result as JSON
--warnings        Show only warnings
--no-color        Disable colorized output
--env             Show only environment variables for the process
--help            Show this help message
```

A single positional argument (without flags) is treated as a process or service name.

---

## 7. Example Outputs

### 7.1 Name Based Query

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

### 7.2 Short Output

```bash
witr --port 5000 --short
```

```
systemd (pid 1) → PM2 v5.3.1: God (pid 1481580) → python (pid 1482060)
```

---

### 7.3 Tree Output

```bash
witr --pid 1482060 --tree
```

```
systemd (pid 1)
  └─ PM2 v5.3.1: God (pid 1481580)
    └─ python (pid 1482060)
```

---

### 7.4 Multiple Matches

#### 7.4.1 Multiple Matching Processes

```bash
witr node
```

```
Multiple matching processes found:

[1] PID 12091  node server.js  (docker)
[2] PID 14233  node index.js   (pm2)
[3] PID 18801  node worker.js  (manual)

Re-run with:
  witr --pid <pid>
```

---

#### 7.4.2 Ambiguous Name (process and service)

```bash
witr nginx
```

```
Ambiguous target: "nginx"

The name matches multiple entities:

[1] PID 2311   nginx: master process   (service)
[2] PID 24891  nginx: worker process   (manual)

witr cannot determine intent safely.
Please re-run with an explicit PID:
  witr --pid <pid>
```

---

## 8. Installation

witr is distributed as a single static binary for Linux and macOS.

---


### 8.1 Homebrew (macOS & Linux)

You can install **witr** using [Homebrew](https://brew.sh/) on macOS or Linux:

```bash
brew install witr
```

See the [Homebrew Formula page](https://formulae.brew.sh/formula/witr#default) for more details.

### 8.2 Arch Linux (AUR)

On Arch Linux and derivatives, install from the [AUR package](https://aur.archlinux.org/packages/witr-bin):

```bash
yay -S witr-bin
# or use your preferred AUR helper
```

---

### 8.3 Script Installation (Recommended)

The easiest way to install **witr** is via the install script.

#### Quick install

```bash
curl -fsSL https://raw.githubusercontent.com/SanCognition/witr/main/install.sh | bash
```

#### Review before install

```bash
curl -fsSL https://raw.githubusercontent.com/SanCognition/witr/main/install.sh -o install.sh
cat install.sh
chmod +x install.sh
./install.sh
```

The script will:

- Detect your operating system (`linux` or `darwin`/macOS)
- Detect your CPU architecture (`amd64` or `arm64`)
- Download the latest released binary and man page
- Install it to `/usr/local/bin/witr`
- Install the man page to `/usr/local/share/man/man1/witr.1`
- Pass INSTALL_PREFIX to override default install path

You may be prompted for your password to write to system directories.

### 8.4 Manual Installation

If you prefer manual installation, follow these simple steps for your platform:

#### Linux amd64 (most PCs/servers):

```bash
# Download the binary
curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/witr-linux-amd64 -o witr-linux-amd64

# Verify checksum (Optional, should print OK)
curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/SHA256SUMS -o SHA256SUMS
grep witr-linux-amd64 SHA256SUMS | sha256sum -c -

# Rename and install
mv witr-linux-amd64 witr && chmod +x witr
sudo mv witr /usr/local/bin/witr

# Install the man page (Optional)
sudo curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/witr.1 -o /usr/local/share/man/man1/witr.1
sudo mandb >/dev/null 2>&1 || true
```

#### Linux arm64 (Raspberry Pi, ARM servers):

```bash
# Download the binary
curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/witr-linux-arm64 -o witr-linux-arm64

# Verify checksum (Optional, should print OK)
curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/SHA256SUMS -o SHA256SUMS
grep witr-linux-arm64 SHA256SUMS | sha256sum -c -

# Rename and install
mv witr-linux-arm64 witr && chmod +x witr
sudo mv witr /usr/local/bin/witr

# Install the man page (Optional)
sudo curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/witr.1 -o /usr/local/share/man/man1/witr.1
sudo mandb >/dev/null 2>&1 || true
```

#### macOS arm64 (Apple Silicon - M1/M2/M3):

```bash
# Download the binary
curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/witr-darwin-arm64 -o witr-darwin-arm64

# Verify checksum (Optional, should print OK)
curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/SHA256SUMS -o SHA256SUMS
grep witr-darwin-arm64 SHA256SUMS | shasum -a 256 -c -

# Rename and install
mv witr-darwin-arm64 witr && chmod +x witr
sudo mv witr /usr/local/bin/witr

# Install the man page (Optional)
sudo mkdir -p /usr/local/share/man/man1
sudo curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/witr.1 -o /usr/local/share/man/man1/witr.1
```

#### macOS amd64 (Intel Macs):

```bash
# Download the binary
curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/witr-darwin-amd64 -o witr-darwin-amd64

# Verify checksum (Optional, should print OK)
curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/SHA256SUMS -o SHA256SUMS
grep witr-darwin-amd64 SHA256SUMS | shasum -a 256 -c -

# Rename and install
mv witr-darwin-amd64 witr && chmod +x witr
sudo mv witr /usr/local/bin/witr

# Install the man page (Optional)
sudo mkdir -p /usr/local/share/man/man1
sudo curl -fsSL https://github.com/SanCognition/witr/releases/latest/download/witr.1 -o /usr/local/share/man/man1/witr.1
```

**Explanation:**

- Download only the binary for your platform/architecture and the SHA256SUMS file.
- Verify the checksum for your binary only (prints OK if valid).
- Rename to witr, make it executable, and move to your PATH.
- Install man page.

### 8.5 Verify Installation:

```bash
witr --version
man witr
```

### 8.6 Uninstallation

To completely remove **witr**:

```bash
sudo rm -f /usr/local/bin/witr
sudo rm -f /usr/local/share/man/man1/witr.1
```

### 8.7 Nix Flake

If you use Nix, you can build **witr** from source and run without installation:

```bash
nix run github:SanCognition/witr -- --help
```

---

## 9. Platform Support

- **Linux** (x86_64, arm64) - Uses `/proc` filesystem for process information
- **macOS** (x86_64, arm64) - Uses `ps`, `lsof`, and `sysctl` for process information

---

### 9.1 Feature Compatibility Matrix

| Feature | Linux | macOS | Notes |
|---------|:-----:|:-----:|-------|
| **Process Inspection** |
| Basic process info (PID, PPID, user, command) | ✅ | ✅ | |
| Full command line | ✅ | ✅ | |
| Process start time | ✅ | ✅ | |
| Working directory | ✅ | ✅ | Linux: `/proc`, macOS: `lsof` |
| Environment variables | ✅ | ⚠️ | macOS: partial via `ps -E`, limited by SIP |
| **Network** |
| Listening ports | ✅ | ✅ | |
| Bind addresses | ✅ | ✅ | |
| Port → PID resolution | ✅ | ✅ | Linux: `/proc/net/tcp`, macOS: `lsof`/`netstat` |
| **Service Detection** |
| systemd | ✅ | ❌ | Linux only |
| launchd | ❌ | ✅ | macOS only |
| Supervisor | ✅ | ✅ | |
| Cron | ✅ | ✅ | |
| Containers | ✅ | ⚠️ | macOS: Docker Desktop, Podman, Colima run in VM |
| **Health & Diagnostics** |
| CPU usage detection | ✅ | ✅ | |
| Memory usage detection | ✅ | ✅ | |
| Zombie process detection | ✅ | ✅ | |
| **Context** |
| Git repo/branch detection | ✅ | ✅ | |
| Container detection | ✅ | ⚠️ | macOS: limited to Docker Desktop, Podman, Colima |

**Legend:** ✅ Full support | ⚠️ Partial/limited support | ❌ Not available

---

### 9.2 Permissions Note

#### Linux

witr inspects `/proc` and may require elevated permissions to explain certain processes.

If you are not seeing the expected information (e.g., missing process ancestry, user, working directory or environment details), try running witr with sudo for elevated permissions:

```bash
sudo witr [your arguments]
```

#### macOS

On macOS, witr uses `ps`, `lsof`, and `launchctl` to gather process information. Some operations may require elevated permissions:

```bash
sudo witr [your arguments]
```

Note: Due to macOS System Integrity Protection (SIP), some system process details may not be accessible even with sudo.

---

## 10. Success Criteria

witr is successful if:

- A user can answer "why is this running?" within seconds
- It reduces reliance on multiple tools
- Output is understandable under stress
- Users trust it during incidents

---

## 11. AI Assistance Disclaimer

This project was developed with assistance from AI/LLMs (including GitHub Copilot, ChatGPT, and related tools), supervised by humans who occasionally knew what they were doing.

---
