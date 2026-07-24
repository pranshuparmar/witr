// shell.js — a tiny fake shell around the witr engine.
//
// It is deliberately NOT a general shell: it simulates witr faithfully and
// offers just enough coreutils flavour (ls, cat, ps, ...) to make the box feel
// real to poke at. Everything routes through the world data.

import { Engine, EXIT } from './engine.js';
import { ESC } from './ansi.js';
import { tokenize, parse } from './parser.js';

// Fallback until app.js fetches the real value from internal/version/VERSION.
const WITR_VERSION_FALLBACK = 'v0.3.3';

const HELP = `Available commands in this playground:

  ${ESC.green}witr${ESC.reset} [name|flags]     the tool itself — try: ${ESC.cyan}witr node${ESC.reset}
  ${ESC.green}witr${ESC.reset} (no args)         launch the interactive TUI dashboard
  ${ESC.green}witr --help${ESC.reset}            full witr flag reference

  ${ESC.dim}ls${ESC.reset} [dir]               list a directory
  ${ESC.dim}cat${ESC.reset} <file>             print a file
  ${ESC.dim}ps${ESC.reset} / ${ESC.dim}top${ESC.reset}               list processes / by CPU
  ${ESC.dim}kill${ESC.reset} <pid>             stop a process (it really goes away here)
  ${ESC.dim}pwd / cd / whoami${ESC.reset}      the usual
  ${ESC.dim}uname${ESC.reset} [-a] / ${ESC.dim}hostname${ESC.reset}  system info
  ${ESC.dim}neofetch${ESC.reset}               about this machine
  ${ESC.dim}clear${ESC.reset}                  clear the screen
  ${ESC.dim}scenario${ESC.reset}               show / switch the machine you're on

Everything here runs against a ${ESC.dimYellow}simulated${ESC.reset} machine — nothing touches your computer.
Type ${ESC.cyan}witr node${ESC.reset} to start, or open the tutorial with the button on the left.`;

const WITR_HELP = `witr — Why Is This Running?

Explains where a running thing came from, how it started, and what chain of
systems is responsible for it existing right now.

Usage:
  witr [flags] [name...]

Flags:
  -c, --container strings  container(s) to look up (repeatable)
      --env                show environment variables for the process
  -x, --exact              use exact name matching (no substring search)
  -f, --file strings       file(s) held open by a process (repeatable)
  -h, --help               help for witr
  -i, --interactive        interactive mode (TUI)
      --json               show result as JSON
      --no-color           disable colorized output
  -p, --pid strings        pid(s) to look up (repeatable)
  -o, --port strings       port(s) to look up (repeatable)
  -s, --short              show only ancestry
  -t, --tree               show only ancestry as a tree
      --verbose            show extended process information
  -v, --version            version for witr
      --warnings           show only warnings

Positional arguments are treated as process or service names (substring match).`;

export class Shell {
  constructor(world) {
    this.version = WITR_VERSION_FALLBACK;
    this.setWorld(world);
    this.cwd = `/home/${world.promptUser}`;
  }

  setVersion(v) { if (v) this.version = v; }

  setWorld(world) {
    this.world = world;
    this.engine = new Engine(world);
    this.cwd = `/home/${world.promptUser}`;
  }

  prompt() {
    const dir = this.cwd === `/home/${this.world.promptUser}` ? '~' : this.cwd;
    return { user: this.world.promptUser, host: this.world.promptHost, dir };
  }

  // Returns { output, exit, action }.
  exec(line) {
    const trimmed = line.trim();
    if (trimmed === '') return { output: '', exit: 0 };
    const tokens = tokenize(trimmed);
    const cmd = tokens[0];
    const args = tokens.slice(1);

    switch (cmd) {
      case 'witr': return this.witr(args);
      case 'help': case '?': return { output: HELP + '\n', exit: 0 };
      case 'clear': return { output: '', exit: 0, action: 'clear' };
      case 'ls': return this.ls(args);
      case 'cat': return this.cat(args);
      case 'pwd': return { output: this.cwd + '\n', exit: 0 };
      case 'cd': return this.cd(args);
      case 'whoami': return { output: this.world.promptUser + '\n', exit: 0 };
      case 'hostname': return { output: this.world.hostname + '\n', exit: 0 };
      case 'uname': return this.uname(args);
      case 'ps': return this.ps(args);
      case 'top': return this.top();
      case 'htop': return { output: `htop: command not found. Try ${ESC.cyan}top${ESC.reset}.\n`, exit: 127 };
      case 'neofetch': case 'witr-info': return { output: this.neofetch(), exit: 0 };
      case 'echo': return { output: args.join(' ') + '\n', exit: 0 };
      case 'scenario': case 'scenarios': return { output: '', exit: 0, action: 'scenario' };
      case 'man': return this.man(args);
      case 'exit': case 'logout': return { output: 'There is no escape from a simulation.\n', exit: 0 };
      case 'sudo': return { output: `${this.world.promptUser} is not in the sudoers file. This incident will (not) be reported.\n`, exit: 1 };
      case 'kill': case 'pkill': return this.kill(cmd, args);
      default:
        return { output: `${cmd}: command not found. Type ${ESC.cyan}help${ESC.reset}.\n`, exit: 127 };
    }
  }

