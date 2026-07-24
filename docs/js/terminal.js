// terminal.js — a small dependency-free terminal widget.
//
// Renders witr's ANSI output as HTML and handles line editing, history, and
// tab-completion. A hidden <input> backs the buffer so paste, IME, and mobile
// keyboards all work; we render the prompt, buffer, and cursor ourselves.

import { ansiToHtml } from './ansi.js';

export class Terminal {
  constructor(root) {
    this.root = root;
    this.history = [];
    this.histIdx = -1;
    this.draft = '';
    this.onSubmit = null;
    this.completer = null;
    this.promptObj = { user: 'you', host: 'witr', dir: '~' };
    this._locked = false;
    // Stick-to-bottom: new output only auto-scrolls while the reader is already
    // near the bottom. Scroll up (or jump to the top after the cold open) and
    // incoming lines stay put so you can read at your own pace.
    this._stick = true;

    root.classList.add('term');
    this.output = document.createElement('div');
    this.output.className = 'term-output';
    this.line = document.createElement('div');
    this.line.className = 'term-line';
    this.input = document.createElement('input');
    this.input.className = 'term-hidden-input';
    this.input.setAttribute('autocomplete', 'off');
    this.input.setAttribute('autocapitalize', 'off');
    this.input.setAttribute('autocorrect', 'off');
    this.input.setAttribute('spellcheck', 'false');
    this.input.setAttribute('aria-label', 'terminal input');

    root.appendChild(this.output);
    root.appendChild(this.line);
    root.appendChild(this.input);

    this._wire();
    this.renderLine();
  }

  _wire() {
    // Focus on click (fires after mouseup, so text selection still works).
    this.root.addEventListener('click', () => {
      if (window.getSelection().toString() === '') this.focus();
    });
    this.input.addEventListener('input', () => this.renderLine());
    this.input.addEventListener('keydown', (e) => this._onKey(e));
    // Track whether the reader is parked near the bottom (so we may auto-follow)
    // or has scrolled up to read (so we leave the view alone).
    this.root.addEventListener('scroll', () => {
      const gap = this.root.scrollHeight - this.root.scrollTop - this.root.clientHeight;
      this._stick = gap < 44;
    });
    document.addEventListener('selectionchange', () => {
      if (document.activeElement === this.input) this.renderLine();
    });
  }

  // Locking hides the live prompt line entirely — otherwise an empty
  // "user@host:~$" sits below scripted output (e.g. during the cold open),
  // reading as a duplicate prompt.
  get locked() { return this._locked; }
  set locked(v) { this._locked = !!v; this.renderLine(); }

  focus() { if (!this.locked) this.input.focus({ preventScroll: true }); }

  _onKey(e) {
    if (this.locked) { e.preventDefault(); return; }
    if (e.key === 'Enter') {
      e.preventDefault();
      this.submit();
    } else if (e.key === 'Tab') {
      e.preventDefault();
      this._complete();
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      this._historyPrev();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      this._historyNext();
    } else if (e.ctrlKey && (e.key === 'c' || e.key === 'C')) {
      e.preventDefault();
      this.echoInput('^C');
      this.setValue('');
    } else if (e.ctrlKey && (e.key === 'l' || e.key === 'L')) {
      e.preventDefault();
      this.clear();
    } else if (e.ctrlKey && (e.key === 'u' || e.key === 'U')) {
      e.preventDefault();
      this.setValue('');
    }
    // Cursor movement / edits fall through to 'input'.
    requestAnimationFrame(() => this.renderLine());
  }

  get value() { return this.input.value; }
  setValue(v) { this.input.value = v; this.renderLine(); }

  _historyPrev() {
    if (this.history.length === 0) return;
    if (this.histIdx === -1) this.draft = this.value;
    this.histIdx = this.histIdx === -1 ? this.history.length - 1 : Math.max(0, this.histIdx - 1);
    this.setValue(this.history[this.histIdx]);
    this.input.setSelectionRange(this.value.length, this.value.length);
  }

  _historyNext() {
    if (this.histIdx === -1) return;
    this.histIdx++;
    if (this.histIdx >= this.history.length) {
      this.histIdx = -1;
      this.setValue(this.draft);
    } else {
      this.setValue(this.history[this.histIdx]);
    }
    this.input.setSelectionRange(this.value.length, this.value.length);
  }

  _complete() {
    if (!this.completer) return;
    const res = this.completer(this.value);
    if (!res) return;
    if (typeof res === 'string') { this.setValue(res); this.input.setSelectionRange(res.length, res.length); return; }
    if (res.value !== undefined) { this.setValue(res.value); this.input.setSelectionRange(res.value.length, res.value.length); }
    if (res.hints && res.hints.length > 1) {
      this.echoInput(this.value);
      this.printHtml(`<span class="a-dim">${res.hints.map(escapeHtml).join('   ')}</span>`);
    }
  }

