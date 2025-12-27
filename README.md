# witr (why-is-this-running)

![witr](https://github.com/user-attachments/assets/adebb063-4706-47a8-9f77-612298354ef7)

---

## 1. Purpose

**witr** exists to answer a single question:

> **Why is this running?**

When something is running on a system—whether it is a process, a service, or something bound to a port—there is always a cause. That cause is often indirect, non-obvious, or spread across multiple layers such as supervisors, containers, services, or shells.

Existing tools (`ps`, `top`, `lsof`, `ss`, `systemctl`, `docker ps`) expose state and metadata. They show *what* is running, but leave the user to infer *why* by manually correlating outputs across tools.

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

Ports, services, containers, and commands all eventually map to **PIDs**. Once a PID is identified, witr builds a causal chain explaining *why that PID exists*.

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


### 5.1 Output principles
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
- systemd unit
- docker container
- pm2
- cron
- interactive shell

Only **one primary source** is selected.

#### Context (best effort)
- Working directory
- Git repository name and branch
- Docker container name / image
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

### 7.1 Name-based query

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

### 7.2 Short output

```bash
witr --port 5000 --short
```

```
systemd (pid 1) → PM2 v5.3.1: God (pid 1481580) → python (pid 1482060)
```

---

### 7.3 Tree output

```bash
witr --pid 1482060 --tree
```

```
systemd (pid 1)
  └─ PM2 v5.3.1: God (pid 1481580)
    └─ python (pid 1482060)
```

---

### 7.4 Multiple matches

#### 7.4.1 Multiple matching processes

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

#### 7.4.2 Ambiguous name (process and service)

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

- **Linux:** Downloads prebuilt binary
- **macOS:** Builds from source (requires Go 1.21+)

---

### 8.1 Script Installation (Recommended)

The easiest way to install **witr** is via the install script. Works on both Linux and macOS.

#### Quick install
```bash
curl -fsSL https://raw.githubusercontent.com/pranshuparmar/witr/main/install.sh | bash
```

#### Review before install
```bash
curl -fsSL https://raw.githubusercontent.com/pranshuparmar/witr/main/install.sh -o install.sh
cat install.sh
chmod +x install.sh
./install.sh
```

The script will:
- Detect your OS (Linux or macOS) and CPU architecture (`amd64` or `arm64`)
- **Linux:** Download the latest prebuilt binary
- **macOS:** Clone the repo and build from source (requires Go 1.21+)
- Install to `/usr/local/bin/witr`
- Install the man page to `/usr/local/share/man/man1/witr.1`

You may be prompted for your password to write to system directories.

### 8.2 Manual Installation

If you prefer manual installation, follow these simple steps for your architecture:

#### For amd64 (most PCs/servers):
```bash
# Download the binary
curl -fsSL https://github.com/pranshuparmar/witr/releases/latest/download/witr-linux-amd64 -o witr-linux-amd64

# Verify checksum (Optional, should print OK)
curl -fsSL https://github.com/pranshuparmar/witr/releases/latest/download/SHA256SUMS -o SHA256SUMS
grep witr-linux-amd64 SHA256SUMS | sha256sum -c -

# Rename and install
mv witr-linux-amd64 witr && chmod +x witr
sudo mv witr /usr/local/bin/witr

# Install the man page (Optional)
sudo curl -fsSL https://github.com/pranshuparmar/witr/releases/latest/download/witr.1 -o /usr/local/share/man/man1/witr.1
sudo mandb >/dev/null 2>&1 || true
```

#### For arm64 (Raspberry Pi, ARM servers):
```bash
# Download the binary
curl -fsSL https://github.com/pranshuparmar/witr/releases/latest/download/witr-linux-arm64 -o witr-linux-arm64

# Verify checksum (Optional, should print OK)
curl -fsSL https://github.com/pranshuparmar/witr/releases/latest/download/SHA256SUMS -o SHA256SUMS
grep witr-linux-arm64 SHA256SUMS | sha256sum -c -

# Rename and install
mv witr-linux-arm64 witr && chmod +x witr
sudo mv witr /usr/local/bin/witr

# Install the man page (Optional)
sudo curl -fsSL https://github.com/pranshuparmar/witr/releases/latest/download/witr.1 -o /usr/local/share/man/man1/witr.1
sudo mandb >/dev/null 2>&1 || true
```

**Explanation:**
- Download only the binary for your architecture and the SHA256SUMS file.
- Verify the checksum for your binary only (prints OK if valid).
- Rename to witr, make it executable, and move to your PATH.
- Install man page.

### 8.3 Verify installation:
```bash
witr --version
man witr
```

### 8.4 Uninstallation

To completely remove **witr**:
```bash
sudo rm -f /usr/local/bin/witr
sudo rm -f /usr/local/share/man/man1/witr.1
```

---

## 9. Platform Support

- **Linux** - Full support via `/proc` filesystem
- **macOS** - Full support via `lsof`, `ps`, `sysctl`

---

### 9.1 macOS Notes

On macOS, witr uses system commands (`lsof`, `ps`, `sysctl`) instead of `/proc`. Most features work identically with these differences:

| Feature | Linux | macOS |
|---------|-------|-------|
| Environment vars (`--env`) | ✓ | Not available |
| Container detection | ✓ | Skipped (Docker runs in VM) |
| Source detection | systemd | launchd |
| Health status | Full (zombie, high-cpu, high-mem) | Basic ("healthy" only) |

#### Building on macOS

```bash
go build -ldflags "-X main.version=dev -X main.commit=$(git rev-parse --short HEAD) -X 'main.buildDate=$(date +%Y-%m-%d)'" -o witr ./cmd/witr
```

---

### 9.2 Permissions Note

**Linux:** witr inspects `/proc` and may require elevated permissions to explain certain processes.

**macOS:** witr uses `lsof` which may require elevated permissions for processes owned by other users.

If you are not seeing the expected information (e.g., missing process ancestry, user, working directory or environment details), try running witr with sudo:

```bash
sudo witr [your arguments]
```

---

## 10. Success Criteria

witr is successful if:
- An engineer can answer "why is this running?" within seconds
- It reduces reliance on multiple tools
- Output is understandable under stress
- Users trust it during incidents

---

## 11. AI Assistance Disclaimer

This project was developed with assistance from AI/LLMs (including GitHub Copilot, ChatGPT, and related tools), supervised by a human who occasionally knew what he was doing.

---
