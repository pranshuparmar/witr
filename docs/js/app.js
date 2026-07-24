// app.js — wires the playground together.

import { Shell } from './shell.js';
import { Terminal } from './terminal.js';
import { SystemMap } from './map.js';
import { Tree } from './tree.js';
import { Incident, INCIDENTS } from './tutorial.js';
import { TUI } from './tui.js';
import { parse, tokenize } from './parser.js';
import { track } from './analytics.js';

const VERSION_URL = 'https://raw.githubusercontent.com/pranshuparmar/witr/main/internal/version/VERSION';

const WORLD_IDS = ['webbox', 'devbox'];
const COMPLETIONS = ['witr', 'ls', 'cat', 'ps', 'top', 'kill', 'pwd', 'cd', 'whoami', 'hostname', 'uname', 'neofetch', 'clear', 'help', 'scenario'];
const WITR_FLAGS = ['--pid', '--port', '--file', '--container', '--short', '--tree', '--json', '--env', '--warnings', '--verbose', '--exact', '--no-color', '--interactive', '--help', '--version'];
const INSTALL_CMD = 'curl -fsSL https://raw.githubusercontent.com/pranshuparmar/witr/main/install.sh | bash';
// Shown (middle trimmed so it fits without a scrollbar); the full command is copied.
const INSTALL_CMD_SHORT = 'curl -fsSL …/install.sh | bash';

class App {
  constructor() {
    this.pristine = {};   // worlds as loaded (never mutated)
    this.worldId = 'webbox';
    this.live = null;     // the mutable working copy
    this._skipCold = false;
    this._autoTimers = {};
  }

  async boot() {
    for (const id of WORLD_IDS) {
      const res = await fetch(`./worlds/${id}.json`);
      this.pristine[id] = await res.json();
    }
    this.live = cloneWorld(this.pristine[this.worldId]);

    this.shell = new Shell(this.live);
    this.term = new Terminal(document.getElementById('terminal'));
    this.map = new SystemMap(document.getElementById('map-canvas'), document.getElementById('map-labels'));
    this.tree = new Tree(document.getElementById('tree-view'));
    this.incident = new Incident();
    this.tui = new TUI(document.getElementById('tui'));

    this.term.onSubmit = (line) => this.handle(line);
    this.term.completer = (v) => this.complete(v);
    this.map.onSelect = (proc) => this.launchFromPid(proc.pid);
    this.map.onClear = () => this.viewClear();
    this.tree.onSelect = (pid) => this.launchFromPid(pid);
    this.tui.onClose = () => this.term.focus();
    this.tui.onKill = (pid) => this.killFromTui(pid);

    this.incident.onChange = () => this.renderIncident();
    this.incident.onResolve = (issue) => { track(`task-resolved-${issue.id}`); this.onIssueResolved(issue); };
    this.incident.onComplete = () => { track(`tutorial-completed-${this.worldId}`); this.onIncidentComplete(); };
    this.incident.onQuestTried = (q) => { track(`quest-tried-${q.id}`); this.onQuestTried(q); };

    this.viewSetWorld(this.live);
    this.map.start();
    window.addEventListener('resize', () => this.map.resize());

    this.wireChrome();
    this.setupView();
    this.setupResizer();
    this.setupLegend();
    this.applyWorld();
    this.enterScenario(true);
    this.fetchVersion();
    this.term.focus();
  }

  // ---- process views (tree + constellation kept in sync) ----------------

  viewSetWorld(w) { this.map.setWorld(w); this.tree.setWorld(w); }
  viewHighlight(pids) { this.resetLegendUI(); this.map.highlightPids(pids); this.tree.highlightPids(pids); }
  viewClear() { this.resetLegendUI(); this.map.clearHighlight(); this.tree.clearHighlight(); }
  viewRemove(pid) { this.map.removeProcess(pid); this.tree.setWorld(this.currentWorld()); }

  // ---- constellation legend (click a category to light up its nodes) ------

  setupLegend() {
    document.querySelectorAll('.lg-item').forEach((b) =>
      b.addEventListener('click', () => this.toggleLegend(b.dataset.legend, b)));
  }

