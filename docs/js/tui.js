// tui.js — a faithful, DOM-based rendition of witr's interactive TUI.
//
// It mirrors the real bubbletea dashboard: a purple "witr" brand badge and
// green/grey tabs (Processes / Ports / Containers / Locks), a status + search
// line, a process table beside a live "Details" ancestry pane, and a footer
// with the real key hints and version. Enter opens the Process Detail view —
// the standard witr output beside the process's environment — where `a` opens
// the action menu (kill / term / pause / resume / nice) exactly like the tool.

import { ansiToHtml } from './ansi.js';

const TABS = ['Processes', 'Ports', 'Containers', 'Locks'];
const DISCLAIMER = '<div class="tui-disclaimer">A simulated recreation for the playground — the real witr TUI may look and behave slightly differently.</div>';
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

export class TUI {
  constructor(rootEl) {
    this.root = rootEl;
    this.open = false;
    this.tab = 0;
    this.sel = 0;
    this.filter = '';
    this.filtering = false;
    this.version = '';
    this.onClose = null;
    this.onKill = null;

    // detail state
    this.state = 'list';        // 'list' | 'detail'
    this.detailPid = null;
    this.detailContainer = null;
    this.detailFocus = 'detail'; // 'detail' | 'env'
    this.actionMenuOpen = false;
    this.pendingAction = null;   // 'kill' | 'term' | 'pause' | 'resume' | 'nice'
    this.statusMsg = '';

    this._tick = null;
    this._keyHandler = (e) => this._onKey(e);
    // Click the backdrop (or the disclaimer beneath it) to close. Test the
    // target directly rather than closest('.tui-window'): a tab/row click
    // re-renders innerHTML, so by the time the event bubbles here that element
    // is detached and closest() would wrongly report "outside".
    this.root.addEventListener('click', (e) => {
      if (this.open && (e.target === this.root || e.target.classList.contains('tui-disclaimer'))) this.close();
    });
  }

  show(world, engine, version) {
    this.world = world;
    this.engine = engine;
    if (version) this.version = version;
    this.open = true;
    this.tab = 0;
    this.sel = 0;
    this.filter = '';
    this.filtering = false;
    this.state = 'list';
    this.detailPid = null;
    this.detailContainer = null;
    this.actionMenuOpen = false;
    this.pendingAction = null;
    this.statusMsg = '';
    this.root.classList.add('tui-open');
    this.root.setAttribute('aria-hidden', 'false');
    document.addEventListener('keydown', this._keyHandler, true);
    this.render();
    // Match top's 3s auto-refresh cadence (relative times, live state).
    this._tick = setInterval(() => { if (this.state === 'list') this.render(); }, 3000);
  }

  close() {
    if (!this.open) return;
    this.open = false;
    this.root.classList.remove('tui-open');
    this.root.setAttribute('aria-hidden', 'true');
    document.removeEventListener('keydown', this._keyHandler, true);
    if (this._tick) clearInterval(this._tick);
    if (this.onClose) this.onClose();
  }

  // ---- data ------------------------------------------------------------

  rows() {
    const w = this.world;
    if (this.tab === 0) {
      // Default sort is by resident memory, descending — same as the real TUI.
      let list = [...w.processes].sort((a, b) => rss(b) - rss(a));
      if (this.filter) {
        const f = this.filter.toLowerCase();
        list = list.filter((p) => (p.command + ' ' + (p.cmdline || '') + ' ' + (p.user || '') + ' ' + p.pid).toLowerCase().includes(f));
      }
      return list;
    }
    if (this.tab === 1) {
      const ports = [];
      for (const p of w.processes) for (const s of p.sockets || []) if (s.state === 'LISTEN') ports.push({ p, s });
      return ports.sort((a, b) => a.s.port - b.s.port);
    }
    if (this.tab === 2) {
      let list = [...(w.containers || [])].sort((a, b) => (a.name || '').localeCompare(b.name || ''));
      if (this.filter) { const f = this.filter.toLowerCase(); list = list.filter((c) => (c.name + ' ' + c.image + ' ' + (c.status || '')).toLowerCase().includes(f)); }
      return list;
    }
    let list = w.locks || [];
    if (this.filter) { const f = this.filter.toLowerCase(); list = list.filter((l) => (l.process + ' ' + l.path + ' ' + l.pid).toLowerCase().includes(f)); }
    return list;
  }

