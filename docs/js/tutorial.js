// tutorial.js — per-scenario incidents.
//
// Each scenario is a single short incident: a cold open that plays itself, then
// a box with real problems the visitor traces with witr. On webbox the tasks
// are informational (witr gives the *why*; the operator decides the fix); on
// devbox they're fix-by-kill (processes they kill, a lock that clears). A
// tracker counts down as each task resolves; clearing them all is the finale
// with the install command. Feature coverage (port/file/pid/verbose/chain)
// falls out of the investigation; the rest live as optional side quests that
// tick off as they're tried.

const gone = (pid) => (w) => !w.processes.some((p) => p.pid === pid);
const targetsPid = (pid) => (ts) => ts.some((t) => t.type === 'pid' && t.value === String(pid));
const targetsName = (name) => (ts) => ts.some((t) => t.type === 'name' && name.includes(t.value.toLowerCase()));

export const INCIDENTS = {
  webbox: {
    coldOpen: [
      { type: 'line', html: '<span class="co-prompt">deploy@webbox</span><span class="co-sep">:</span><span class="co-dir">~</span><span class="co-sep">$</span> ./deploy.sh', delay: 333 },
      { type: 'line', html: '<span class="a-dim">▸ building expense-manager …</span> <span class="a-green">done</span>', delay: 433 },
      { type: 'line', html: '<span class="a-dim">▸ health-checking :5000 …</span> <span class="a-green">ok</span>', delay: 433 },
      { type: 'line', html: '<span class="a-dim">▸ starting metrics endpoint on :8000 …</span>', delay: 533 },
      { type: 'line', html: '<span class="a-red">✗ Error: listen EADDRINUSE: address already in use 0.0.0.0:8000</span>', delay: 333 },
      { type: 'line', html: '<span class="a-dim">  deploy aborted. something is already on that port.</span>', delay: 733 },
      { type: 'note', html: 'Every deploy hits this eventually — and <b>witr</b> answers it in one command. Your turn: investigate what’s on the port.', delay: 900 },
    ],
    briefing: 'That <span class="a-red">EADDRINUSE</span> is one of <b>three</b> questions a quick sweep raised on <b>webbox</b>. None of these need forcing — trace each and let witr tell you the <i>why</i>. Start by investigating what’s on :8000; the tracker on the left counts down.',
    issues: [
      {
        id: 'squatter', severity: 'high', title: 'What is holding :8000?',
        blurb: "Your deploy just died on <code>EADDRINUSE</code> — something is already bound to <b>:8000</b>. But <i>what</i> is it, and who started it? Ask witr: <code>witr --port 8000</code>.",
        find: 'witr --port 8000',
        touched: (c) => c.targets.some((t) => (t.type === 'port' && t.value === '8000')) || targetsPid(8123)(c.targets),
        resolveOnFind: true,
        done: "There it is — a stray <code>python3 -m http.server</code> (pid 8123) a teammate backgrounded over SSH, still holding the port. Your call now: free it with <code>kill 8123</code>, or point the deploy at another port. witr gave you the <i>why</i>; the fix is yours.",
      },
      {
        id: 'lock', severity: 'warn', title: 'apt won’t run — who holds the dpkg lock?',
        blurb: "A teammate says <code>apt</code> is frozen — something is holding <code>/var/lib/dpkg/lock</code>. Before assuming the worst, ask witr <i>who</i> has it: <code>witr --file /var/lib/dpkg/lock</code>.",
        find: 'witr --file /var/lib/dpkg/lock',
        touched: (c) => c.targets.some((t) => (t.type === 'file' && t.value.includes('dpkg')) || (t.type === 'pid' && t.value === '33871')),
        resolveOnFind: true,
        done: "It's the scheduled <b>unattended-upgrade</b> (pid 33871) — completely expected; someone just forgot it runs on a timer. Nothing is stuck: it frees the lock the moment it finishes. Knowing <i>why</i> was the whole fix.",
      },
      {
        id: 'verbose', severity: 'warn', title: 'How heavy is the app, really?',
        blurb: "The Node app has been up for weeks — is it holding more than it should? Rather than guess at its footprint, ask witr for the whole picture: <code>witr node --verbose</code>.",
        find: 'witr node --verbose',
        touched: (c) => !!c.flags.verbose && c.targets.some((t) => t.type === 'name' && 'node'.includes((t.value || '').toLowerCase())),
        resolveOnFind: true,
        done: "There's the deep dive — memory, threads, open files, sockets and I/O in one view. Add <code>--verbose</code> to any query when you need the full footprint of a process.",
      },
    ],
    sideQuests: [
      { id: 'tree', cmd: 'witr node --tree', label: 'the full family tree', test: (c) => c.flags.tree, note: '<b>--tree</b> draws the whole ancestry top-down and lists the target’s children.' },
      { id: 'short', cmd: 'witr node --short', label: 'the one-line causal chain', test: (c) => c.flags.short, note: '<b>--short</b> collapses the answer to a single causal line — great for scripts and chat.' },
      { id: 'json', cmd: 'witr node --json', label: 'machine-readable output', test: (c) => c.flags.json, note: 'Any query takes <b>--json</b>; witr also returns exit codes — <code>0</code> clean, <code>1</code> warnings, <code>2</code> not-found.' },
      { id: 'verbose', cmd: 'witr node --verbose', label: 'the deep dive', test: (c) => c.flags.verbose, note: '<b>--verbose</b> adds memory, threads, open files and sockets — the full picture.' },
      { id: 'env', cmd: 'witr node --env', label: 'environment variables', test: (c) => c.flags.env, note: '<b>--env</b> dumps the process’s environment variables.' },
      { id: 'container', cmd: 'witr --container redis', label: 'a container with no host process', test: (c) => c.targets.some((t) => t.type === 'container'), note: '<b>--container</b> finds Docker/Podman/compose workloads by name, image or service — even with no visible host process.' },
      { id: 'tui', cmd: 'witr', label: 'the live TUI dashboard', test: (c) => c.action === 'tui', note: '<b>witr</b> with no arguments opens the live TUI — Processes / Ports / Containers / Locks.' },
    ],
  },

  devbox: {
    coldOpen: [
      { type: 'line', html: '<span class="co-prompt">pranshu@devbox</span><span class="co-sep">:</span><span class="co-dir">~/projects/shop</span><span class="co-sep">$</span> git commit -m "wip"', delay: 500 },
      { type: 'line', html: '<span class="a-red">fatal: Unable to create \'.git/index.lock\': File exists.</span>', delay: 550 },
      { type: 'line', html: '<span class="a-dim">  Another git process seems to be running in this repository.</span>', delay: 1100 },
      { type: 'note', html: 'A stale lock — but which process? Your turn: ask <b>witr</b> to resolve the file to its owner.', delay: 900 },
    ],
    briefing: "That stale <code>.git/index.lock</code> is one of three things gumming up <b>devbox</b> — plus a <code>python3</code> zombie nobody reaped and something pinning the CPU. Trace each with witr and sort it out.",
    issues: [
      {
        id: 'gitlock', severity: 'high', title: 'git index.lock blocking every commit',
        blurb: "Every <code>git</code> command in <code>shop</code> dies with <i>Unable to create '.git/index.lock': File exists</i>. Some process is holding that lock — <code>witr --file …/.git/index.lock</code> will name it.",
        foundBlurb: "There it is — a crashed <code>git commit</code> (pid 7300) still clutching <code>.git/index.lock</code>. It's doing no work; clearing the stale lock unblocks every git command.",
        find: 'witr --file /home/pranshu/projects/shop/.git/index.lock', fixHint: 'kill 7300', fixLabel: 'Clear the stale lock',
        touched: (c) => targetsPid(7300)(c.targets) || c.targets.some((t) => t.type === 'file' && t.value.includes('index.lock')),
        resolved: gone(7300), done: "Lock released — git works again. witr traced it to a crashed <code>git commit</code> that never let go; clearing the stale lock was all it needed.",
      },
      {
        id: 'zombie', severity: 'warn', title: 'Zombie process nobody reaped',
        blurb: "A <code>python3</code> process is stuck <b>&lt;defunct&gt;</b> — a zombie. You can't kill a zombie directly; it only clears once its parent reaps it. <code>witr --pid 6120</code> shows whose child it is.",
        foundBlurb: "Its parent is <code>build.sh</code> (pid 6100), which never reaped it. End the parent and the kernel clears the zombie — killing the zombie itself does nothing.",
        find: 'witr --pid 6120', fixHint: 'kill 6100', fixLabel: 'Reap via parent · kill 6100',
        touched: (c) => targetsPid(6120)(c.targets) || targetsPid(6100)(c.targets),
        resolved: gone(6120), done: 'Parent gone, zombie reaped. A defunct child only clears when its parent waits on it (or dies) — killing the zombie itself does nothing.',
      },
      {
        id: 'ffmpeg', severity: 'high', title: 'A process is pinning the CPU',
        blurb: "<code>top</code> shows <b>pid 6001</b> pegged near <b>98% CPU</b> and the fans are screaming — but the pid alone tells you nothing. What is it, and is it safe to kill? <code>witr --pid 6001</code>.",
        foundBlurb: "It's a stray <code>ffmpeg</code> encode you kicked off from a shell and forgot — nothing depends on it, so it's safe to stop. Checking first is why you reach for witr, not just <code>kill</code>.",
        find: 'witr --pid 6001', fixHint: 'kill 6001', fixLabel: 'Stop it · kill 6001',
        touched: (c) => targetsPid(6001)(c.targets) || targetsName('ffmpeg')(c.targets),
        resolved: gone(6001), done: "CPU's back to idle. The fans can rest.",
      },
    ],
    sideQuests: [
      { id: 'tree', cmd: 'witr code --tree', label: "VS Code's whole process family", test: (c) => c.flags.tree, note: '<b>--tree</b> draws the whole ancestry top-down and lists the target’s children.' },
      { id: 'short', cmd: 'witr --port 5173 --short', label: 'the Vite server in one line', test: (c) => c.flags.short, note: '<b>--short</b> collapses the answer to a single causal line — great for scripts and chat.' },
      { id: 'json', cmd: 'witr code --json', label: 'machine-readable output', test: (c) => c.flags.json, note: 'Any query takes <b>--json</b>; witr also returns exit codes — <code>0</code> clean, <code>1</code> warnings, <code>2</code> not-found.' },
      { id: 'verbose', cmd: 'witr --port 5173 --verbose', label: 'the deep dive', test: (c) => c.flags.verbose, note: '<b>--verbose</b> adds memory, threads, open files and sockets — the full picture.' },
      { id: 'env', cmd: 'witr --port 5173 --env', label: 'environment variables', test: (c) => c.flags.env, note: '<b>--env</b> dumps the process’s environment variables.' },
      { id: 'containers', cmd: 'witr --container shop', label: 'the docker-compose stack', test: (c) => c.targets.some((t) => t.type === 'container'), note: '<b>--container</b> matches every container in a compose project — pass the exact name to pick one.' },
      { id: 'tui', cmd: 'witr', label: 'the live TUI dashboard', test: (c) => c.action === 'tui', note: '<b>witr</b> with no arguments opens the live TUI — Processes / Ports / Containers / Locks.' },
    ],
  },
};