  toggleLegend(type, btn) {
    if (this._legendActive === type) {
      this._legendActive = null;
      this.map.clearHighlight();
      this.resetLegendUI();
      return;
    }
    this._legendActive = type;
    this.map.highlightByType(type);
    this.tree.clearHighlight();
    document.querySelectorAll('.lg-item').forEach((x) => x.classList.toggle('active', x === btn));
  }

  resetLegendUI() {
    this._legendActive = null;
    document.querySelectorAll('.lg-item.active').forEach((x) => x.classList.remove('active'));
  }

  setupView() {
    let v = 'tree';
    try { v = localStorage.getItem('witr-view') || 'tree'; } catch (_) {}
    document.querySelectorAll('.vt').forEach((b) => b.addEventListener('click', () => this.setView(b.dataset.view)));
    this.setView(v);
  }

  setView(v) {
    this.view = v === 'map' ? 'map' : 'tree';
    document.getElementById('view-panel').classList.toggle('show-map', this.view === 'map');
    document.querySelectorAll('.vt').forEach((b) => b.classList.toggle('active', b.dataset.view === this.view));
    document.getElementById('view-hint').textContent = this.view === 'map' ? 'click a node to inspect · empty space to reset' : 'click a row to inspect it';
    try { localStorage.setItem('witr-view', this.view); } catch (_) {}
    if (this.view === 'map') requestAnimationFrame(() => this.map.resize());
  }

  setupResizer() {
    const handle = document.getElementById('vsplit');
    const col = document.querySelector('.right-col');
    let dragging = false;
    const move = (e) => {
      if (!dragging) return;
      const rect = col.getBoundingClientRect();
      const cy = e.touches ? e.touches[0].clientY : e.clientY;
      const pct = Math.max(18, Math.min(80, ((cy - rect.top) / rect.height) * 100));
      col.style.setProperty('--incident-h', pct + '%');
    };
    const stop = () => { dragging = false; handle.classList.remove('dragging'); document.body.style.userSelect = ''; };
    handle.addEventListener('pointerdown', (e) => { dragging = true; handle.classList.add('dragging'); document.body.style.userSelect = 'none'; e.preventDefault(); });
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', stop);
  }

  async fetchVersion() {
    try {
      const res = await fetch(VERSION_URL);
      if (res.ok) { const v = (await res.text()).trim(); if (v) this.shell.setVersion('v' + v); }
    } catch (_) { /* offline / file:// — keep the fallback */ }
  }

  // ---- scenario entry ---------------------------------------------------

  enterScenario() {
    const def = INCIDENTS[this.worldId];
    if (def) {
      track(`tutorial-started-${this.worldId}`);
      this.incident.load(def);
      this.incident.start();      // phase = coldopen
      this.playColdOpen(def);
    } else {
      this.incident.stop();
      this.welcome();
    }
  }

  // ---- cold open (plays itself) -----------------------------------------

  async playColdOpen(def) {
    this._skipCold = false;
    this.term.locked = true;   // hides the live prompt (setter) so it isn't shown twice during the cold open
    this.renderIncident();
    for (const step of def.coldOpen) {
      await this.sleep(step.delay);
      if (step.type === 'line') this.term.printHtml(`<div class="co-line">${step.html}</div>`);
      else if (step.type === 'note') this.term.printHtml(`<div class="co-note">${step.html}</div>`);
      else if (step.type === 'run') {
        this.term.locked = false;
        await this.term.typeAndRun(step.cmd, { speed: this._skipCold ? 6 : 34 });
      }
    }
    this.term.locked = false;
    this.incident.beginInvestigation();
    this.term.printHtml(`<div class="co-brief"><span class="co-brief-tag">🚨 Incident</span> ${def.briefing}</div>`);
    this.renderIncident();
    this.term.focus();
    // The whole cold-open story is now on screen. Rewind to the top so the
    // reader can take in the first witr answer from the beginning, at their own
    // pace, instead of landing at the bottom of a tall scrollback. Do it after a
    // beat (and once more) so it wins against the focus/print scroll-to-bottom.
    this.term.scrollToTop();
    setTimeout(() => this.term.scrollToTop(), 60);
  }

  skipColdOpen() { this._skipCold = true; }

  sleep(ms) {
    return new Promise((resolve) => {
      if (this._skipCold) return resolve();
      setTimeout(resolve, ms);
    });
  }