  // ---- keys ------------------------------------------------------------

  _onKey(e) {
    if (!this.open) return;
    e.stopPropagation();
    if (this.state === 'detail') { this._onDetailKey(e); return; }

    if (this.filtering) {
      if (e.key === 'Enter' || e.key === 'Escape') { this.filtering = false; e.preventDefault(); this.render(); return; }
      if (e.key === 'Backspace') { this.filter = this.filter.slice(0, -1); e.preventDefault(); this.sel = 0; this.render(); return; }
      if (e.key.length === 1) { this.filter += e.key; e.preventDefault(); this.sel = 0; this.render(); return; }
      return;
    }

    const rows = this.rows();
    switch (e.key) {
      case 'Escape': case 'q': e.preventDefault(); this.close(); break;
      case 'Tab':
        e.preventDefault();
        this.tab = (this.tab + (e.shiftKey ? TABS.length - 1 : 1)) % TABS.length;
        this.sel = 0; this.filter = ''; this.statusMsg = ''; this.render();
        break;
      case '1': case '2': case '3': case '4':
        e.preventDefault();
        this.tab = Math.min(TABS.length - 1, parseInt(e.key, 10) - 1); this.sel = 0; this.filter = ''; this.statusMsg = ''; this.render();
        break;
      case 'ArrowDown': case 'j':
        e.preventDefault(); this.sel = Math.min(rows.length - 1, this.sel + 1); this.render(); break;
      case 'ArrowUp': case 'k':
        e.preventDefault(); this.sel = Math.max(0, this.sel - 1); this.render(); break;
      case 'Enter':
        e.preventDefault(); this._openDetail(rows[this.sel]); break;
      case '/':
        if (this.tab !== 1) { e.preventDefault(); this.filtering = true; this.render(); }
        break;
      default: break;
    }
  }

  _onDetailKey(e) {
    const pid = this.detailPid;
    if (this.pendingAction) {
      if (e.key === 'y' || e.key === 'Y') {
        e.preventDefault(); this._performAction(this.pendingAction, pid); return;
      }
      if (e.key === 'n' || e.key === 'N' || e.key === 'Escape') {
        e.preventDefault(); this.pendingAction = null; this.render(); return;
      }
      return;
    }
    if (this.actionMenuOpen) {
      const map = { k: 'kill', t: 'term', p: 'pause', r: 'resume', n: 'nice' };
      if (map[e.key]) { e.preventDefault(); this.actionMenuOpen = false; this.pendingAction = map[e.key]; this.render(); return; }
      if (e.key === 'Escape' || e.key === 'q') { e.preventDefault(); this.actionMenuOpen = false; this.render(); return; }
      return;
    }
    switch (e.key) {
      case 'Escape': case 'q':
        e.preventDefault(); this.state = 'list'; this.detailPid = null; this.detailContainer = null; this.render(); break;
      case 'a':
        if (this.detailPid != null) { e.preventDefault(); this.actionMenuOpen = true; this.statusMsg = ''; this.render(); }
        break;
      case 'Tab':
        e.preventDefault(); this.detailFocus = this.detailFocus === 'detail' ? 'env' : 'detail'; this.render(); break;
      default: break;
    }
  }

