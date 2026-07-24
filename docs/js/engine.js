// engine.js — a faithful in-browser reimplementation of witr's output layer.
//
// Given a "world" (a fake machine described in worlds/*.json) and a parsed
// command, it produces the exact bytes the real `witr` binary would print,
// including ANSI colour escapes. The port mirrors, function for function:
//   internal/output/standard.go, short.go, tree.go, children.go, docker.go,
//   json.go, envonly.go, started.go   and   internal/app/app.go (routing).
//
// Fidelity is verified by fixtures/gen (Go, using witr's real output package)
// against scripts/check-fixtures.mjs.

import { ESC } from './ansi.js';

const MAX_DISPLAY_ITEMS = 10;

// Exit codes — internal/app/app.go.
export const EXIT = {
  OK: 0,
  WARNINGS: 1,
  NOT_FOUND: 2,
  PERMISSION: 3,
  INVALID_INPUT: 4,
  INTERNAL: 5,
};

export class Engine {
  constructor(world) {
    this.world = world;
    this.procByPid = new Map();
    this.reindex();
    // Clock is injectable so the fixture generator and the browser agree.
    this._now = () => Date.now();
  }

  // Rebuild the pid index after the world mutates (e.g. a process is killed).
  reindex() {
    this.procByPid = new Map();
    for (const p of this.world.processes) this.procByPid.set(p.pid, p);
  }

  setNow(fn) { this._now = fn; }
  now() { return this._now(); }

  // ---- resolution -------------------------------------------------------

  ancestryOf(proc) {
    // Returns [pid1 ... target], matching model.Result.Ancestry ordering
    // (the target is the last element).
    const chain = [];
    const seen = new Set();
    let cur = proc;
    while (cur && !seen.has(cur.pid)) {
      chain.push(cur);
      seen.add(cur.pid);
      if (cur.ppid === 0) break;
      cur = this.procByPid.get(cur.ppid);
    }
    chain.reverse();
    return chain;
  }

  childrenOf(pid) {
    return this.world.processes
      .filter((p) => p.ppid === pid && p.pid !== pid)
      .sort((a, b) => a.pid - b.pid);
  }

  resolveName(name, exact) {
    const lower = name.toLowerCase();
    const out = [];
    for (const p of this.world.processes) {
      const comm = (p.command || '').toLowerCase();
      const cmd = (p.cmdline || '').toLowerCase();
      let match;
      if (exact) {
        match = comm === lower || matchesExactToken(cmd, lower);
      } else {
        match = comm.includes(lower) || cmd.includes(lower);
      }
      if (match) out.push(p.pid);
    }
    out.sort((a, b) => a - b);
    return out;
  }

  resolvePort(port) {
    // Prefer a LISTEN socket, otherwise any socket on the port.
    let fallback = null;
    for (const p of this.world.processes) {
      for (const s of p.sockets || []) {
        if (s.port === port) {
          if (s.state === 'LISTEN') return p.pid;
          if (fallback === null) fallback = p.pid;
        }
      }
    }
    return fallback;
  }

