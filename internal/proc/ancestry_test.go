package proc

import (
	"fmt"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func processMapReader(processes map[int]model.Process) func(int) (model.Process, error) {
	return func(pid int) (model.Process, error) {
		process, ok := processes[pid]
		if !ok {
			return model.Process{}, fmt.Errorf("process %d not found", pid)
		}
		return process, nil
	}
}

func TestResolveAncestryOrdersParentChainFromRoot(t *testing.T) {
	now := time.Now()
	processes := map[int]model.Process{
		1:   {PID: 1, Command: "init", StartedAt: now.Add(-3 * time.Hour)},
		100: {PID: 100, PPID: 1, Command: "shell", StartedAt: now.Add(-2 * time.Hour)},
		200: {PID: 200, PPID: 100, Command: "worker", StartedAt: now.Add(-time.Hour)},
	}

	chain, err := resolveAncestry(200, processMapReader(processes))
	if err != nil {
		t.Fatalf("resolveAncestry: %v", err)
	}

	want := []int{1, 100, 200}
	if len(chain) != len(want) {
		t.Fatalf("chain length = %d, want %d: %+v", len(chain), len(want), chain)
	}
	for i, pid := range want {
		if chain[i].PID != pid {
			t.Errorf("chain[%d].PID = %d, want %d", i, chain[i].PID, pid)
		}
	}
}

func TestResolveAncestryStopsBeforeRecycledParentPID(t *testing.T) {
	now := time.Now()
	processes := map[int]model.Process{
		// PID 100 now belongs to an unrelated process that started after its
		// supposed child. It must not be included in the returned chain.
		100: {PID: 100, PPID: 1, Command: "unrelated-shell", StartedAt: now.Add(-30 * time.Minute)},
		200: {PID: 200, PPID: 100, Command: "parent", StartedAt: now.Add(-2 * time.Hour)},
		300: {PID: 300, PPID: 200, Command: "worker", StartedAt: now.Add(-time.Hour)},
	}

	chain, err := resolveAncestry(300, processMapReader(processes))
	if err != nil {
		t.Fatalf("resolveAncestry: %v", err)
	}

	want := []int{200, 300}
	if len(chain) != len(want) {
		t.Fatalf("chain length = %d, want %d: %+v", len(chain), len(want), chain)
	}
	for i, pid := range want {
		if chain[i].PID != pid {
			t.Errorf("chain[%d].PID = %d, want %d", i, chain[i].PID, pid)
		}
	}
}

func TestResolveAncestryKeepsParentsWithUnknownStartTimes(t *testing.T) {
	processes := map[int]model.Process{
		1:   {PID: 1, Command: "init"},
		200: {PID: 200, PPID: 1, Command: "worker"},
	}

	chain, err := resolveAncestry(200, processMapReader(processes))
	if err != nil {
		t.Fatalf("resolveAncestry: %v", err)
	}
	if len(chain) != 2 || chain[0].PID != 1 || chain[1].PID != 200 {
		t.Fatalf("chain = %+v, want PIDs [1 200]", chain)
	}
}

func TestResolveAncestryKeepsEqualStartTimes(t *testing.T) {
	startedAt := time.Now()
	processes := map[int]model.Process{
		1:   {PID: 1, Command: "init", StartedAt: startedAt.Add(-time.Hour)},
		100: {PID: 100, PPID: 1, Command: "fast-parent", StartedAt: startedAt},
		200: {PID: 200, PPID: 100, Command: "fast-child", StartedAt: startedAt},
	}

	chain, err := resolveAncestry(200, processMapReader(processes))
	if err != nil {
		t.Fatalf("resolveAncestry: %v", err)
	}
	if len(chain) != 3 || chain[0].PID != 1 || chain[1].PID != 100 || chain[2].PID != 200 {
		t.Fatalf("chain = %+v, want PIDs [1 100 200]", chain)
	}
}