  _openDetail(row) {
    if (!row) return;
    if (this.tab === 0) { this.detailPid = row.pid; this.detailContainer = null; }
    else if (this.tab === 2) { this.detailContainer = row; this.detailPid = null; }
    else if (this.tab === 3) {
      const owner = this.engine.procByPid.get(row.pid);
      if (!owner) return;
      this.detailPid = owner.pid; this.detailContainer = null;
    } else { return; } // Ports: no detail (matches the real footer)
    this.state = 'detail';
    this.detailFocus = 'detail';
    this.actionMenuOpen = false;
    this.pendingAction = null;
    this.statusMsg = '';
    this.render();
  }

  _performAction(action, pid) {
    const proc = this.engine.procByPid.get(pid);
    const name = proc ? proc.command : `pid ${pid}`;
    this.pendingAction = null;
    if (action === 'kill' || action === 'term') {
      if (this.onKill) this.onKill(pid);
      else { this.world.processes = this.world.processes.filter((p) => p.pid !== pid); this.engine.reindex(); }
      // The process is gone — drop back to the list.
      this.state = 'list';
      this.detailPid = null;
      this.sel = Math.max(0, Math.min(this.sel, this.rows().length - 1));
      this.statusMsg = `Sent ${action === 'kill' ? 'SIGKILL' : 'SIGTERM'} to ${name} (pid ${pid})`;
    } else {
      const verb = { pause: 'Paused', resume: 'Resumed', nice: 'Reniced' }[action];
      this.statusMsg = `${verb} ${name} (pid ${pid}) — simulated`;
    }
    this.render();
  }

  // ---- render ----------------------------------------------------------

  render() {
    if (!this.open) return;
    if (this.state === 'detail') { this._renderDetail(); this._wireCommon(); return; }

    const w = this.world;
    const rows = this.rows();
    if (this.sel >= rows.length) this.sel = Math.max(0, rows.length - 1);

    const tabs = TABS.map((t, i) =>
      `<span class="tui-tab${i === this.tab ? ' active' : ''}" data-tab="${i}">${i + 1}. ${t}</span>`).join('');

    let statusClass = '';
    let statusText = 'Mode: Navigation (Press / to search)';
    if (this.statusMsg) { statusText = escapeHtml(this.statusMsg); statusClass = ' err'; }
    else if (this.filtering) { statusText = 'Mode: Searching (↑↓ to navigate, Esc/Enter to stop)'; statusClass = ' searching'; }

    const placeholders = [
      'Search PID, Name, User, Command...',
      'Search Port, Protocol, Address, State...',
      'Search ID, Name, Runtime, Image, Status, Ports,…',
      'Search PID, Process, Type, Mode, Path…',
    ];
    const inputLine = (this.filtering || this.filter)
      ? `<span class="tui-prompt">&gt; </span>${escapeHtml(this.filter)}${this.filtering ? '<span class="tui-caret">▏</span>' : ''}`
      : `<span class="tui-prompt">&gt; </span><span class="tui-muted">${placeholders[this.tab]}</span>`;

    let main;
    if (this.tab === 0) main = this._procsView(rows);
    else if (this.tab === 1) main = this._portsView(rows);
    else if (this.tab === 2) main = this._containersView(rows);
    else main = this._locksView(rows);

    this.root.innerHTML = `
      <div class="tui-window" role="dialog" aria-label="witr interactive dashboard">
        <div class="tui-top">
          <span class="tui-brand" data-home title="Back to processes">witr</span>${tabs}
          <button class="tui-x" data-close title="Close (q)">✕</button>
        </div>
        <div class="tui-spacer"></div>
        <div class="tui-status${statusClass}">${statusText}</div>
        <div class="tui-input">${inputLine}</div>
        <div class="tui-main">${main}</div>
        <div class="tui-foot">${this._footer(rows.length)}</div>
      </div>
      ${DISCLAIMER}`;

    this._wireCommon();
    this.root.querySelectorAll('.tui-tab').forEach((b) =>
      b.addEventListener('click', () => { this.tab = +b.dataset.tab; this.sel = 0; this.filter = ''; this.statusMsg = ''; this.render(); }));
    this.root.querySelectorAll('.tui-r[data-i]').forEach((r) =>
      r.addEventListener('click', () => { this.sel = +r.dataset.i; this.render(); }));
    this.root.querySelectorAll('.tui-r[data-i]').forEach((r) =>
      r.addEventListener('dblclick', () => { this.sel = +r.dataset.i; this._openDetail(this.rows()[this.sel]); }));
  }