  resolveFile(path) {
    const lock = (this.world.locks || []).find((l) => l.path === path);
    if (lock) return lock.pid;
    // Also honour a process's declared lockedFiles ("<path> (MODE)").
    for (const p of this.world.processes) {
      for (const lf of p.lockedFiles || []) {
        if (lf.replace(/\s*\([^)]*\)\s*$/, '') === path) return p.pid;
      }
    }
    return null;
  }

  resolveContainerByPort(port) {
    // Mirrors procpkg.ResolveContainerByPort: a published host port with no
    // visible owning process falls back to the container that exposes it.
    for (const c of this.world.containers || []) {
      const re = /:(\d+)->/g;
      let m;
      while ((m = re.exec(c.ports || '')) !== null) {
        if (parseInt(m[1], 10) === port) return c;
      }
    }
    return null;
  }

  resolveContainer(query, exact) {
    const lower = query.toLowerCase();
    return (this.world.containers || []).filter((c) => {
      const fields = [c.name, c.image, c.command, c.composeProject, c.composeService];
      if (exact) return (c.name || '').toLowerCase() === lower;
      return fields.some((f) => (f || '').toLowerCase().includes(lower));
    });
  }

  // ---- result assembly --------------------------------------------------

  buildResult(pid) {
    const proc = this.procByPid.get(pid);
    if (!proc) return null;
    const ancestry = this.ancestryOf(proc);
    const children = this.childrenOf(pid);
    const source = this.resolveSource(proc, ancestry);
    // Restart count comes only from a systemd unit's NRestarts, mirroring
    // pipeline.AnalyzePID — not an arbitrary per-process field.
    let restartCount = 0;
    if (source.type === 'systemd' && source.details && source.details.NRestarts) {
      restartCount = parseInt(source.details.NRestarts, 10) || 0;
    }
    return {
      target: proc,
      ancestry,
      children,
      process: proc,
      source,
      restartCount,
      warnings: proc.warnings || [],
    };
  }

  resolveSource(proc, ancestry) {
    if (proc.source) return proc.source;
    // Derive from the nearest ancestor that declares a source.
    for (let i = ancestry.length - 2; i >= 0; i--) {
      if (ancestry[i].source) return ancestry[i].source;
    }
    return { type: 'unknown', name: '' };
  }

  // ---- time (started.go) ------------------------------------------------

  startedAtMs(proc) {
    if (proc.startedAgo == null) return null;
    return this.now() - proc.startedAgo * 1000;
  }

  // ---- top-level routing (app.go) --------------------------------------

  run(parsed) {
    const { targets, flags } = parsed;
    const multiMode = targets.length > 1;
    let out = '';
    let highest = EXIT.OK;
    const jsonResults = [];

    targets.forEach((t, i) => {
      if (multiMode && !flags.json) {
        out += printDivider(t, flags.color, i > 0);
      }
      const r = this.processTarget(t, flags, multiMode, jsonResults);
      out += r.text;
      if (r.exit > highest) highest = r.exit;
    });

    if (flags.json && multiMode) {
      out = '[' + jsonResults.join(',\n') + ']\n';
    } else if (flags.json && jsonResults.length === 1 && out === '') {
      out = jsonResults[0] + '\n';
    }
    return { text: out, exit: highest };
  }

  processTarget(t, flags, multiMode, jsonResults) {
    if (flags.env) return this.processEnvTarget(t, flags, multiMode, jsonResults);
    if (t.type === 'container') return this.processContainerTarget(t, flags, multiMode, jsonResults);

    let pids;
    if (t.type === 'pid') {
      const pid = parseInt(t.value, 10);
      pids = this.procByPid.has(pid) ? [pid] : [];
    } else if (t.type === 'port') {
      const pid = this.resolvePort(parseInt(t.value, 10));
      pids = pid ? [pid] : [];
      if (!pid) {
        const c = this.resolveContainerByPort(parseInt(t.value, 10));
        if (c) return this.renderSingleContainer(c, `port ${t.value}`, flags, multiMode, jsonResults);
      }
    } else if (t.type === 'file') {
      const pid = this.resolveFile(t.value);
      pids = pid ? [pid] : [];
    } else {
      pids = this.resolveName(t.value, flags.exact);
    }

    if (pids.length === 0) {
      return { text: this.notFound(t, flags, jsonResults), exit: EXIT.NOT_FOUND };
    }
    if (pids.length > 1) {
      if (multiMode && flags.json) {
        jsonResults.push(jsonErrorEntry(t, `multiple processes matched (${pids.length} results)`));
        return { text: '', exit: EXIT.INVALID_INPUT };
      }
      const hint = flags.env ? 'witr --pid <pid> --env' : 'witr --pid <pid>';
      return { text: this.printMultiMatch(pids, flags.color, hint), exit: EXIT.INVALID_INPUT };
    }

    const r = this.buildResult(pids[0]);

    if (flags.json) {
      let js;
      if (flags.short) js = toTreeJSON(r);
      else if (flags.tree) js = toTreeJSON(r);
      else if (flags.warnings) js = toWarningsJSON(r);
      else js = toJSON(r, this);
      if (multiMode) { jsonResults.push(js); return { text: '', exit: exitFor(r) }; }
      return { text: js + '\n', exit: exitFor(r) };
    }

    let text;
    if (flags.short) text = renderShort(r, flags.color);
    else if (flags.tree) text = renderTree(r, flags.color);
    else if (flags.warnings) text = renderWarnings(r, flags.color);
    else text = this.renderStandard(r, flags.color, flags.verbose, t);

    return { text, exit: exitFor(r) };
  }

  processEnvTarget(t, flags, multiMode, jsonResults) {
    const pids = t.type === 'pid'
      ? (this.procByPid.has(parseInt(t.value, 10)) ? [parseInt(t.value, 10)] : [])
      : this.resolveName(t.value, flags.exact);
    if (pids.length === 0) return { text: 'No matching process found.\n', exit: EXIT.NOT_FOUND };
    if (pids.length > 1) {
      return { text: this.printMultiMatch(pids, flags.color, 'witr --pid <pid> --env'), exit: EXIT.INVALID_INPUT };
    }
    const r = this.buildResult(pids[0]);
    if (flags.json) {
      const js = toEnvJSON(r);
      if (multiMode) { jsonResults.push(js); return { text: '', exit: EXIT.OK }; }
      return { text: js + '\n', exit: EXIT.OK };
    }
    return { text: renderEnvOnly(r, flags.color), exit: EXIT.OK };
  }

  processContainerTarget(t, flags, multiMode, jsonResults) {
    const matches = this.resolveContainer(t.value, flags.exact);
    if (matches.length === 0) {
      return { text: this.notFound(t, flags, jsonResults), exit: EXIT.NOT_FOUND };
    }
    if (matches.length > 1) {
      return { text: printContainerMultiMatch(matches, flags.color), exit: EXIT.INVALID_INPUT };
    }
    return this.renderSingleContainer(matches[0], `container ${t.value}`, flags, multiMode, jsonResults);
  }

  renderSingleContainer(match, label, flags, multiMode, jsonResults) {
    if (flags.json) {
      const js = containerToJSON(label, match);
      if (multiMode) { jsonResults.push(js); return { text: '', exit: EXIT.OK }; }
      return { text: js + '\n', exit: EXIT.OK };
    }
    if (flags.short) return { text: renderContainerShort(match, flags.color), exit: EXIT.OK };
    if (flags.tree) return { text: renderContainerTree(match, flags.color), exit: EXIT.OK };
    if (flags.warnings) return { text: renderContainerWarnings(match, flags.color), exit: EXIT.OK };
    return { text: renderContainerFallback(label, match, flags.color, flags.verbose, this), exit: EXIT.OK };
  }

  notFound(t, flags, jsonResults) {
    if (flags.json) {
      jsonResults.push(jsonErrorEntry(t, 'no matching process or service found'));
      return '';
    }
    const q = t.value;
    return `Error: no running process or service named "${q}"\n\n` +
      `No matching process or service found. Please check your query or try a different name/port/PID.\n` +
      `For usage and options, run: witr --help\n`;
  }

  printMultiMatch(pids, color, hint) {
    let s = 'Multiple matching processes found:\n\n';
    pids.forEach((pid, i) => {
      const p = this.procByPid.get(pid);
      const command = p ? p.command : 'unknown';
      const cmdline = p ? p.cmdline : '';
      if (color) {
        s += `[${i + 1}] ${ESC.green}${command}${ESC.reset} (${ESC.dim}pid ${pid}${ESC.reset})\n    ${cmdline}\n`;
      } else {
        s += `[${i + 1}] ${command} (pid ${pid})\n    ${cmdline}\n`;
      }
    });
    s += '\nRe-run with:\n';
    s += `  ${hint}\n`;
    return s;
  }

  // ---- standard render (standard.go) -----------------------------------

  renderStandard(r, color, verbose, target) {
    if (r.ancestry.length === 0) return 'No process information available.\n';
    let o = '';
    const proc = r.ancestry[r.ancestry.length - 1];

    const targetName = proc.command;
    o += color
      ? `${ESC.blue}Target${ESC.reset}      : ${targetName}\n\n`
      : `Target      : ${targetName}\n\n`;

    // Process line + health/forked tags.
    o += color
      ? `${ESC.blue}Process${ESC.reset}     : ${ESC.green}${proc.command}${ESC.reset} (${ESC.dim}pid ${proc.pid}${ESC.reset})`
      : `Process     : ${proc.command} (pid ${proc.pid})`;
    if (proc.health && proc.health !== 'healthy') {
      o += color ? ` ${ESC.red}[${proc.health}]${ESC.reset}` : ` [${proc.health}]`;
    }
    if (proc.forked === 'forked') {
      o += color ? ` ${ESC.dimYellow}{forked}${ESC.reset}` : ` {forked}`;
    }
    o += '\n';

    if (proc.user && proc.user !== 'unknown') {
      o += color ? `${ESC.blue}User${ESC.reset}        : ${proc.user}\n` : `User        : ${proc.user}\n`;
    }
    if (proc.container) {
      o += color ? `${ESC.blue}Container${ESC.reset}   : ${proc.container}\n` : `Container   : ${proc.container}\n`;
    }
    if (proc.service) {
      o += color ? `${ESC.blue}Service${ESC.reset}     : ${proc.service}\n` : `Service     : ${proc.service}\n`;
    }

    const cmd = proc.cmdline || proc.command;
    o += color ? `${ESC.blue}Command${ESC.reset}     : ${cmd}\n` : `Command     : ${cmd}\n`;

    const [rel, abs] = formatStartedAt(this.startedAtMs(proc), this.now());
    const startVal = abs ? `${rel} (${abs})` : rel;
    o += color ? `${ESC.magenta}Started${ESC.reset}     : ${startVal}\n` : `Started     : ${startVal}\n`;

    if (r.restartCount > 0) {
      o += color ? `${ESC.magenta}Restarts${ESC.reset}    : ${r.restartCount}\n` : `Restarts    : ${r.restartCount}\n`;
    }
    if (r.source.details && r.source.details.schedule) {
      const sch = r.source.details.schedule;
      o += color ? `${ESC.magenta}Schedule${ESC.reset}    : ${sch}\n` : `Schedule    : ${sch}\n`;
    }

    // Why It Exists chain.
    o += color ? `\n${ESC.magenta}Why It Exists${ESC.reset} :\n  ` : `\nWhy It Exists :\n  `;
    r.ancestry.forEach((p, i) => {
      const name = chainName(p);
      if (color) {
        const nameColor = i === r.ancestry.length - 1 ? ESC.green : '';
        o += `${nameColor}${name}${ESC.reset} (${ESC.dim}pid ${p.pid}${ESC.reset})`;
        if (i < r.ancestry.length - 1) o += ` ${ESC.magenta}→${ESC.reset} `;
      } else {
        o += `${name} (pid ${p.pid})`;
        if (i < r.ancestry.length - 1) o += ` → `;
      }
    });
    o += '\n\n';

    // Source.
    const label = r.source.type;
    const name = r.source.name;
    if (name && name !== label) {
      o += color ? `${ESC.cyan}Source${ESC.reset}      : ${name} (${label})\n` : `Source      : ${name} (${label})\n`;
    } else {
      o += color ? `${ESC.cyan}Source${ESC.reset}      : ${label}\n` : `Source      : ${label}\n`;
    }
    if (r.source.description) {
      o += color ? `${ESC.cyan}Description${ESC.reset} : ${r.source.description}\n` : `Description : ${r.source.description}\n`;
    }
    if (r.source.unitFile) {
      let ul = 'Unit File';
      if (label === 'launchd') ul = 'Plist File';
      else if (label === 'windows_service') ul = 'Registry Key';
      else if (label === 'bsdrc') ul = 'Rc Script';
      const pad = ul.length < 12 ? ' '.repeat(12 - ul.length) : ' ';
      o += color ? `${ESC.cyan}${ul}${ESC.reset}${pad}: ${r.source.unitFile}\n` : `${ul}${pad}: ${r.source.unitFile}\n`;
    }

    // Context group.
    if (proc.workingDir && proc.workingDir !== 'unknown') {
      o += color ? `\n${ESC.cyan}Working Dir${ESC.reset} : ${proc.workingDir}\n` : `\nWorking Dir : ${proc.workingDir}\n`;
    }
    if (proc.gitRepo) {
      if (proc.gitBranch) {
        o += color ? `${ESC.cyan}Git Repo${ESC.reset}    : ${proc.gitRepo} (${proc.gitBranch})\n` : `Git Repo    : ${proc.gitRepo} (${proc.gitBranch})\n`;
      } else {
        o += color ? `${ESC.cyan}Git Repo${ESC.reset}    : ${proc.gitRepo}\n` : `Git Repo    : ${proc.gitRepo}\n`;
      }
    }

    // Sockets.
    const visible = visibleSockets(proc.sockets || []);
    sortSockets(visible);
    visible.forEach((s, i) => {
      if (i >= MAX_DISPLAY_ITEMS) return;
      const line = formatSocket(s);
      if (i === 0) o += color ? `${ESC.green}Sockets${ESC.reset}     : ${line}\n` : `Sockets     : ${line}\n`;
      else o += `              ${line}\n`;
    });
    if (visible.length > MAX_DISPLAY_ITEMS) {
      o += `              ... and ${visible.length - MAX_DISPLAY_ITEMS} more\n`;
    }

    // Warnings.
    if (r.warnings.length > 0) {
      o += color ? `\n${ESC.red}Warnings${ESC.reset}    :\n` : `\nWarnings    :\n`;
      for (const w of r.warnings) o += `  • ${w}\n`;
    }

    if (verbose) o += this.renderVerbose(r, proc, color, target);
    return o;
  }

  renderVerbose(r, proc, color, target) {
    let o = '\n';

    if (proc.memory && proc.memory.vms > 0) {
      o += color ? `\n${ESC.green}Memory${ESC.reset}:\n` : `\nMemory:\n`;
      o += `  Virtual  : ${formatBytes(proc.memory.vms)}\n`;
      o += `  Resident : ${formatBytes(proc.memory.rss)}\n`;
      if (proc.memory.shared > 0) o += `  Shared   : ${formatBytes(proc.memory.shared)}\n`;
    }

    if (proc.io && (proc.io.readBytes > 0 || proc.io.writeBytes > 0)) {
      o += color ? `\n${ESC.green}I/O Statistics${ESC.reset}:\n` : `\nI/O Statistics:\n`;
      if (proc.io.readBytes > 0) o += `  Read  : ${formatBytes(proc.io.readBytes)} (${proc.io.readOps || 0} ops)\n`;
      if (proc.io.writeBytes > 0) o += `  Write : ${formatBytes(proc.io.writeBytes)} (${proc.io.writeOps || 0} ops)\n`;
    }

    // File context: open files + locks. "Open Files" comes from the file
    // context (openFiles/fileLimit), distinct from the fd list below — matching
    // witr's model.FileContext vs model.Process.FDCount.
    const locked = proc.lockedFiles || [];
    if (proc.openFiles > 0 && !proc.fileLimit) {
      o += color ? `\n${ESC.green}Open Files${ESC.reset}  : ${proc.openFiles} of unlimited\n` : `\nOpen Files  : ${proc.openFiles} of unlimited\n`;
    } else if (proc.openFiles > 0 && proc.fileLimit > 0) {
      const pct = (proc.openFiles / proc.fileLimit) * 100;
      if (color && pct > 80) {
        o += `\n${ESC.red}Open Files${ESC.reset}  : ${ESC.dimYellow}${proc.openFiles} of ${proc.fileLimit} (${pct.toFixed(0)}%)${ESC.reset}\n`;
      } else {
        o += color
          ? `\n${ESC.green}Open Files${ESC.reset}  : ${proc.openFiles} of ${proc.fileLimit} (${pct.toFixed(0)}%)\n`
          : `\nOpen Files  : ${proc.openFiles} of ${proc.fileLimit} (${pct.toFixed(0)}%)\n`;
      }
    }
    if (locked.length > 0) {
      o += color ? `${ESC.green}Locks${ESC.reset}       : ${locked[0]}\n` : `Locks       : ${locked[0]}\n`;
      for (let i = 1; i < locked.length; i++) {
        if (i >= MAX_DISPLAY_ITEMS) { o += `              ... and ${locked.length - i} more\n`; break; }
        o += `              ${locked[i]}\n`;
      }
    }

    if (proc.fdCount > 0) {
      o += color
        ? `\n${ESC.green}File Descriptors${ESC.reset}: ${proc.fdLimit ? proc.fdCount + '/' + proc.fdLimit : proc.fdCount + '/unlimited'}\n`
        : `\nFile Descriptors: ${proc.fdLimit ? proc.fdCount + '/' + proc.fdLimit : proc.fdCount + '/unlimited'}\n`;
    }

    // Socket state — only for port queries.
    if (target && target.type === 'port') {
      const info = this.socketInfoFor(proc, parseInt(target.value, 10));
      if (info) {
        o += color ? `${ESC.green}Socket${ESC.reset}      : ${info.state}\n` : `Socket      : ${info.state}\n`;
        if (info.explanation) o += `              ${info.explanation}\n`;
        if (info.workaround) o += color ? `              ${ESC.dimYellow}${info.workaround}${ESC.reset}\n` : `              ${info.workaround}\n`;
      }
    }

    if (proc.threadCount > 1) {
      o += color ? `\n${ESC.green}Threads${ESC.reset}: ${proc.threadCount}\n` : `\nThreads: ${proc.threadCount}\n`;
    }

    if (r.children.length > 0) {
      o += '\n';
      o += printChildren(proc, r.children, color);
    }
    return o;
  }

  socketInfoFor(proc, port) {
    const override = (this.world.socketOverrides || {})[String(port)];
    if (override) return override;
    const s = (proc.sockets || []).find((x) => x.port === port);
    if (!s) return null;
    return { state: displayState(s.state), explanation: '', workaround: '' };
  }
}

