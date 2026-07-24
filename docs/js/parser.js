// parser.js — parse a `witr ...` command line into { targets, flags }.
//
// Mirrors witr's CLI surface (internal/app/app.go flag set). Targets preserve
// the order they were typed, matching witr's "results appear in the order you
// typed them" behaviour. Value flags accept comma-separated lists (pflag
// StringSlice) and are repeatable.

const BOOL_FLAGS = {
  '--short': 'short', '-s': 'short',
  '--tree': 'tree', '-t': 'tree',
  '--json': 'json',
  '--env': 'env',
  '--warnings': 'warnings',
  '--verbose': 'verbose',
  '--exact': 'exact', '-x': 'exact',
  '--no-color': 'noColor',
  '--interactive': 'interactive', '-i': 'interactive',
  '--help': 'help', '-h': 'help',
  '--version': 'version', '-v': 'version',
};

const VALUE_FLAGS = {
  '--pid': 'pid', '-p': 'pid',
  '--port': 'port', '-o': 'port',
  '--file': 'file', '-f': 'file',
  '--container': 'container', '-c': 'container',
};

// Tokenize respecting single/double quotes.
export function tokenize(line) {
  const tokens = [];
  let cur = '';
  let quote = null;
  let has = false;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (quote) {
      if (ch === quote) quote = null;
      else cur += ch;
    } else if (ch === '"' || ch === "'") {
      quote = ch;
      has = true;
    } else if (ch === ' ' || ch === '\t') {
      if (has) { tokens.push(cur); cur = ''; has = false; }
    } else {
      cur += ch;
      has = true;
    }
  }
  if (has) tokens.push(cur);
  return tokens;
}

export function parse(tokens) {
  const flags = {
    short: false, tree: false, json: false, env: false, warnings: false,
    verbose: false, exact: false, noColor: false, interactive: false,
    help: false, version: false, color: true,
  };
  const targets = [];
  const errors = [];

  const pushTargets = (type, raw) => {
    for (const v of String(raw).split(',')) {
      const val = v.trim();
      if (val !== '') targets.push({ type, value: val });
    }
  };

  for (let i = 0; i < tokens.length; i++) {
    let tok = tokens[i];

    // --flag=value
    if (tok.startsWith('--') && tok.includes('=')) {
      const eq = tok.indexOf('=');
      const name = tok.slice(0, eq);
      const val = tok.slice(eq + 1);
      if (VALUE_FLAGS[name]) { pushTargets(VALUE_FLAGS[name], val); continue; }
      if (BOOL_FLAGS[name]) { flags[BOOL_FLAGS[name]] = true; continue; }
      errors.push(`unknown flag: ${name}`); continue;
    }

    if (BOOL_FLAGS[tok] !== undefined) { flags[BOOL_FLAGS[tok]] = true; continue; }

    if (VALUE_FLAGS[tok] !== undefined) {
      const next = tokens[i + 1];
      if (next === undefined) { errors.push(`flag needs an argument: ${tok}`); break; }
      pushTargets(VALUE_FLAGS[tok], next);
      i++;
      continue;
    }

    // -pVALUE / -p=VALUE (short value flag with attached value)
    if (tok.length > 2 && tok[0] === '-' && tok[1] !== '-' && VALUE_FLAGS[tok.slice(0, 2)]) {
      let val = tok.slice(2);
      if (val.startsWith('=')) val = val.slice(1);
      pushTargets(VALUE_FLAGS[tok.slice(0, 2)], val);
      continue;
    }

    // Grouped short bool flags: -st, -sx, etc.
    if (tok.length > 2 && tok[0] === '-' && tok[1] !== '-' && [...tok.slice(1)].every((c) => BOOL_FLAGS['-' + c])) {
      for (const c of tok.slice(1)) flags[BOOL_FLAGS['-' + c]] = true;
      continue;
    }

    if (tok.startsWith('-') && tok !== '-') { errors.push(`unknown flag: ${tok}`); continue; }

    // Positional → name target.
    pushTargets('name', tok);
  }

  flags.color = !flags.noColor;
  return { targets, flags, errors };
}
