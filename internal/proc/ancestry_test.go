package proc

import (
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// withProcessReader swaps the production processReader for the duration of t.
func withProcessReader(t *testing.T, fn func(pid int) (model.Process, error)) {
	t.Helper()
	orig := processReader
	processReader = fn
	t.Cleanup(func() { processReader = orig })
}

func mkProc(pid, ppid int, startedAt time.Time) model.Process {
	return model.Process{
		PID:       pid,
		PPID:      ppid,
		Command:   "stub",
		StartedAt: startedAt,
	}
}

func TestResolveAncestry_RecycledPPIDStopsWalk(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	withProcessReader(t, func(pid int) (model.Process, error) {
		switch pid {
		case 100: // self
			return mkProc(100, 50, t0), nil
		case 50: // original parent, has the right start time
			return mkProc(50, 30, t0.Add(-1*time.Hour)), nil
		case 30: // recycles the PID slot: started AFTER its child (100)
			return mkProc(30, 10, t0.Add(1*time.Minute)), nil
		case 10: // would be reached if the check were missing
			return mkProc(10, 1, t0.Add(-2*time.Hour)), nil
		}
		return model.Process{}, errNotFound
	})

	chain, err := ResolveAncestry(100)
	if err != nil {
		t.Fatalf("ResolveAncestry: %v", err)
	}

	// Chain is reversed: root -> self. Expected: 50, 100 — and NOT 30 or 10.
	if len(chain) != 2 {
		t.Fatalf("chain length = %d; want 2 (walked past recycled PID)", len(chain))
	}
	if chain[0].PID != 50 {
		t.Errorf("chain[0].PID = %d; want 50 (real parent)", chain[0].PID)
	}
	if chain[1].PID != 100 {
		t.Errorf("chain[1].PID = %d; want 100 (self)", chain[1].PID)
	}
}

func TestResolveAncestry_EqualTimestampsRemainValid(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	withProcessReader(t, func(pid int) (model.Process, error) {
		switch pid {
		case 100:
			return mkProc(100, 50, t0), nil
		case 50:
			return mkProc(50, 30, t0), nil // same resolution as child
		case 30:
			return mkProc(30, 1, t0), nil
		case 1:
			return mkProc(1, 0, t0), nil
		}
		return model.Process{}, errNotFound
	})

	chain, err := ResolveAncestry(100)
	if err != nil {
		t.Fatalf("ResolveAncestry: %v", err)
	}
	if len(chain) != 4 {
		t.Fatalf("chain length = %d; want 4 (equal timestamps must be accepted)", len(chain))
	}
}

func TestResolveAncestry_MissingStartTimeFallsThrough(t *testing.T) {
	// If ReadProcess cannot determine StartedAt on a platform (e.g. macOS
	// stub path), the check is skipped so the walk still progresses.
	withProcessReader(t, func(pid int) (model.Process, error) {
		switch pid {
		case 100:
			return mkProc(100, 50, time.Time{}), nil // unknown
		case 50:
			return mkProc(50, 30, time.Time{}), nil // unknown
		case 30:
			return mkProc(30, 0, time.Time{}), nil
		}
		return model.Process{}, errNotFound
	})

	chain, err := ResolveAncestry(100)
	if err != nil {
		t.Fatalf("ResolveAncestry: %v", err)
	}
	if len(chain) != 3 {
		t.Fatalf("chain length = %d; want 3 (no timestamp, walk continues)", len(chain))
	}
}

// errNotFound is what ReadProcess returns for a missing PID. Tests use it to
// short-circuit the walk when their table runs out of entries.
var errNotFound = &procErr{"not found"}

type procErr struct{ msg string }

func (e *procErr) Error() string { return e.msg }