function exitFor(r) {
  return r.warnings && r.warnings.length > 0 ? EXIT.WARNINGS : EXIT.OK;
}

// ---- short / tree / children / env (short.go, tree.go, children.go) -----

function chainName(p) {
  if (p.command) return p.command;
  if (p.cmdline) return p.cmdline;
  return '(unknown)';
}

function renderShort(r, color) {
  let o = '';
  r.ancestry.forEach((proc, i) => {
    if (i > 0) o += color ? `${ESC.magenta} → ${ESC.reset}` : ` → `;
    if (color) {
      const nameColor = i === r.ancestry.length - 1 ? ESC.green : '';
      o += `${nameColor}${chainName(proc)}${ESC.reset} (${ESC.dim}pid ${proc.pid}${ESC.reset})`;
    } else {
      o += `${chainName(proc)} (pid ${proc.pid})`;
    }
  });
  return o + '\n';
}

function renderTree(r, color) {
  let o = '';
  const chain = r.ancestry;
  chain.forEach((proc, i) => {
    const indent = '  '.repeat(i);
    if (i > 0) o += color ? `${indent}${ESC.magenta}└─ ${ESC.reset}` : `${indent}└─ `;
    if (color) {
      const cmdColor = i === chain.length - 1 ? ESC.green : '';
      o += `${cmdColor}${chainName(proc)}${ESC.reset} (${ESC.dim}pid ${proc.pid}${ESC.reset})\n`;
    } else {
      o += `${chainName(proc)} (pid ${proc.pid})\n`;
    }
  });

  const children = r.children;
  if (children.length === 0) return o;
  const baseIndent = '  '.repeat(chain.length);
  const limit = 10;
  const count = children.length;
  for (let i = 0; i < children.length; i++) {
    if (i >= limit) {
      const remaining = count - limit;
      o += color ? `${baseIndent}${ESC.magenta}└─ ${ESC.reset}... and ${remaining} more\n` : `${baseIndent}└─ ... and ${remaining} more\n`;
      break;
    }
    const isLast = i === count - 1 || (i === limit - 1 && count <= limit);
    const connector = isLast ? '└─ ' : '├─ ';
    const child = children[i];
    if (color) {
      o += `${baseIndent}${ESC.magenta}${connector}${ESC.reset}${chainName(child)} (${ESC.dim}pid ${child.pid}${ESC.reset})\n`;
    } else {
      o += `${baseIndent}${connector}${chainName(child)} (pid ${child.pid})\n`;
    }
  }
  return o;
}