  // kill/pkill mutate the world: the target process (and its subtree) goes away,
  // which the engine, map, TUI, and incident tracker all reflect.
  kill(cmd, args) {
    const pids = [];
    let bad = '';
    if (cmd === 'pkill') {
      const name = args.find((a) => !a.startsWith('-'));
      if (!name) return { output: 'usage: pkill <name>\n', exit: 1 };
      for (const p of this.world.processes) {
        if ((p.command || '').toLowerCase().includes(name.toLowerCase())) pids.push(p.pid);
      }
      if (pids.length === 0) return { output: `pkill: no process matched "${name}"\n`, exit: 1 };
    } else {
      for (const a of args) {
        if (a.startsWith('-')) continue; // signal flag, ignored by the sim
        const n = parseInt(a, 10);
        if (Number.isNaN(n)) { bad += `kill: illegal pid: ${a}\n`; continue; }
        if (!this.world.processes.some((p) => p.pid === n)) { bad += `kill: (${n}): No such process\n`; continue; }
        pids.push(n);
      }
      if (pids.length === 0) return { output: bad || 'usage: kill [-signal] <pid>\n', exit: 1 };
    }

    const killed = this.killProcesses(pids);
    let out = bad;
    for (const p of killed) {
      out += `${ESC.dim}[sim] terminated ${p.command} (pid ${p.pid})${ESC.reset}\n`;
    }
    return { output: out, exit: bad ? 1 : 0, action: 'killed', killed: killed.map((p) => p.pid) };
  }

  // Remove the given pids and their descendants; returns the removed processes.
  killProcesses(pids) {
    const toRemove = new Set();
    const collect = (pid) => {
      if (toRemove.has(pid)) return;
      toRemove.add(pid);
      for (const c of this.world.processes) if (c.ppid === pid) collect(c.pid);
    };
    for (const pid of pids) collect(pid);
    const removed = this.world.processes.filter((p) => toRemove.has(p.pid));
    this.world.processes = this.world.processes.filter((p) => !toRemove.has(p.pid));
    // A killed process also drops any locks it held.
    if (this.world.locks) this.world.locks = this.world.locks.filter((l) => !toRemove.has(l.pid));
    this.engine.reindex();
    return removed;
  }

  witr(args) {
    const { targets, flags, errors } = parse(args);
    if (flags.help) return { output: WITR_HELP + '\n', exit: 0 };
    if (flags.version) return { output: `witr ${this.version}\n`, exit: 0 };
    if (errors.length > 0) {
      return { output: errors.map((e) => `Error: ${e}`).join('\n') + '\n', exit: EXIT.INVALID_INPUT };
    }
    // No targets, or explicit -i → launch the TUI.
    if (targets.length === 0 || flags.interactive) {
      return { output: '', exit: 0, action: 'tui' };
    }
    const { text, exit } = this.engine.run({ targets, flags });
    return { output: text, exit };
  }

  // ---- filesystem flavour ----------------------------------------------

  resolvePath(p) {
    if (!p) return this.cwd;
    if (p === '~') return `/home/${this.world.promptUser}`;
    if (p.startsWith('~/')) return `/home/${this.world.promptUser}/` + p.slice(2);
    if (p.startsWith('/')) return normalizePath(p);
    return normalizePath(this.cwd + '/' + p);
  }

  ls(args) {
    const target = args.find((a) => !a.startsWith('-'));
    const path = this.resolvePath(target);
    const fs = this.world.fs || {};
    const entries = fs[path];
    if (!entries) {
      // A file rather than a dir?
      if ((this.world.files || {})[path]) return { output: path + '\n', exit: 0 };
      return { output: `ls: cannot access '${target || path}': No such file or directory\n`, exit: 2 };
    }
    const colored = entries.map((e) => (e.endsWith('/') ? `${ESC.blue}${e}${ESC.reset}` : e));
    return { output: colored.join('  ') + '\n', exit: 0 };
  }

  cat(args) {
    if (args.length === 0) return { output: 'cat: missing file operand\n', exit: 1 };
    const path = this.resolvePath(args[0]);
    const content = (this.world.files || {})[path];
    if (content == null) {
      if ((this.world.fs || {})[path]) return { output: `cat: ${args[0]}: Is a directory\n`, exit: 1 };
      return { output: `cat: ${args[0]}: No such file or directory\n`, exit: 1 };
    }
    return { output: content.endsWith('\n') ? content : content + '\n', exit: 0 };
  }