  // ---- command handling -------------------------------------------------

  handle(line) {
    const res = this.shell.exec(line);
    if (res.action === 'clear') { this.term.clear(); return; }
    if (res.output) this.term.print(res.output);

    if (res.action === 'tui') {
      this.term.print(dimNote('opening interactive dashboard… (press q or Esc to return)'));
      setTimeout(() => this.tui.show(this.currentWorld(), this.shell.engine, this.shell.version), 260);
    }
    if (res.action === 'scenario') this.openScenario();
    if (res.action === 'killed' && res.killed) {
      for (const pid of res.killed) this.map.removeProcess(pid);
      this.tree.setWorld(this.currentWorld());
      this.refreshHostChip();
    }

    const ctx = this.analyze(line, res);
    this.updateMap(ctx);
    this.incident.observe(ctx);
    this.maybeScheduleAutoResolve();
    if (this.incident.phase === 'done') this.refreshQuests();
    this.term.setPrompt(this.shell.prompt());
  }

  // Keep the finale's side-quest checklist in sync as they're tried afterwards.
  refreshQuests() {
    const el = document.getElementById('finale-quests');
    if (!el) return;
    el.innerHTML = `<span class="fq-h">Keep poking:</span>${this.questsHtml()}`;
    el.querySelectorAll('[data-cmd]').forEach((b) =>
      b.addEventListener('click', () => { if (!this.term.locked) this.term.typeAndRun(b.dataset.cmd); }));
  }

  analyze(line, res) {
    const tokens = tokenize(line.trim());
    const isWitr = tokens[0] === 'witr';
    const { targets, flags } = isWitr ? parse(tokens.slice(1)) : { targets: [], flags: {} };
    return { line, isWitr, targets, flags, exit: res.exit, action: res.action, world: this.currentWorld() };
  }

  updateMap(ctx) {
    if (!ctx.isWitr || ctx.targets.length === 0) return;
    const eng = this.shell.engine;
    for (const t of ctx.targets) {
      let pid = null;
      if (t.type === 'pid') pid = eng.procByPid.has(+t.value) ? +t.value : null;
      else if (t.type === 'port') pid = eng.resolvePort(+t.value);
      else if (t.type === 'file') pid = eng.resolveFile(t.value);
      else if (t.type === 'name') { const m = eng.resolveName(t.value, ctx.flags.exact); if (m.length === 1) pid = m[0]; }
      else if (t.type === 'container') {
        const runtime = this.currentWorld().processes.find((p) => /docker|containerd/.test(p.command));
        if (runtime) pid = runtime.pid;
      }
      if (pid) {
        const proc = eng.procByPid.get(pid);
        if (proc) { this.viewHighlight(eng.ancestryOf(proc).map((p) => p.pid)); return; }
      }
    }
    this.viewClear();
  }

  launchFromPid(pid) {
    if (this.tui.open || this.term.locked) return;
    if (!this.shell.engine.procByPid.has(pid)) return;
    this.term.focus();
    this.term.typeAndRun(`witr --pid ${pid}`);
  }

  // ---- lock auto-resolve ------------------------------------------------

  // Some issues (e.g. the dpkg lock) resolve on their own once investigated —
  // schedule that the first time such an issue is found.
  maybeScheduleAutoResolve() {
    if (!this.incident.active) return;
    for (const issue of this.incident.issues()) {
      const cfg = issue.autoResolve;
      if (!cfg) continue;
      if (this._autoTimers[issue.id] || this.incident.resolved.has(issue.id)) continue;
      if (!this.incident.found.has(issue.id)) continue;
      if (cfg.waiting) {
        this.term.printHtml(`<div class="learned"><span class="learned-badge a-dimyellow">…</span> ${cfg.waiting}</div>`);
      }
      this._autoTimers[issue.id] = setTimeout(() => {
        const w = this.currentWorld();
        const remove = new Set(cfg.remove || []);
        w.processes = w.processes.filter((p) => !remove.has(p.pid));
        w.locks = (w.locks || []).filter((l) => !remove.has(l.pid));
        this.shell.engine.reindex();
        for (const pid of remove) this.map.removeProcess(pid);
        this.tree.setWorld(w);
        this.refreshHostChip();
        this.incident.observe({ targets: [], flags: {}, world: w });
      }, cfg.delayMs);
    }
  }