function printChildren(root, children, color) {
  let o = '';
  const rootName = root.command || root.cmdline || 'unknown';
  o += color
    ? `${ESC.green}Children${ESC.reset} of ${rootName} (${ESC.dim}pid ${root.pid}${ESC.reset}):\n`
    : `Children of ${rootName} (pid ${root.pid}):\n`;
  if (children.length === 0) {
    return o + (color ? `${ESC.green}No child processes found.${ESC.reset}\n` : `No child processes found.\n`);
  }
  const limit = 10;
  const count = children.length;
  for (let i = 0; i < children.length; i++) {
    if (i >= limit) {
      const remaining = count - limit;
      o += color ? `  ${ESC.magenta}└─ ${ESC.reset}... and ${remaining} more\n` : `  └─ ... and ${remaining} more\n`;
      break;
    }
    const isLast = i === count - 1 || (i === limit - 1 && count <= limit);
    const connector = isLast ? '└─ ' : '├─ ';
    const child = children[i];
    const childName = child.command || child.cmdline || 'unknown';
    o += color
      ? `  ${ESC.magenta}${connector}${ESC.reset}${childName} (${ESC.dim}pid ${child.pid}${ESC.reset})\n`
      : `  ${connector}${childName} (pid ${child.pid})\n`;
  }
  return o;
}