  cd(args) {
    if (args.length === 0 || args[0] === '~') { this.cwd = `/home/${this.world.promptUser}`; return { output: '', exit: 0 }; }
    const path = this.resolvePath(args[0]);
    const fs = this.world.fs || {};
    if (fs[path] || fs[path + '/']) { this.cwd = path; return { output: '', exit: 0 }; }
    return { output: `cd: ${args[0]}: No such file or directory\n`, exit: 1 };
  }

  uname(args) {
    if (args.includes('-a')) {
      const w = this.world;
      return { output: `Linux ${w.hostname} ${w.kernel} #1 SMP ${w.arch} GNU/Linux\n`, exit: 0 };
    }
    return { output: 'Linux\n', exit: 0 };
  }

  // A static top-style snapshot: processes sorted by CPU then memory. This is
  // how you *find* the pid that's pinning a core (the devbox CPU task).
  top() {
    const w = this.world;
    const total = w.memTotalBytes || 8 * 1024 * 1024 * 1024;
    const procs = [...w.processes].sort((a, b) => {
      const c = (b.cpuPercent || 0) - (a.cpuPercent || 0);
      if (c !== 0) return c;
      return ((b.memory && b.memory.rss) || 0) - ((a.memory && a.memory.rss) || 0);
    });
    const totalCpu = procs.reduce((s, p) => s + (p.cpuPercent || 0), 0);
    let o = `${ESC.dim}top - snapshot  up 21 days,  ${w.processes.length} tasks,  load average: ${(totalCpu / 100).toFixed(2)}${ESC.reset}\n`;
    o += `${ESC.dim}%Cpu(s): ${totalCpu.toFixed(1)} us   MiB Mem: ${(total / 1048576).toFixed(0)} total${ESC.reset}\n\n`;
    o += `${ESC.dim}  PID USER       %CPU  %MEM COMMAND${ESC.reset}\n`;
    for (const p of procs.slice(0, 12)) {
      const cpu = (p.cpuPercent || 0).toFixed(1).padStart(5);
      const mem = ((((p.memory && p.memory.rss) || 0) / total) * 100).toFixed(1).padStart(5);
      const hot = (p.cpuPercent || 0) > 50;
      const line = `${String(p.pid).padStart(5)} ${(p.user || '').padEnd(10).slice(0, 10)} ${cpu} ${mem} ${p.command}`;
      o += hot ? `${ESC.red}${line}${ESC.reset}\n` : `${line}\n`;
    }
    o += `${ESC.dim}(static snapshot — pass a pid to witr to find out *why* it's running)${ESC.reset}\n`;
    return { output: o, exit: 0 };
  }

  ps() {
    let o = `${ESC.dim}  PID USER      COMMAND${ESC.reset}\n`;
    const procs = [...this.world.processes].sort((a, b) => a.pid - b.pid);
    for (const p of procs) {
      const pid = String(p.pid).padStart(5);
      const user = (p.user || '').padEnd(9).slice(0, 9);
      o += `${pid} ${user} ${p.cmdline || p.command}\n`;
    }
    return { output: o, exit: 0 };
  }

  neofetch() {
    const w = this.world;
    const art = [
      `${ESC.cyan}   __      _ _        ${ESC.reset}`,
      `${ESC.cyan}  / /_ __ (_) |_ _ __ ${ESC.reset}`,
      `${ESC.cyan} / / '_ \\| | __| '__|${ESC.reset}`,
      `${ESC.cyan}/ /| | | | | |_| |   ${ESC.reset}`,
      `${ESC.cyan}\\_/|_| |_|_|\\__|_|   ${ESC.reset}`,
    ];
    const info = [
      `${ESC.green}${w.promptUser}@${w.hostname}${ESC.reset}`,
      `${ESC.dim}─────────────────${ESC.reset}`,
      `${ESC.blue}OS${ESC.reset}:     ${w.distro} ${w.arch}`,
      `${ESC.blue}Kernel${ESC.reset}: ${w.kernel}`,
      `${ESC.blue}Procs${ESC.reset}:  ${w.processes.length}`,
      `${ESC.blue}Shell${ESC.reset}:  witr-playground`,
      `${ESC.dimYellow}note${ESC.reset}:   this machine is simulated`,
    ];
    let o = '';
    const rows = Math.max(art.length, info.length);
    for (let i = 0; i < rows; i++) {
      const left = art[i] || '                     ';
      o += `${left}   ${info[i] || ''}\n`;
    }
    return o;
  }

  man(args) {
    if (args[0] === 'witr') return { output: WITR_HELP + '\n', exit: 0 };
    return { output: `No manual entry for ${args[0] || ''}\n`, exit: 1 };
  }
}

function normalizePath(p) {
  const parts = p.split('/');
  const out = [];
  for (const part of parts) {
    if (part === '' || part === '.') continue;
    if (part === '..') out.pop();
    else out.push(part);
  }
  return '/' + out.join('/');
}