export class Incident {
  constructor() {
    this.active = false;
    this.phase = 'idle'; // idle | coldopen | investigating | done
    this.def = null;
    this.found = new Set();
    this.resolved = new Set();
    this.tried = new Set();
    this.onChange = null;
    this.onResolve = null;
    this.onComplete = null;
    this.onQuestTried = null;
  }

  load(def) { this.def = def; }
  issues() { return this.def ? this.def.issues : []; }
  sideQuests() { return this.def ? this.def.sideQuests || [] : []; }

  start() {
    this.active = true;
    this.phase = 'coldopen';
    this.found.clear();
    this.resolved.clear();
    this.tried.clear();
    this._emit();
  }

  stop() { this.active = false; this.phase = 'idle'; this._emit(); }
  beginInvestigation() { if (this.active) { this.phase = 'investigating'; this._emit(); } }

  total() { return this.issues().length; }
  remaining() { return this.total() - this.resolved.size; }

  status(issue) {
    if (this.resolved.has(issue.id)) return 'resolved';
    if (this.found.has(issue.id)) return 'found';
    return 'open';
  }

  // Called after each executed command. ctx = { targets, flags, action, world }.
  observe(ctx) {
    if (!this.active) return [];
    ctx = { ...ctx, targets: ctx.targets || [], flags: ctx.flags || {} };

    // Side quests keep tracking even after the incident is resolved, so the
    // finale checklist ticks off as they're tried.
    for (const q of this.sideQuests()) {
      if (!this.tried.has(q.id)) {
        try {
          if (q.test(ctx)) {
            this.tried.add(q.id);
            if (this.onQuestTried) this.onQuestTried(q);
            this._emit();
          }
        } catch (_) {}
      }
    }
    if (this.phase === 'done') return [];

    const newlyResolved = [];
    for (const issue of this.issues()) {
      if (!this.found.has(issue.id) && issue.touched(ctx)) {
        this.found.add(issue.id);
        // Informational issues are answered the moment they're investigated —
        // witr tells you the why, and that's the whole task.
        if (issue.resolveOnFind && !this.resolved.has(issue.id)) {
          this.resolved.add(issue.id);
          newlyResolved.push(issue);
        }
        this._emit();
      }
    }

    for (const issue of this.issues()) {
      if (this.resolved.has(issue.id)) continue;
      if (typeof issue.resolved === 'function' && issue.resolved(ctx.world)) {
        this.resolved.add(issue.id); newlyResolved.push(issue);
      }
    }
    for (const issue of newlyResolved) if (this.onResolve) this.onResolve(issue);
    if (newlyResolved.length) this._emit();
    if (this.remaining() === 0 && this.total() > 0 && this.phase !== 'done') {
      this.phase = 'done';
      if (this.onComplete) this.onComplete();
      this._emit();
    }
    return newlyResolved;
  }

  _emit() { if (this.onChange) this.onChange(); }
}