function renderWarnings(r, color) {
  let o = '';
  const proc = r.ancestry.length > 0 ? r.ancestry[r.ancestry.length - 1] : r.process;
  if (color) {
    o += `${ESC.blue}Process${ESC.reset}     : ${ESC.green}${proc.command}${ESC.reset} (${ESC.dim}pid ${proc.pid}${ESC.reset})\n`;
    o += `${ESC.blue}Command${ESC.reset}     : ${proc.cmdline || proc.command}\n`;
  } else {
    o += `Process     : ${proc.command} (pid ${proc.pid})\n`;
    o += `Command     : ${proc.cmdline || proc.command}\n`;
  }
  if (r.warnings.length === 0) {
    return o + (color ? `${ESC.red}Warnings${ESC.reset}    : ${ESC.green}No warnings.${ESC.reset}\n` : `Warnings    : No warnings.\n`);
  }
  o += color ? `${ESC.red}Warnings${ESC.reset}    :\n` : `Warnings    :\n`;
  for (const w of r.warnings) o += `  • ${w}\n`;
  return o;
}

function renderEnvOnly(r, color) {
  let o = '';
  const proc = r.ancestry.length > 0 ? r.ancestry[r.ancestry.length - 1] : r.process;
  const name = proc.command;
  if (color) {
    o += `${ESC.blue}Process${ESC.reset}     : ${ESC.green}${name}${ESC.reset} (${ESC.dim}pid ${r.process.pid}${ESC.reset})\n`;
  } else {
    o += `Process     : ${name} (pid ${r.process.pid})\n`;
  }
  o += color ? `${ESC.blue}Command${ESC.reset}     : ${r.process.cmdline}\n` : `Command     : ${r.process.cmdline}\n`;
  const env = r.process.env || [];
  if (env.length > 0) {
    o += color ? `${ESC.blue}Environment${ESC.reset} :\n` : `Environment :\n`;
    for (const e of env) o += `  ${e}\n`;
  } else {
    o += color
      ? `${ESC.blue}Environment${ESC.reset} : ${ESC.red}No environment variables found.${ESC.reset}\n`
      : `Environment : No environment variables found.\n`;
  }
  return o;
}