  _wireCommon() {
    const x = this.root.querySelector('[data-close]');
    if (x) x.addEventListener('click', () => this.close());
    const home = this.root.querySelector('[data-home]');
    if (home) home.addEventListener('click', () => this._goHome());
  }

  // Clicking the "witr" badge returns to the Processes list from anywhere.
  _goHome() {
    this.state = 'list';
    this.tab = 0;
    this.sel = 0;
    this.filter = '';
    this.filtering = false;
    this.detailPid = null;
    this.detailContainer = null;
    this.actionMenuOpen = false;
    this.pendingAction = null;
    this.statusMsg = '';
    this.render();
  }

  _footer(total) {
    let help;
    switch (this.tab) {
      case 1: help = `Total: ${total} [LISTEN] | p/t/n/s: Sort | a: Toggle All | Esc/q: Quit | Tab: Focus | Up/Down: Scroll`; break;
      case 2: help = `Total: ${total} | Enter: Detail | i/n/r/g/s: Sort | /: Search | Esc/q: Quit | Up/Down: Scroll`; break;
      case 3: help = `Total: ${total} [LOCKED] | Enter: Detail | a: Toggle Open Files | p/n/t/m/f: Sort | /: Search | Esc/q: Quit | Up/Down: Scroll`; break;
      default: help = `Total: ${total} | Enter: Detail | p/n/u/c/m/t: Sort | Esc/q: Quit | Tab: Focus | Up/Down: Scroll`;
    }
    return `<span class="tui-help">${escapeHtml(help)}</span><span class="tui-ver">${escapeHtml(this.version)}</span>`;
  }

  // ---- list views ------------------------------------------------------

  _procsView(rows) {
    const now = this.engine.now();
    const total = this.world.memTotalBytes || 8 * 1024 * 1024 * 1024;
    let table = `<div class="tui-r head"><span class="tui-num">PID</span><span>User</span><span>Name</span>` +
      `<span class="tui-num">CPU%</span><span class="tui-num">Mem ↓</span><span>Started</span><span>Command</span></div>`;
    let body = '';
    rows.forEach((p, i) => {
      const started = fmtStarted(now - (p.startedAgo || 0) * 1000);
      const cpu = `${(p.cpuPercent || 0).toFixed(1)}%`;
      const r = (p.memory && p.memory.rss) || 0;
      const mem = r > 0 ? `${fmtBytes(r)} (${(r / total * 100).toFixed(1)}%)` : '0 B';
      body += `<div class="tui-r${i === this.sel ? ' sel' : ''}" data-i="${i}">` +
        `<span class="tui-num">${p.pid}</span><span>${escapeHtml(p.user || '')}</span><span>${escapeHtml(p.command)}</span>` +
        `<span class="tui-num">${cpu}</span><span class="tui-num">${escapeHtml(mem)}</span>` +
        `<span>${escapeHtml(started)}</span><span>${escapeHtml(p.cmdline || p.command)}</span></div>`;
    });
    return `<div class="tui-pane tui-pane-list"><div class="tui-rows">${table}${body}</div></div>` +
      this._sidePane(rows[this.sel]);
  }

  _sidePane(sel) {
    let head = 'Details';
    let body = '<div class="tui-muted">no process</div>';
    if (sel) {
      head = `PID ${sel.pid}`;
      body = this._treeHtml(sel);
    }
    return `<div class="tui-pane tui-pane-side divider${this.detailFocus === 'side' ? ' focus' : ''}">` +
      `<div class="tui-hdr">${escapeHtml(head)}</div><div class="tui-side-body">${body}</div></div>`;
  }