  // ---- incident outcomes ------------------------------------------------

  onIssueResolved(issue) {
    this.term.printHtml(`<div class="learned"><span class="learned-badge">✓ Resolved</span> ${issue.done || (issue.autoResolve && issue.autoResolve.done) || ''}</div>`);
  }

  onIncidentComplete() {
    const w = this.currentWorld();
    this.term.printHtml(`<div class="finale-card">
      <div class="finale-badge">✓ ${escapeHtml(w.hostname)} is green</div>
      <div class="finale-title">You just worked an incident with witr — every question traced to its <i>why</i> in one command.</div>
      <div class="finale-sub">It does exactly this on a real machine, against live processes:</div>
      <div class="tut-install-row">
        <pre class="tut-install" title="${escapeAttr(INSTALL_CMD)}">${INSTALL_CMD_SHORT}</pre>
        <button class="tut-copy" data-copy="${escapeAttr(INSTALL_CMD)}" title="Copy install command"><span class="copy-icon">⧉</span> Copy</button>
      </div>
      <div class="finale-tip">Lost, or want the full reference? Run <button class="tip-cmd" data-cmd="witr --help"><code>witr --help</code></button> anytime.</div>
      <div class="finale-quests" id="finale-quests"><span class="fq-h">Keep poking:</span>${this.questsHtml()}</div>
    </div>`);
    // Wire every command button in the finale card (the tip + the quests).
    const card = this.term.output.lastElementChild;
    if (card) {
      card.querySelectorAll('[data-cmd]').forEach((b) =>
        b.addEventListener('click', () => { if (!this.term.locked) this.term.typeAndRun(b.dataset.cmd); }));
      const cp = card.querySelector('[data-copy]');
      if (cp) cp.addEventListener('click', () => this.copyToClipboard(cp.dataset.copy, cp));
    }
    this.term.scroll();
  }

  questsHtml() {
    return this.incident.sideQuests().map((q) => {
      const done = this.incident.tried.has(q.id);
      return `<button class="sq${done ? ' done' : ''}" data-cmd="${escapeAttr(q.cmd)}"><span class="sq-ic">${done ? '✓' : '○'}</span><code>${escapeHtml(q.cmd)}</code> — ${q.label}</button>`;
    }).join('');
  }

  // ---- incident / free-play panel ---------------------------------------