// ---- container renders (docker.go) -----------------------------------

function containerSourceLabel(m) {
  if (m.runtime === 'docker' && m.composeProject && m.composeService) {
    return `docker-compose: ${m.composeProject}/${m.composeService}`;
  }
  return m.runtime || 'container';
}

function containerChain(m) {
  const runtime = m.runtime || 'container';
  const segs = [runtime];
  if (m.composeProject) segs.push(`${m.composeProject} (docker-compose)`);
  segs.push(m.name);
  return segs;
}

function containerStateTag(m) {
  const state = (m.state || '').toLowerCase();
  const health = (m.health || '').toLowerCase();
  if (health && health !== 'healthy') return health;
  if (health === 'healthy') return 'healthy';
  if (state && state !== 'running') return state;
  return '';
}

function shortContainerID(id) {
  return id && id.length > 12 ? id.slice(0, 12) : id;
}

function renderContainerFallback(label, m, color, verbose, engine) {
  let o = '';
  o += color ? `${ESC.blue}Target${ESC.reset}      : ${label}\n\n` : `Target      : ${label}\n\n`;
  const id = shortContainerID(m.id);
  o += color
    ? `${ESC.blue}Container${ESC.reset}   : ${ESC.green}${m.name}${ESC.reset} (${ESC.dim}id ${id}${ESC.reset})`
    : `Container   : ${m.name} (id ${id})`;
  const tag = containerStateTag(m);
  if (tag) {
    const c = tag === 'healthy' ? ESC.green : ESC.red;
    o += color ? ` ${c}[${tag}]${ESC.reset}` : ` [${tag}]`;
  }
  o += '\n';
  if (m.image) o += color ? `${ESC.blue}Image${ESC.reset}       : ${m.image}\n` : `Image       : ${m.image}\n`;
  if (m.command) o += color ? `${ESC.blue}Command${ESC.reset}     : ${m.command}\n` : `Command     : ${m.command}\n`;
  if (m.startedAgo != null) {
    const [rel, abs] = formatStartedAt(engine.now() - m.startedAgo * 1000, engine.now());
    o += color ? `${ESC.magenta}Started${ESC.reset}     : ${rel} (${abs})\n` : `Started     : ${rel} (${abs})\n`;
  }
  if (m.createdAgo != null && m.createdAgo !== m.startedAgo) {
    const [, abs] = formatStartedAt(engine.now() - m.createdAgo * 1000, engine.now());
    o += color ? `${ESC.blue}Created${ESC.reset}     : ${abs}\n` : `Created     : ${abs}\n`;
  }
  if (m.networks) o += color ? `${ESC.blue}Network${ESC.reset}     : ${m.networks}\n` : `Network     : ${m.networks}\n`;

  o += color ? `\n${ESC.magenta}Why It Exists${ESC.reset} :\n  ` : `\nWhy It Exists :\n  `;
  o += chainInline(containerChain(m), color) + '\n';

  o += color ? `\n${ESC.cyan}Source${ESC.reset}      : ${containerSourceLabel(m)}\n` : `\nSource      : ${containerSourceLabel(m)}\n`;

  if (m.ports) o += containerSockets(m.ports, color);

  if (verbose) {
    if (m.mounts) o += color ? `\n${ESC.blue}Mounts${ESC.reset}      : ${m.mounts}\n` : `\nMounts      : ${m.mounts}\n`;
    if (m.composeConfigFile) o += color ? `${ESC.blue}Compose File${ESC.reset}: ${m.composeConfigFile}\n` : `Compose File: ${m.composeConfigFile}\n`;
    if (m.composeWorkingDir) o += color ? `${ESC.blue}Compose Dir${ESC.reset} : ${m.composeWorkingDir}\n` : `Compose Dir : ${m.composeWorkingDir}\n`;
  }

  o += color
    ? `\n${ESC.dimYellow}Note${ESC.reset}        : The owning process is not visible in this environment.\n`
    : `\nNote        : The owning process is not visible in this environment.\n`;
  return o;
}

