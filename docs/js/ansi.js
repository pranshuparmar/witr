// ANSI helpers.
//
// The engine reproduces witr's output *including its ANSI escape codes*, so the
// colored output is byte-for-byte identical to the real binary. The terminal
// renderer then converts those escape codes into styled HTML spans. Keeping the
// escapes in the engine (rather than emitting HTML directly) is what lets the
// fixture check diff engine output against the real `witr` binary.

// The exact SGR codes witr uses (internal/output/colors.go). Bright variants
// (90-97) are used for contrast on dark themes.
export const ESC = {
  reset: '\x1b[0m',
  red: '\x1b[91m',
  green: '\x1b[92m',
  blue: '\x1b[94m',
  cyan: '\x1b[96m',
  magenta: '\x1b[95m',
  dim: '\x1b[90m',
  dimYellow: '\x1b[93m',
};

// Map each SGR code witr emits to a CSS class consumed by styles.css.
const CODE_CLASS = {
  '91': 'a-red',
  '92': 'a-green',
  '94': 'a-blue',
  '96': 'a-cyan',
  '95': 'a-magenta',
  '90': 'a-dim',
  '93': 'a-dimyellow',
};

function escapeHtml(s) {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

// Convert a string containing witr's ANSI escapes into HTML. witr always resets
// after each colored token, so a single "current class" model is sufficient —
// no nesting stack required.
export function ansiToHtml(input) {
  const re = /\x1b\[([0-9;]*)m/g;
  let html = '';
  let last = 0;
  let openClass = null;
  let m;

  const closeSpan = () => {
    if (openClass) {
      html += '</span>';
      openClass = null;
    }
  };

  while ((m = re.exec(input)) !== null) {
    html += escapeHtml(input.slice(last, m.index));
    last = re.lastIndex;
    const code = m[1] === '' ? '0' : m[1];
    if (code === '0') {
      closeSpan();
    } else if (CODE_CLASS[code]) {
      closeSpan();
      openClass = CODE_CLASS[code];
      html += `<span class="${openClass}">`;
    }
    // Unknown codes are simply dropped.
  }
  html += escapeHtml(input.slice(last));
  closeSpan();
  return html;
}

// Strip ANSI escapes entirely (used for width math and copy-to-clipboard).
export function stripAnsi(input) {
  return input.replace(/\x1b\[[0-9;]*m/g, '');
}
