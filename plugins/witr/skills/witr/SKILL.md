---
name: witr
description: Explains why a process is running and what started it, using the witr CLI. Use when a port is already in use (EADDRINUSE), a dev server or other process was left running, an unknown or stuck process needs explaining, or you need to know which service, container, shell, supervisor or scheduled job started a process, holds a port or holds a file lock. Use it instead of chaining lsof, ps, ss, netstat and docker ps.
license: Apache-2.0
---

# witr: why is this running?

`witr` traces a process, port, container or open file back to what started it and keeps it running (a systemd unit, launchd job, container, cron job, shell, PM2 or supervisor, a Windows service, ...), and flags anything suspicious. It only reads. It never stops, kills or changes anything.

Reach for it instead of chaining `lsof`, `ps`, `ss`, `netstat` and `docker ps`.

## Before you start

Check that it's installed with `witr --version`. If it isn't, tell the user and suggest one of these; don't install anything without asking:

- macOS or Linux: `brew install witr`, or `curl -fsSL https://raw.githubusercontent.com/pranshuparmar/witr/main/install.sh | bash`
- Windows: `winget install -e --id PranshuParmar.witr`
- With Go: `go install github.com/pranshuparmar/witr/cmd/witr@latest`

Always give a target and `--json`. Without a target, `--json` exits with code 4 and asks for one; a bare `witr` opens an interactive dashboard, or exits with code 4 when there's no terminal.

## Pick the command

| Situation | Command |
|-----------|---------|
| Port already in use (`EADDRINUSE`) | `witr --port 3000 --json` |
| A process by name | `witr node --json` (matches substrings; add `--exact` for the exact name) |
| A known PID | `witr --pid 1234 --json` |
| A container | `witr --container web --json` (name, image, Compose service or ID prefix) |
| What holds a file or lock | `witr --file /var/lib/dpkg/lock --json` |
| Just the chain of parents | `witr --pid 1234 --short` (one line) or `--tree` |
| Several things at once | `witr --port 3000 --port 5432 --json` (a JSON list, one entry per target, in order) |
| A process's environment | `witr --pid 1234 --env --json` |
| More detail | add `--verbose` (memory, I/O, open files, socket state) |

## Read the result

Check the exit code first:

| Exit | Meaning | What to do |
|------|---------|------------|
| 0 | Found, no warnings | Report it |
| 1 | Found, with warnings. Not a failure | Report it and mention the warnings that matter |
| 2 | Not found | Nothing on this system runs or listens there; say so. For a port, `Error` may say something outside this system (another WSL distro, a VM) holds it |
| 3 | Permission denied | It belongs to another user; suggest re-running with `sudo`, and ask first |
| 4 | Ambiguous or bad input | `Matches` lists the candidates: re-run with `--pid` for the right one, or fix the input. Servers with same-named workers (nginx, postgres) list every worker; the one you want is usually the parent |
| 5 | Internal error | Report the `Error` |
| 6 | Found, but what started it can't be traced (Linux, macOS, FreeBSD) | The process that started it has exited; say so |

With several targets, the most severe result wins.

Then the JSON (the full contract is in the README, section "7.3 JSON Output"):

- **Found:** `Process` (`PID`, `Command`, `Cmdline`, `User`, `StartedAt`, `WorkingDir`, `Sockets`, ...), `Ancestry` (the chain from the top down to the process), `Source` (`Type`: `systemd`, `launchd`, `bsdrc`, `container`, `shell`, `ssh`, `cron`, `supervisor`, `windows_service`, `init` or `unknown`; `Name`; `Description`), `Warnings`, and `Container` when it runs in one. Environment variables aren't included: only `--env --json` prints them, and they often hold secrets.
- **Failed lookup:** `{Target, Error}`, plus `Matches` when the target was ambiguous.
- **`--short --json`:** a list of `{PID, Command}`. Where the process that started one has exited, that entry also has `PPID` and `ParentExited: true`.
- **A container whose processes aren't visible** (Docker Desktop, a VM): `{ContainerName, Image, Ports, ..., Note}`.

Match on fields and exit codes, not on wording: the text of `Error`, `Note`, `Description` and warnings can change between releases.

## Explain it to the user

Lead with the answer: what it is, and what started it. For example: "Port 3000 is held by `node` (pid 4821), started by PM2 from `~/work/api`." Mention warnings that matter, such as a public bind, running as root, a deleted binary or a restart loop.

## Freeing a port or stopping something

witr only explains. Don't kill or stop anything on your own: tell the user what holds the port and what manages it, and ask before acting.

When they agree, stop it through whatever manages it, or it will come straight back. Use `Source.Type`, `Source.Name` and `Container`:

| Managed by | Stop with |
|------------|-----------|
| systemd | `systemctl stop <unit>` (`--user` for a user unit) |
| launchd | `launchctl bootout gui/$(id -u)/<label>` (`sudo launchctl bootout system/<label>` for a daemon) |
| FreeBSD rc (`bsdrc`) | `service <name> stop` |
| Docker / Podman / nerdctl | `docker stop <name>` / `podman stop <name>` / `nerdctl stop <name>` |
| Kubernetes (`Source.Name` is `kubernetes`) | Deleting the pod only gets it rescheduled: scale down or remove what owns it, e.g. `kubectl scale deployment <name> --replicas=0` |
| Incus / LXD / LXC | `incus stop <name>` / `lxc stop <name>` / `lxc-stop -n <name>` |
| PM2 | `pm2 stop <name>` |
| supervisor | `supervisorctl stop <name>` |
| cron | `kill <pid>` stops this run only; the job runs again on schedule until its crontab entry is removed |
| Windows service | `Stop-Service <name>` |
| A shell or nothing traceable | `kill <pid>` (`Stop-Process -Id <pid>` on Windows) |

Check `Ancestry` as well: when PM2 or supervisord sits between a systemd unit and the process (for example `systemd → PM2 → node`, with `Source` the `pm2-<user>.service` unit), stop the one app through PM2 or supervisor. Stopping the unit stops every app it runs.