function containerSockets(ports, color) {
  let o = '';
  ports.split(', ').forEach((e, i) => {
    const safe = e.trim();
    if (i === 0) o += color ? `${ESC.green}Sockets${ESC.reset}     : ${safe}\n` : `Sockets     : ${safe}\n`;
    else o += `              ${safe}\n`;
  });
  return o;
}

function chainInline(segs, color) {
  let o = '';
  segs.forEach((s, i) => {
    if (i > 0) o += color ? ` ${ESC.magenta}→${ESC.reset} ` : ` → `;
    if (i === segs.length - 1 && color) o += `${ESC.green}${s}${ESC.reset}`;
    else o += s;
  });
  return o;
}

function renderContainerShort(m, color) {
  return chainInline(containerChain(m), color) + '\n';
}

function renderContainerTree(m, color) {
  let o = '';
  const segs = containerChain(m);
  segs.forEach((s, i) => {
    const indent = '  '.repeat(i);
    if (i === 0 && color && segs.length === 1) o += `${ESC.green}${s}${ESC.reset}\n`;
    else if (i === 0) o += `${s}\n`;
    else if (i === segs.length - 1 && color) o += `${indent}${ESC.magenta}└─ ${ESC.green}${s}${ESC.reset}\n`;
    else if (color) o += `${indent}${ESC.magenta}└─ ${ESC.reset}${s}\n`;
    else o += `${indent}└─ ${s}\n`;
  });
  return o;
}

function renderContainerWarnings(m, color) {
  if (color) {
    return `${ESC.blue}Container${ESC.reset}   : ${ESC.green}${m.name}${ESC.reset}\n` +
      `${ESC.red}Warnings${ESC.reset}    : ${ESC.green}No warnings (workload process not visible).${ESC.reset}\n`;
  }
  return `Container   : ${m.name}\nWarnings    : No warnings (workload process not visible).\n`;
}

// ---- JSON (json.go, docker.go) ---------------------------------------

function toJSON(r, engine) {
  // Mirrors the shape of a marshalled model.Result closely enough for the demo.
  const obj = {
    Target: { Type: r.target._targetType || 'name', Value: r.target.command },
    Process: {
      PID: r.process.pid, PPID: r.process.ppid, Command: r.process.command,
      Cmdline: r.process.cmdline || '', User: r.process.user || '',
      StartedAt: formatStartedAt(engine.now() - (r.process.startedAgo || 0) * 1000, engine.now())[1],
      WorkingDir: r.process.workingDir || '', GitRepo: r.process.gitRepo || '',
      GitBranch: r.process.gitBranch || '',
    },
    Ancestry: r.ancestry.map((p) => ({ PID: p.pid, Command: p.command })),
    Source: { Type: r.source.type, Name: r.source.name || '', Description: r.source.description || '' },
    Warnings: r.warnings || [],
  };
  return JSON.stringify(obj, null, 2);
}

function toTreeJSON(r) {
  const obj = { Ancestry: r.ancestry.map((p) => ({ PID: p.pid, Command: p.command })) };
  if (r.children.length > 0) obj.Children = r.children.map((p) => ({ PID: p.pid, Command: p.command }));
  return JSON.stringify(obj, null, 2);
}

function toWarningsJSON(r) {
  const proc = r.ancestry.length > 0 ? r.ancestry[r.ancestry.length - 1] : r.process;
  return JSON.stringify({
    PID: r.process.pid, Process: proc.command,
    Command: r.process.cmdline || r.process.command, Warnings: r.warnings || [],
  }, null, 2);
}

function toEnvJSON(r) {
  const proc = r.ancestry.length > 0 ? r.ancestry[r.ancestry.length - 1] : r.process;
  return JSON.stringify({
    PID: r.process.pid, Process: proc.command,
    Command: r.process.cmdline || '', Env: r.process.env || [],
  }, null, 2);
}

function containerToJSON(label, m) {
  const obj = {
    Target: label, Runtime: m.runtime, ContainerID: m.id, ContainerName: m.name,
    Image: m.image, Command: m.command, State: m.state, Status: m.status, Health: m.health,
    Networks: m.networks, Mounts: m.mounts, Ports: m.ports,
    ComposeProject: m.composeProject, ComposeService: m.composeService,
    ComposeConfigFile: m.composeConfigFile, ComposeWorkingDir: m.composeWorkingDir,
    Source: containerSourceLabel(m), Chain: containerChain(m),
    Note: 'The owning process is not visible in this environment. This is common when the runtime runs in a separate namespace (e.g., Docker Desktop, WSL2 distro, macOS VM).',
  };
  // Drop empty optional fields to mirror `json:",omitempty"`.
  for (const k of ['Command', 'State', 'Status', 'Health', 'Networks', 'Mounts', 'Ports', 'ComposeProject', 'ComposeService', 'ComposeConfigFile', 'ComposeWorkingDir']) {
    if (!obj[k]) delete obj[k];
  }
  return JSON.stringify(obj, null, 2);
}