  setPrompt(p) { this.promptObj = p; this.renderLine(); }

  promptHtml() {
    const { user, host, dir } = this.promptObj;
    return `<span class="p-user">${escapeHtml(user)}@${escapeHtml(host)}</span>` +
      `<span class="p-sep">:</span><span class="p-dir">${escapeHtml(dir)}</span><span class="p-sep">$</span> `;
  }

  renderLine() {
    if (this._locked) { this.line.innerHTML = ''; this.line.style.display = 'none'; return; }
    this.line.style.display = '';
    const val = this.input.value;
    let pos = this.input.selectionStart;
    if (pos == null) pos = val.length;
    const before = escapeHtml(val.slice(0, pos));
    const at = val.slice(pos, pos + 1);
    const after = escapeHtml(val.slice(pos + 1));
    const cursorChar = at === '' ? '&nbsp;' : escapeHtml(at);
    const focused = document.activeElement === this.input ? ' focused' : '';
    this.line.innerHTML = `${this.promptHtml()}<span class="term-typed">${before}</span>` +
      `<span class="term-cursor${focused}">${cursorChar}</span><span class="term-typed">${after}</span>`;
  }

  // Echo the prompt + a command into the scrollback (used on submit & scripted runs).
  echoInput(text) {
    const div = document.createElement('div');
    div.className = 'term-row';
    div.innerHTML = `${this.promptHtml()}<span class="term-typed">${escapeHtml(text)}</span>`;
    this.output.appendChild(div);
    this.scroll();
  }

  submit() {
    const val = this.value;
    this._stick = true;   // a fresh command — follow its output to the bottom
    this.echoInput(val);
    if (val.trim() !== '') {
      this.history.push(val);
      if (this.history.length > 500) this.history.shift();
    }
    this.histIdx = -1;
    this.draft = '';
    this.setValue('');
    if (this.onSubmit) this.onSubmit(val);
  }

  // Print ANSI text as one or more rows.
  print(ansiText) {
    if (ansiText === '') return;
    const html = ansiToHtml(ansiText);
    this.printHtml(html, true);
  }

  printHtml(html, pre = false) {
    const div = document.createElement('div');
    div.className = pre ? 'term-block' : 'term-row';
    div.innerHTML = html;
    this.output.appendChild(div);
    this.scroll();
  }

  clear() {
    this.output.innerHTML = '';
  }

  scroll() {
    if (!this._stick) return;            // reader is scrolled up — don't yank them down
    this.root.scrollTop = this.root.scrollHeight;
  }

  // Force-follow the bottom again (used when the reader issues a command and
  // expects to see its output).
  stickToBottom() {
    this._stick = true;
    this.root.scrollTop = this.root.scrollHeight;
  }

  scrollToTop() {
    this._stick = false;                 // reading from the top; later output won't yank
    this.root.scrollTop = 0;
  }

  // Put a row near the top of the viewport so its output reads from the start.
  revealRowAtTop(row) {
    this._stick = false;
    const delta = row.getBoundingClientRect().top - this.root.getBoundingClientRect().top - 10;
    this.root.scrollTop += delta;
  }

  // Programmatically run a command with a typewriter effect (tutorial helper).
  // `pause` is a beat after the command finishes typing, before it runs, so the
  // reader can register the command before the output appears. `revealTop` parks
  // the command line at the top afterwards so the reader starts at its output.
  typeAndRun(cmd, { speed = 34, pause = 340, revealTop = true } = {}) {
    return new Promise((resolve) => {
      this.locked = true;
      this._stick = true;   // running a command — follow it to the bottom
      this.line.innerHTML = '';
      let i = 0;
      const buf = { v: '' };
      const rowPromptOnly = document.createElement('div');
      rowPromptOnly.className = 'term-row typing-row';
      this.output.appendChild(rowPromptOnly);
      const tick = () => {
        buf.v = cmd.slice(0, i);
        const cursor = i <= cmd.length ? '<span class="term-cursor focused">&nbsp;</span>' : '';
        rowPromptOnly.innerHTML = `${this.promptHtml()}<span class="term-typed">${escapeHtml(buf.v)}</span>${cursor}`;
        this.scroll();
        if (i < cmd.length) { i++; setTimeout(tick, speed); }
        else {
          rowPromptOnly.innerHTML = `${this.promptHtml()}<span class="term-typed">${escapeHtml(cmd)}</span>`;
          if (cmd.trim() !== '') this.history.push(cmd);
          this.histIdx = -1;
          // Beat to let the command register, then run it.
          setTimeout(() => {
            this.locked = false;
            if (this.onSubmit) this.onSubmit(cmd);
            // Rewind so the just-run command sits at the top and its output can
            // be read from the beginning, rather than landing at the bottom.
            if (revealTop) this.revealRowAtTop(rowPromptOnly);
            resolve();
          }, pause);
        }
      };
      setTimeout(tick, 180);
    });
  }
}

function escapeHtml(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