  _treeHtml(proc) {
    const chain = this.engine.ancestryOf(proc);
    const kids = this.engine.childrenOf(proc.pid);
    let s = `<span class="tui-sec">Ancestry Tree:</span>\n`;
    chain.forEach((p, i) => {
      const indent = '  '.repeat(i);
      const conn = i > 0 ? `${indent}<span class="tui-conn">└─</span> ` : '';
      const label = `${escapeHtml(p.command)} <span class="tui-muted">(pid ${p.pid})</span>`;
      const isTarget = i === chain.length - 1;
      s += `${conn}${isTarget ? `<span class="tui-target">${label}</span>` : label}\n`;
    });
    if (kids.length) {
      const base = '  '.repeat(chain.length);
      const limit = 10;
      kids.slice(0, limit).forEach((c, i) => {
        const last = i === Math.min(kids.length, limit) - 1 && kids.length <= limit;
        const conn = last ? '└─' : '├─';
        s += `${base}<span class="tui-conn">${conn}</span> ${escapeHtml(c.command)} <span class="tui-muted">(pid ${c.pid})</span>\n`;
      });
      if (kids.length > limit) s += `${base}<span class="tui-conn">└─</span> <span class="tui-muted">… and ${kids.length - limit} more</span>\n`;
    }
    if (proc.cmdline) s += `\n<span class="tui-sec">Command:</span>\n${escapeHtml(proc.cmdline)}\n`;
    return s;
  }

  _portsView(rows) {
    let table = `<div class="tui-r t-ports head"><span class="tui-num">Port ↑</span><span>Protocol</span><span>Address</span><span>State</span></div>`;
    let body = '';
    rows.forEach(({ s }, i) => {
      body += `<div class="tui-r t-ports${i === this.sel ? ' sel' : ''}" data-i="${i}">` +
        `<span class="tui-num">${s.port}</span><span>${escapeHtml(s.protocol || 'TCP')}</span>` +
        `<span>${escapeHtml(s.address)}</span><span>${escapeHtml(s.state || 'LISTEN')}</span></div>`;
    });
    if (!rows.length) body = '<div class="tui-empty">no listening ports</div>';
    return `<div class="tui-pane tui-pane-list"><div class="tui-rows">${table}${body}</div></div>` +
      this._portSidePane(rows[this.sel]);
  }

  _portSidePane(sel) {
    let body = '<div class="tui-muted">select a port</div>';
    if (sel) {
      const port = sel.s.port;
      const attached = this.world.processes.filter((p) => (p.sockets || []).some((s) => s.port === port));
      if (attached.length) {
        let t = `<div class="tui-r t-attached head"><span class="tui-num">PID</span><span>User</span><span>Name</span><span>Command</span></div>`;
        attached.forEach((p) => {
          t += `<div class="tui-r t-attached"><span class="tui-num">${p.pid}</span><span>${escapeHtml(p.user || '')}</span>` +
            `<span>${escapeHtml(p.command)}</span><span>${escapeHtml(p.cmdline || p.command)}</span></div>`;
        });
        body = `<div class="tui-rows">${t}</div>`;
      } else body = '<div class="tui-muted">no attached process</div>';
    }
    return `<div class="tui-pane tui-pane-portside divider"><div class="tui-hdr">Attached Processes</div>${body}</div>`;
  }