function jsonErrorEntry(t, msg) {
  return JSON.stringify({ Target: { Type: t.type, Value: t.value }, Error: msg }, null, 2);
}

// ---- dividers / multi-match containers (app.go) ----------------------

function targetLabel(t) {
  switch (t.type) {
    case 'pid': return `pid: ${t.value}`;
    case 'port': return `port: ${t.value}`;
    case 'file': return `file: ${t.value}`;
    case 'container': return `container: ${t.value}`;
    default: return `name: ${t.value}`;
  }
}

function printDivider(t, color, needsNewline) {
  const label = targetLabel(t);
  let o = '';
  if (needsNewline) o += '\n';
  o += color ? `${ESC.cyan}----- [${label}] -----${ESC.reset}\n` : `----- [${label}] -----\n`;
  return o;
}

function printContainerMultiMatch(matches, color) {
  let o = 'Multiple matching containers found:\n\n';
  matches.forEach((m, i) => {
    if (color) o += `[${i + 1}] ${ESC.green}${m.name}${ESC.reset} (${ESC.dim}${m.runtime}${ESC.reset})\n`;
    else o += `[${i + 1}] ${m.name} (${m.runtime})\n`;
    let detail = `image: ${m.image}`;
    if (m.status) detail += `, status: ${m.status}`;
    if (m.ports) detail += `, ports: ${m.ports}`;
    o += `    ${detail}\n`;
  });
  o += '\nRe-run with the exact container name to disambiguate:\n';
  o += '  witr -c <container-name> --exact\n';
  return o;
}

// ---- sockets (standard.go) -------------------------------------------

function displayState(state) {
  if (state === '') return '?';
  if (state === 'LISTEN') return 'LISTENING';
  return state;
}

function socketSortRank(state) {
  switch (state) {
    case 'LISTEN': return 0;
    case 'OPEN': return 1;
    case 'ESTABLISHED': return 2;
    default: return 3;
  }
}

function visibleSockets(sockets) {
  return sockets.filter((s) => s.address !== '' && s.port > 0).map((s) => ({ ...s }));
}

function sortSockets(sockets) {
  sockets.sort((a, b) => {
    if (a.address !== b.address) return a.address < b.address ? -1 : 1;
    if (a.port !== b.port) return a.port - b.port;
    return socketSortRank(a.state) - socketSortRank(b.state);
  });
}

function joinHostPort(addr, port) {
  // net.JoinHostPort wraps IPv6 (addresses containing ':') in brackets.
  if (addr.includes(':')) return `[${addr}]:${port}`;
  return `${addr}:${port}`;
}

function formatSocket(s) {
  const hostPort = joinHostPort(s.address, s.port);
  const proto = s.protocol || '?';
  return `${hostPort} (${proto} | ${displayState(s.state)})`;
}

// ---- bytes (standard.go) ---------------------------------------------

function formatBytes(n) {
  const unit = 1024;
  if (n < unit) return `${n} B`;
  let div = unit;
  let exp = 0;
  for (let m = Math.floor(n / unit); m >= unit; m = Math.floor(m / unit)) {
    div *= unit;
    exp++;
  }
  return `${(n / div).toFixed(1)} ${'KMGTPE'[exp]}B`;
}

// ---- started (started.go) --------------------------------------------

const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

// formatStartedAt(absoluteMs, nowMs) -> [relativePhrase, absoluteString].
// Ports internal/output/started.go exactly; the absolute timestamp is rendered
// in UTC so the browser and the Go fixture generator agree byte-for-byte.
export function formatStartedAt(ms, nowMs) {
  if (ms == null) return ['unknown', ''];
  const dur = nowMs - ms; // ms
  const hours = dur / 3600000;
  const minutes = dur / 60000;
  let rel;
  if (hours >= 48) rel = `${Math.floor(Math.floor(hours) / 24)} days ago`;
  else if (hours >= 24) rel = '1 day ago';
  else if (hours >= 2) rel = `${Math.floor(hours)} hours ago`;
  else if (minutes >= 60) rel = '1 hour ago';
  else {
    const mins = Math.floor(minutes);
    rel = mins > 0 ? `${mins} min ago` : 'just now';
  }
  const abs = formatAbsUTC(ms);
  return [rel, abs];
}

function formatAbsUTC(ms) {
  const d = new Date(ms);
  const wd = WEEKDAYS[d.getUTCDay()];
  const y = d.getUTCFullYear();
  const mo = String(d.getUTCMonth() + 1).padStart(2, '0');
  const da = String(d.getUTCDate()).padStart(2, '0');
  const h = String(d.getUTCHours()).padStart(2, '0');
  const mi = String(d.getUTCMinutes()).padStart(2, '0');
  const s = String(d.getUTCSeconds()).padStart(2, '0');
  return `${wd} ${y}-${mo}-${da} ${h}:${mi}:${s} +00:00`;
}

function matchesExactToken(cmdLower, nameLower) {
  return cmdLower.split(/\s+/).some((tok) => {
    const base = tok.split('/').pop();
    return tok === nameLower || base === nameLower;
  });
}
