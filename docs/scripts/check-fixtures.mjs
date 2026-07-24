// check-fixtures.mjs — verify the browser engine reproduces witr's real output.
//
// Fixtures under docs/fixtures/ are generated from witr's actual output
// package (see fixtures/gen). This runs the JS engine over the same world with
// the same pinned clock and asserts byte-for-byte equality — the guarantee that
// the playground never lies about what `witr` prints.
//
//   node docs/scripts/check-fixtures.mjs
//
// Exit 0 on match, 1 on any drift.

import { readFileSync, existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { Engine } from '../js/engine.js';

const here = dirname(fileURLToPath(import.meta.url));
const fixturesDir = join(here, '..', 'fixtures');
const world = JSON.parse(readFileSync(join(here, '..', 'worlds', 'webbox.json'), 'utf8'));
const meta = JSON.parse(readFileSync(join(fixturesDir, '_meta.json'), 'utf8'));

const NOW = meta.generatedAtMs;

function flagsFor(mode, color) {
  const f = {
    short: mode === 'short', tree: mode === 'tree', json: false,
    env: mode === 'env', warnings: mode === 'warnings',
    verbose: mode === 'verbose', exact: false, color,
  };
  return f;
}

function runCase(c, color) {
  const eng = new Engine(world);
  eng.setNow(() => NOW);
  const target = { type: c.kind, value: c.value };
  const { text } = eng.run({ targets: [target], flags: flagsFor(c.mode, color) });
  return text;
}

let failed = 0;
let passed = 0;

for (const c of meta.cases) {
  for (const [color, ext] of [[false, '.txt'], [true, '.ansi']]) {
    const path = join(fixturesDir, c.name + ext);
    if (!existsSync(path)) { console.error(`✗ ${c.name}${ext}: golden file missing`); failed++; continue; }
    const golden = readFileSync(path, 'utf8');
    const got = runCase(c, color);
    if (got === golden) {
      passed++;
    } else {
      failed++;
      console.error(`\n✗ MISMATCH ${c.name}${ext} (${c.mode} ${c.kind} ${c.value})`);
      printDiff(golden, got);
    }
  }
}

console.log(`\n${failed === 0 ? '✓' : '✗'} fixtures: ${passed} passed, ${failed} failed`);
process.exit(failed === 0 ? 0 : 1);

function printDiff(want, got) {
  const w = want.split('\n');
  const g = got.split('\n');
  const n = Math.max(w.length, g.length);
  for (let i = 0; i < n; i++) {
    if (w[i] !== g[i]) {
      console.error(`  line ${i + 1}:`);
      console.error(`    want: ${JSON.stringify(w[i] ?? '<none>')}`);
      console.error(`    got:  ${JSON.stringify(g[i] ?? '<none>')}`);
    }
  }
}