  _containersView(rows) {
    let table = `<div class="tui-r t-containers head"><span>ID</span><span>Name ↑</span><span>Runtime</span><span>Image</span><span>Status</span><span>Ports</span><span>Command</span></div>`;
    let body = '';
    rows.forEach((c, i) => {
      const id = (c.id || '').slice(0, 12);
      body += `<div class="tui-r t-containers${i === this.sel ? ' sel' : ''}" data-i="${i}">` +
        `<span>${escapeHtml(id)}</span><span>${escapeHtml(c.name)}</span><span>${escapeHtml(c.runtime || 'docker')}</span>` +
        `<span>${escapeHtml(c.image)}</span><span>${escapeHtml(c.status || c.state)}</span><span>${escapeHtml(c.ports || '—')}</span>` +
        `<span>${escapeHtml(c.command || '')}</span></div>`;
    });
    if (!rows.length) body = '<div class="tui-empty">no containers</div>';
    return `<div class="tui-pane tui-pane-list"><div class="tui-rows">${table}${body}</div></div>`;
  }

  _locksView(rows) {
    let table = `<div class="tui-r t-locks head"><span class="tui-num">PID</span><span>Process</span><span>Type</span><span>Mode</span><span>Path</span></div>`;
    let body = '';
    rows.forEach((l, i) => {
      body += `<div class="tui-r t-locks${i === this.sel ? ' sel' : ''}" data-i="${i}">` +
        `<span class="tui-num">${l.pid}</span><span>${escapeHtml(l.process)}</span><span>${escapeHtml(l.type)}</span>` +
        `<span>${escapeHtml(l.mode)}</span><span>${escapeHtml(l.path)}</span></div>`;
    });
    if (!rows.length) body = '<div class="tui-empty">no file locks</div>';
    return `<div class="tui-pane tui-pane-list"><div class="tui-rows">${table}${body}</div></div>`;
  }

  // ---- detail view -----------------------------------------------------

  _renderDetail() {
    const badge = this.detailContainer
      ? `<span class="tui-pidbadge">ID ${escapeHtml((this.detailContainer.id || '').slice(0, 12))}</span>`
      : `<span class="tui-pidbadge">PID ${this.detailPid}</span>`;

    let main;
    if (this.detailContainer) {
      main = `<div class="tui-pane tui-pane-detail"><div class="tui-hdr accent">Container Detail</div>` +
        `<div class="tui-body-scroll"><pre>${this._containerDetailHtml(this.detailContainer)}</pre></div></div>`;
    } else {
      const detailFocused = this.detailFocus === 'detail';
      main =
        `<div class="tui-pane tui-pane-detail"><div class="tui-hdr ${detailFocused ? 'accent' : ''}">Process Detail</div>` +
        `<div class="tui-body-scroll"><pre>${this._processDetailHtml(this.detailPid)}</pre></div></div>` +
        `<div class="tui-pane tui-pane-env divider${detailFocused ? '' : ' focus'}"><div class="tui-hdr ${detailFocused ? '' : 'accent'}">Environment Variables</div>` +
        `<div class="tui-body-scroll"><pre>${this._envHtml(this.detailPid)}</pre></div></div>`;
    }

    this.root.innerHTML = `
      <div class="tui-window" role="dialog" aria-label="witr process detail">
        <div class="tui-top">
          <span class="tui-brand" data-home title="Back to processes">witr</span>${badge}
          <button class="tui-x" data-close title="Back (q)">✕</button>
        </div>
        <div class="tui-spacer"></div>
        <div class="tui-main">${main}</div>
        <div class="tui-foot ${this._detailFootClass()}">${this._detailFooter()}</div>
      </div>
      ${DISCLAIMER}`;
    // Scroll-position arrows (↓/↑/↕) on each pane header, like the real TUI.
    this._updateDetailArrows();
    this.root.querySelectorAll('.tui-body-scroll').forEach((el) =>
      el.addEventListener('scroll', () => this._updateDetailArrows()));
  }