  renderIncident() {
    const panel = document.getElementById('tutorial');
    // Give the map the whole right column when there's no incident. The topbar
    // button doubles as a mode indicator: highlighted "Tutorial" while one runs,
    // an un-highlighted "Free play" invitation once you've stepped out.
    document.querySelector('.layout').classList.toggle('no-incident', !this.incident.active);
    const btnT = document.getElementById('btn-tutorial');
    // Both modes (guided tutorial and free play) are "active" states — keep the
    // button highlighted throughout; only the label changes.
    btnT.classList.add('active');
    btnT.textContent = this.incident.active ? 'Tutorial' : 'Free play';
    btnT.title = this.incident.active ? 'Exit the tutorial and explore freely' : 'Restart the guided tutorial';
    if (!this.incident.active) { panel.classList.add('hidden'); return; }
    panel.classList.remove('hidden');

    if (this.incident.phase === 'coldopen') {
      panel.innerHTML = `
        <div class="tut-head"><span class="tut-kicker alert">● Incident detected</span>
          <button class="tut-skip" data-skip>Skip intro ⏭</button></div>
        <h2 class="tut-title">${escapeHtml(this.currentWorld().hostname)}</h2>
        <p class="tut-story">Something just broke. Watching witr trace the cause…</p>`;
      const sk = panel.querySelector('[data-skip]');
      if (sk) sk.addEventListener('click', () => this.skipColdOpen());
      return;
    }

    const done = this.incident.remaining() === 0;
    const total = this.incident.total();
    const resolved = total - this.incident.remaining();
    const issues = this.incident.issues();
    // Flash the single button the visitor should reach for next — the first
    // issue that still needs an action — so it's obvious where to go.
    const nextIssue = issues.find((iss) => {
      const s = this.incident.status(iss);
      return s === 'open' || (s === 'found' && iss.fixHint);
    });
    const nextId = nextIssue ? nextIssue.id : null;
    const rows = issues.map((issue) => {
      const st = this.incident.status(issue);
      const icon = st === 'resolved' ? '✓' : (st === 'found' ? '◔' : '○');
      const flash = issue.id === nextId ? ' flash' : '';
      let action = '';
      if (st === 'open') {
        action = `<button class="btn btn-sm${flash}" data-cmd="${escapeAttr(issue.find)}">Investigate</button>`;
      } else if (st === 'found' && issue.fixHint) {
        const label = issue.fixLabel ? escapeHtml(issue.fixLabel) : `Fix · <code>${escapeHtml(issue.fixHint)}</code>`;
        action = `<button class="btn btn-sm btn-primary${flash}" data-cmd="${escapeAttr(issue.fixHint)}">${label}</button>`;
      } else if (st === 'found') {
        action = `<span class="issue-wait">clearing on its own…</span>`;
      }
      // Once investigated, swap the curiosity blurb for the "here's what witr
      // found — now do X" message, so the card doesn't read stale next to a
      // freshly-appeared fix button.
      const bodyMsg = st === 'resolved'
        ? (issue.done || (issue.autoResolve && issue.autoResolve.done) || '')
        : (st === 'found' && issue.foundBlurb ? issue.foundBlurb : issue.blurb);
      return `<div class="issue ${st} sev-${issue.severity}">
        <div class="issue-top"><span class="issue-ic">${icon}</span><span class="issue-title">${issue.title}</span></div>
        <div class="issue-blurb${st === 'resolved' ? ' done' : ''}">${bodyMsg}</div>
        ${action ? `<div class="issue-act">${action}</div>` : ''}
      </div>`;
    }).join('');

    panel.innerHTML = `
      <div class="tut-head">
        <span class="tut-kicker ${done ? 'ok' : 'alert'}">${done ? '● All clear' : '● Incident · ' + escapeHtml(this.currentWorld().hostname)}</span>
        <button class="tut-skip" data-freeplay>Free play →</button>
      </div>
      <div class="health"><div class="health-bar"><span style="width:${(resolved / total) * 100}%"></span></div>
        <span class="health-n">${resolved} / ${total} resolved</span></div>
      <div class="issues">${rows}</div>
      ${this.toolkitHtml()}
      ${done ? `<div class="tut-actions"><button class="btn btn-primary" data-freeplay>Explore freely →</button><button class="btn" data-replay>Replay incident</button></div>` : ''}`;

    panel.querySelectorAll('[data-cmd]').forEach((b) =>
      b.addEventListener('click', () => { if (!this.term.locked) this.term.typeAndRun(b.dataset.cmd); }));
    panel.querySelectorAll('[data-freeplay]').forEach((b) =>
      b.addEventListener('click', () => this.exitToFreePlay()));
    const rp = panel.querySelector('[data-replay]');
    if (rp) rp.addEventListener('click', () => this.resetScenario());

    // When the next actionable task changes (one just completed), scroll it into
    // view so its flashing button is never left below the fold. Re-query inside
    // the rAF: several synchronous re-renders can rebuild the panel first, so a
    // button captured now would already be detached.
    if (nextId && nextId !== this._lastFlashId) {
      requestAnimationFrame(() => {
        const p = document.getElementById('tutorial');
        const fb = p && p.querySelector('.btn.flash');
        if (!fb) return;
        const pr = p.getBoundingClientRect();
        const br = fb.getBoundingClientRect();
        if (br.bottom > pr.bottom - 2 || br.top < pr.top + 2) {
          const card = fb.closest('.issue') || fb;
          p.scrollTop += card.getBoundingClientRect().top - pr.top - 12;
        }
      });
    }
    this._lastFlashId = nextId;

    // The moment the box goes green, glide it to the "Explore" section so the
    // toolkit and "Explore freely" call-to-action are what the visitor lands on.
    if (done && !this._incidentWasDone) {
      requestAnimationFrame(() => {
        const p = document.getElementById('tutorial');
        const target = p && (p.querySelector('.toolkit') || p.querySelector('.tut-actions'));
        if (target) p.scrollTop += target.getBoundingClientRect().top - p.getBoundingClientRect().top - 12;
      });
    }
    this._incidentWasDone = done;
  }

