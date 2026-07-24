// analytics.js — thin, optional wrapper around GoatCounter (goatcounter.com).
//
// The playground must never depend on analytics: when the counter script is
// blocked (adblockers), offline, or absent, every call here is a silent no-op.
// Counts are anonymous and cookieless — nothing is stored in the visitor's
// browser — and the dashboard is public: https://witr.goatcounter.com
//
// count.js loads async, so events fired before it arrives (e.g. the tutorial
// starting on page load) are queued and flushed once it's ready; if it never
// arrives the queue is quietly abandoned. Each event is counted at most once
// per page load: the interesting numbers are per-visit ratios (started vs
// completed), not repeat clicks.

const seen = new Set();
const queue = [];
let retryTimer = null;

function flush() {
  const gc = window.goatcounter;
  if (!gc || typeof gc.count !== 'function') return false;
  while (queue.length) {
    const path = queue.shift();
    try {
      gc.count({ path, event: true });
    } catch (_) {
      // Analytics must never surface an error to the playground.
    }
  }
  return true;
}

export function track(name) {
  if (seen.has(name)) return;
  seen.add(name);
  queue.push(name);
  if (flush() || retryTimer) return;
  let tries = 0;
  retryTimer = setInterval(() => {
    if (flush() || ++tries >= 15) {
      clearInterval(retryTimer);
      retryTimer = null;
    }
  }, 1000);
}