  _updateDetailArrows() {
    this.root.querySelectorAll('.tui-pane').forEach((pane) => {
      const hdr = pane.querySelector('.tui-hdr');
      const sc = pane.querySelector('.tui-body-scroll');
      if (!hdr || !sc) return;
      const base = hdr.dataset.base || hdr.textContent.replace(/ [↓↑↕]$/, '');
      hdr.dataset.base = base;
      let arrow = '';
      if (sc.scrollHeight > sc.clientHeight + 1) {
        const atTop = sc.scrollTop <= 1;
        const atBottom = sc.scrollTop + sc.clientHeight >= sc.scrollHeight - 1;
        arrow = (!atTop && !atBottom) ? ' ↕' : (atTop ? ' ↓' : ' ↑');
      }
      hdr.textContent = base + arrow;
    });
  }

  _detailFootClass() {
    if (this.actionMenuOpen) return 'actions';
    if (this.pendingAction) return 'confirm';
    if (this.statusMsg) return 'err';
    return '';
  }

  _detailFooter() {
    const pid = this.detailPid;
    if (this.actionMenuOpen) return `Esc/q: cancel | Actions:  [k]ill  [t]erm  [p]ause  [r]esume  [n]ice`;
    if (this.pendingAction) {
      const verb = { kill: 'Kill', term: 'Terminate', pause: 'Pause', resume: 'Resume', nice: 'Renice' }[this.pendingAction];
      return `${verb} PID ${pid}? [y]es / [n]o`;
    }
    if (this.statusMsg) return escapeHtml(this.statusMsg);
    const help = this.detailContainer
      ? 'Esc/q: Back | Up/Down: Scroll'
      : 'a: Actions | Esc/q: Back | Tab: Focus | Up/Down: Scroll';
    return `<span class="tui-help">${escapeHtml(help)}</span><span class="tui-ver">${escapeHtml(this.version)}</span>`;
  }

  _processDetailHtml(pid) {
    // The real TUI shows the *verbose*, *colorized* standard output in the pane.
    const res = this.engine.run({ targets: [{ type: 'pid', value: String(pid) }], flags: { verbose: true, color: true } });
    return ansiToHtml((res.text || '').replace(/\n$/, ''));
  }

  _envHtml(pid) {
    const proc = this.engine.procByPid.get(pid);
    const env = (proc && proc.env) || [];
    if (!env.length) return '<span class="tui-muted">No environment variables found.</span>';
    return env.map((e) => escapeHtml(e)).join('\n');
  }

  _containerDetailHtml(c) {
    const lines = [];
    const add = (k, v) => { if (v) lines.push(`<span class="tui-sec">${k}</span>  ${escapeHtml(String(v))}`); };
    add('Name', c.name); add('Image', c.image); add('Runtime', c.runtime); add('State', c.state);
    add('Status', c.status); add('Health', c.health); add('Ports', c.ports); add('Networks', c.networks);
    add('Mounts', c.mounts);
    if (c.composeProject) add('Compose', `${c.composeProject}/${c.composeService}`);
    add('Config', c.composeConfigFile); add('Command', c.command);
    return lines.join('\n');
  }
}

function fmtStarted(ms) {
  if (ms == null) return '';
  const d = new Date(ms);
  const mo = MONTHS[d.getUTCMonth()];
  const da = String(d.getUTCDate()).padStart(2, '0');
  const h = String(d.getUTCHours()).padStart(2, '0');
  const mi = String(d.getUTCMinutes()).padStart(2, '0');
  const s = String(d.getUTCSeconds()).padStart(2, '0');
  return `${mo} ${da} ${h}:${mi}:${s}`;
}

function rss(p) { return (p.memory && p.memory.rss) || 0; }

function fmtBytes(n) {
  const unit = 1024;
  if (n < unit) return `${n} B`;
  let div = unit, exp = 0;
  for (let m = Math.floor(n / unit); m >= unit; m = Math.floor(m / unit)) { div *= unit; exp++; }
  return `${(n / div).toFixed(1)} ${'KMGTPE'[exp]}B`;
}

function escapeHtml(s) {
  return String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