  // Leaving the tutorial hands the box over for free exploration: stop the
  // incident, wipe the screen so the story doesn't linger, and drop a short
  // welcome so free play never starts on a blank void.
  exitToFreePlay() {
    track('free-play-entered');
    if (this._autoTimers) { for (const id of Object.keys(this._autoTimers)) clearTimeout(this._autoTimers[id]); this._autoTimers = {}; }
    this.incident.stop();
    this.term.clear();
    this.welcome();
    // Reset the process views to their initial, unfiltered state.
    this.viewClear();
    this.term.setPrompt(this.shell.prompt());
    this.term.focus();
  }

  // The "Explore witr" toolkit — visible throughout the incident so the tool's
  // breadth (every output mode) is front and centre, not hidden until the end.
  toolkitHtml() {
    const quests = this.incident.sideQuests();
    if (!quests.length) return '';
    const rows = quests.map((q) => {
      const on = this.incident.tried.has(q.id);
      return `<button class="tk-row${on ? ' done' : ''}" data-cmd="${escapeAttr(q.cmd)}" title="${escapeAttr(q.cmd)}">` +
        `<span class="tk-ic">${on ? '✓' : '○'}</span>` +
        `<span class="tk-cmd"><code>${escapeHtml(q.cmd)}</code></span>` +
        `<span class="tk-label">${q.label}</span></button>`;
    }).join('');
    return `<div class="toolkit">
      <div class="tk-head"><span>Explore witr’s modes</span><span class="tk-count">${this.incident.tried.size} / ${quests.length}</span></div>
      <div class="tk-list">${rows}</div>
    </div>`;
  }

  onQuestTried(q) {
    if (q && q.note) this.term.printHtml(`<div class="learned"><span class="learned-badge">witr</span> ${q.note}</div>`);
  }

  welcome() {
    const w = this.currentWorld();
    const hints = this.worldId === 'devbox'
      ? 'Try <code>witr code</code>, inspect the <code>witr --port 5173</code> dev server, reap the <code>witr --pid 6120</code> zombie, or open the <code>witr</code> dashboard.'
      : 'Try <code>witr node</code>, see what’s on <code>witr --port 5000</code>, trace <code>witr nginx</code>, or open the <code>witr</code> dashboard.';
    this.term.printHtml(`<div class="welcome">
      <div class="welcome-logo">witr <span>· why is this running?</span></div>
      <div class="welcome-sub">Free play on <b>${escapeHtml(w.promptUser)}@${escapeHtml(w.hostname)}</b> — a <span class="sim-badge">simulated</span> ${escapeHtml(w.distro)}. Nothing here touches your real computer; it’s a recreation, so the real witr may look and behave slightly differently.</div>
      <div class="welcome-hint">${hints} Explore with <code>ls</code> / <code>ps</code>, and type <code>help</code> anytime.</div>
    </div>`);
  }

  // ---- completion -------------------------------------------------------

  complete(value) {
    const tokens = value.split(' ');
    const last = tokens[tokens.length - 1];
    if (tokens.length <= 1) {
      const hits = COMPLETIONS.filter((c) => c.startsWith(last));
      if (hits.length === 1) return hits[0] + ' ';
      return { value, hints: hits };
    }
    if (tokens[0] === 'witr') {
      let pool;
      if (last.startsWith('-')) pool = WITR_FLAGS.filter((f) => f.startsWith(last));
      else pool = this.currentWorld().processes.map((p) => p.command).filter((c, i, a) => a.indexOf(c) === i).filter((c) => c.startsWith(last));
      if (pool.length === 1) { tokens[tokens.length - 1] = pool[0]; return tokens.join(' ') + ' '; }
      if (pool.length > 1) {
        const pre = commonPrefix(pool);
        if (pre.length > last.length) { tokens[tokens.length - 1] = pre; return { value: tokens.join(' '), hints: pool }; }
        return { value, hints: pool };
      }
    }
    return null;
  }

  // ---- chrome -----------------------------------------------------------

