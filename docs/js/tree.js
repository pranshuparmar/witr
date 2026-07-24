// tree.js — a readable, scrollable process tree for the right-hand view.
//
// An HTML/CSS alternative to the 3D constellation: every process is a labelled
// row, indented by ancestry, the active chain highlighted, click to inspect.
// It shares the SystemMap surface (setWorld / highlightPids / clearHighlight /
// removeProcess / onSelect) so the app can treat the two views interchangeably.

export class Tree {
  constructor(container) {
    this.el = container;
    this.world = null;
    this.onSelect = null;
    this.highlightSet = new Set();
    this.el.addEventListener('click', (e) => {
      const row = e.target.closest('.tree-row');
      if (row && this.onSelect) this.onSelect(parseInt(row.dataset.pid, 10));
    });
  }

  setWorld(world) { this.world = world; this.render(); }
  removeProcess() { this.render(); }   // world already mutated by the caller
  resize() {}                          // no-op; kept for interface parity

  highlightPids(pids) { this.highlightSet = new Set(pids); this._applyHighlight(true); }
  clearHighlight() { this.highlightSet = new Set(); this._applyHighlight(false); }

  render() {
    if (!this.world) return;
    const procs = this.world.processes;
    const byPid = new Map(procs.map((p) => [p.pid, p]));
    const kids = new Map();
    for (const p of procs) {
      if (!kids.has(p.ppid)) kids.set(p.ppid, []);
      kids.get(p.ppid).push(p);
    }
    const roots = procs.filter((p) => !byPid.has(p.ppid)).sort((a, b) => a.pid - b.pid);

    let html = '';
    const walk = (p, depth) => {
      html += this._row(p, depth);
      const cs = (kids.get(p.pid) || []).filter((c) => c.pid !== p.pid).sort((a, b) => a.pid - b.pid);
      for (const c of cs) walk(c, depth + 1);
    };
    for (const r of roots) walk(r, 0);

    this.el.innerHTML = html || '<div class="tree-empty">no processes</div>';
    this._applyHighlight(this.highlightSet.size > 0, false);
  }

  _row(p, depth) {
    const warn = (p.warnings && p.warnings.length) || (p.health && p.health !== 'healthy');
    const listener = (p.sockets || []).some((s) => s.state === 'LISTEN');
    const kind = warn ? 'warn' : (listener ? 'listener' : (depth === 0 ? 'root' : 'proc'));
    const tag = p.health && p.health !== 'healthy' ? ` <span class="tr-tag">[${esc(p.health)}]</span>` : '';
    const guide = depth > 0 ? `<span class="tr-guide">${'│ '.repeat(depth - 1)}└ </span>` : '';
    return `<div class="tree-row k-${kind}" data-pid="${p.pid}" title="${esc(p.cmdline || p.command)}">` +
      `<span class="tr-name">${guide}<span class="tr-dot"></span>${esc(p.command)}${tag}</span>` +
      `<span class="tr-pid">${p.pid}</span>` +
      `<span class="tr-user">${esc(p.user || '')}</span></div>`;
  }

  _applyHighlight(hasHl, scroll = true) {
    this.el.classList.toggle('has-hl', hasHl);
    let firstOn = null;
    for (const row of this.el.querySelectorAll('.tree-row')) {
      const on = hasHl && this.highlightSet.has(parseInt(row.dataset.pid, 10));
      row.classList.toggle('on-chain', on);
      if (on && !firstOn) firstOn = row;
    }
    if (scroll && firstOn) firstOn.scrollIntoView({ block: 'nearest' });
  }
}

function esc(s) {
  return String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