  wireChrome() {
    this.setupTheme();
    document.getElementById('btn-tui').addEventListener('click', () => this.openTui());
    document.getElementById('btn-tutorial').addEventListener('click', () => {
      if (this.incident.active) this.exitToFreePlay();
      else this.startTutorial();
    });
    document.getElementById('btn-scenario').addEventListener('click', () => this.openScenario());
    document.getElementById('btn-reset').addEventListener('click', () => this.resetScenario());
    const modal = document.getElementById('scenario-modal');
    modal.addEventListener('click', (e) => { if (e.target === modal) modal.classList.remove('open'); });
    document.querySelectorAll('[data-scenario]').forEach((b) =>
      b.addEventListener('click', () => this.switchWorld(b.dataset.scenario)));

    // Mobile ☰ menu (the wrapper is display:contents on desktop, so this only
    // ever does anything on small screens).
    const menu = document.getElementById('top-menu');
    const menuBtn = document.getElementById('btn-menu');
    menuBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      const open = menu.classList.toggle('open');
      menuBtn.setAttribute('aria-expanded', String(open));
    });
    menu.querySelectorAll('.btn').forEach((b) =>
      b.addEventListener('click', () => { menu.classList.remove('open'); menuBtn.setAttribute('aria-expanded', 'false'); }));
    document.addEventListener('click', (e) => {
      if (!e.target.closest('#top-menu') && !e.target.closest('#btn-menu')) {
        menu.classList.remove('open');
        menuBtn.setAttribute('aria-expanded', 'false');
      }
    });

    // Install popup.
    const install = document.getElementById('install-modal');
    document.getElementById('btn-install').addEventListener('click', () => { track('install-modal-opened'); install.classList.add('open'); });
    document.querySelectorAll('a.brand-mark, a.btn-star').forEach((a) =>
      a.addEventListener('click', () => track(a.classList.contains('btn-star') ? 'outbound-star' : 'outbound-repo')));
    install.addEventListener('click', (e) => { if (e.target === install) install.classList.remove('open'); });
    document.querySelector('[data-close-install]').addEventListener('click', () => install.classList.remove('open'));
    install.querySelectorAll('[data-copy]').forEach((b) =>
      b.addEventListener('click', () => this.copyToClipboard(b.dataset.copy, b)));

    document.getElementById('chips').addEventListener('click', (e) => {
      const chip = e.target.closest('[data-cmd]');
      if (chip && !this.term.locked) this.term.typeAndRun(chip.dataset.cmd);
    });
  }

  // Copy text to the clipboard and flash a brief confirmation on the button.
  copyToClipboard(text, btn) {
    track('install-copied');
    const done = () => {
      if (!btn) return;
      btn.classList.add('copied');
      const icon = btn.querySelector('.io-copyicon, .copy-icon');
      const prev = icon ? icon.textContent : null;
      if (icon) icon.textContent = '✓';
      setTimeout(() => { btn.classList.remove('copied'); if (icon && prev != null) icon.textContent = prev; }, 1400);
    };
    try {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done, () => this._fallbackCopy(text, done));
      } else { this._fallbackCopy(text, done); }
    } catch (_) { this._fallbackCopy(text, done); }
  }

  _fallbackCopy(text, done) {
    try {
      const ta = document.createElement('textarea');
      ta.value = text; ta.style.position = 'fixed'; ta.style.opacity = '0';
      document.body.appendChild(ta); ta.select();
      document.execCommand('copy'); document.body.removeChild(ta);
      done();
    } catch (_) { /* clipboard unavailable */ }
  }

  // ---- theme ------------------------------------------------------------

  setupTheme() {
    let saved = null;
    try { saved = localStorage.getItem('witr-theme'); } catch (_) {}
    if (saved === 'light' || saved === 'dark') document.documentElement.setAttribute('data-theme', saved);
    this.updateThemeIcon();
    this.map.applyTheme(this.effectiveTheme());
    document.getElementById('btn-theme').addEventListener('click', () => this.toggleTheme());
  }

  effectiveTheme() {
    // Light by default; dark only when the user has explicitly toggled it. The
    // OS colour-scheme preference is intentionally ignored.
    const attr = document.documentElement.getAttribute('data-theme');
    return attr === 'dark' ? 'dark' : 'light';
  }

  toggleTheme() {
    const next = this.effectiveTheme() === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    try { localStorage.setItem('witr-theme', next); } catch (_) {}
    this.updateThemeIcon();
    this.map.applyTheme(next);
  }

  updateThemeIcon() {
    document.getElementById('btn-theme').textContent = this.effectiveTheme() === 'dark' ? '🌙' : '☀️';
  }

  openTui() {
    track('tui-opened');
    this.tui.show(this.currentWorld(), this.shell.engine, this.shell.version);
    this.incident.observe({ targets: [], flags: {}, action: 'tui', world: this.currentWorld() });
    if (this.incident.phase === 'done') this.refreshQuests();
  }

  // A kill issued from inside the TUI removes the process (and its subtree) from
  // the shared world, then syncs every other view and the incident tracker.
  killFromTui(pid) {
    const killed = this.shell.killProcesses([pid]);
    for (const p of killed) this.map.removeProcess(p.pid);
    this.tree.setWorld(this.currentWorld());
    this.refreshHostChip();
    this.incident.observe({ targets: [{ type: 'pid', value: String(pid) }], flags: {}, action: 'kill', world: this.currentWorld() });
    if (this.incident.phase === 'done') this.refreshQuests();
  }

  openScenario() { document.getElementById('scenario-modal').classList.add('open'); }

  // Reset the current scenario to its pristine state (restores killed procs).
  // Restore the pristine world for the current scenario without touching the
  // mode — both reset paths share this.
  restoreWorld() {
    for (const id of Object.keys(this._autoTimers)) clearTimeout(this._autoTimers[id]);
    this._autoTimers = {};
    this.live = cloneWorld(this.pristine[this.worldId]);
    this.shell.setWorld(this.live);
    this.viewSetWorld(this.live);
    this.map.resize();
    this.applyWorld();
  }

  startTutorial() {
    this.restoreWorld();
    this.term.clear();
    this.enterScenario(false);
    this.term.setPrompt(this.shell.prompt());
    this.term.focus();
  }

  // Reset keeps the current mode and scenario intact: mid-tutorial it restarts
  // the incident; in free play it hands back a pristine box, still in free play.
  resetScenario() {
    if (this.incident.active) this.startTutorial();
    else { this.restoreWorld(); this.exitToFreePlay(); }
  }

  switchWorld(id) {
    if (!this.pristine[id]) return;
    track(`scenario-switched-${id}`);
    this.worldId = id;
    document.getElementById('scenario-modal').classList.remove('open');
    this.resetScenario();
  }

  currentWorld() { return this.live; }

  refreshHostChip() {
    const w = this.currentWorld();
    document.getElementById('host-distro').textContent = `${w.distro} · ${w.processes.length} procs`;
  }

  applyWorld() {
    const w = this.currentWorld();
    document.getElementById('host-name').textContent = `${w.promptUser}@${w.hostname}`;
    document.getElementById('term-title').textContent = `${w.promptUser}@${w.hostname}: ~`;
    this.refreshHostChip();
    this.term.setPrompt(this.shell.prompt());
    this.renderIncident();
    const chips = this.worldId === 'webbox'
      ? ['witr --port 8000', 'witr --file /var/lib/dpkg/lock', 'witr node --verbose', 'witr ng', 'witr']
      : ['witr --file /home/pranshu/projects/shop/.git/index.lock', 'witr --pid 6120', 'top', 'witr --pid 6001', 'witr'];
    document.getElementById('chips').innerHTML = chips.map((c) => `<button class="chip" data-cmd="${escapeAttr(c)}">${escapeHtml(c)}</button>`).join('');
  }
}

function cloneWorld(w) {
  return typeof structuredClone === 'function' ? structuredClone(w) : JSON.parse(JSON.stringify(w));
}
function dimNote(s) { return `\x1b[90m${s}\x1b[0m\n`; }
function commonPrefix(arr) {
  if (arr.length === 0) return '';
  let p = arr[0];
  for (const s of arr) { while (!s.startsWith(p)) p = p.slice(0, -1); }
  return p;
}
function escapeHtml(s) { return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'); }
function escapeAttr(s) { return escapeHtml(s).replace(/"/g, '&quot;'); }

new App().boot().catch((e) => {
  document.getElementById('terminal').textContent = 'Failed to load playground: ' + e.message;
  // eslint-disable-next-line no-console
  console.error(e);
});
