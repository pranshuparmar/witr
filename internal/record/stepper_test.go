package record

import (
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func createMultiSnapshotSeries() []*model.Snapshot {
	baseTime := time.Date(2026, 10, 5, 14, 0, 0, 0, time.UTC)
	var list []*model.Snapshot

	for i := 0; i < 4; i++ {
		curTime := baseTime.Add(time.Duration(i*5) * time.Second)
		snap := &model.Snapshot{
			Version:   CurrentSnapshotVersion,
			Index:     i,
			Timestamp: curTime,
			Hostname:  "worker-1",
			Processes: []model.Process{
				{
					PID:       1,
					PPID:      0,
					Command:   "systemd",
					Cmdline:   "/sbin/init",
					StartedAt: baseTime.Add(-1 * time.Hour),
				},
				{
					PID:       100,
					PPID:      1,
					Command:   "sshd",
					Cmdline:   "/usr/sbin/sshd -D",
					StartedAt: baseTime.Add(-30 * time.Minute),
				},
				{
					PID:       200 + i,
					PPID:      100,
					Command:   "worker",
					Cmdline:   "/opt/bin/worker --job run",
					StartedAt: curTime.Add(-10 * time.Second),
				},
			},
			Ports: []model.OpenPort{
				{
					PID:      100,
					Port:     22,
					Protocol: "tcp",
					State:    "LISTEN",
				},
			},
			Containers: []*model.ContainerMatch{
				{
					ID:   "box1",
					Name: "redis-cache",
				},
			},
			LockedFiles: []*model.LockedFile{
				{
					PID:  200 + i,
					Path: "/tmp/worker.lock",
					Type: "FLOCK",
				},
			},
		}
		list = append(list, snap)
	}
	return list
}

func TestStepperNavigation(t *testing.T) {
	snaps := createMultiSnapshotSeries()
	s, err := NewStepper(snaps)
	if err != nil {
		t.Fatalf("NewStepper error: %v", err)
	}

	if s.Count() != 4 {
		t.Fatalf("got count %d, want 4", s.Count())
	}
	if s.CurrentIndex() != 0 {
		t.Errorf("initial index = %d, want 0", s.CurrentIndex())
	}

	// Prev at boundary returns false
	if _, ok := s.Prev(); ok {
		t.Error("expected false for Prev() at start")
	}

	// Step forward
	nextSnap, ok := s.Next()
	if !ok || s.CurrentIndex() != 1 || nextSnap.Index != 1 {
		t.Errorf("Next() failed, index=%d, ok=%v", s.CurrentIndex(), ok)
	}

	// StepTo arbitrary
	stSnap, err := s.StepTo(3)
	if err != nil || s.CurrentIndex() != 3 || stSnap.Index != 3 {
		t.Errorf("StepTo(3) failed: %v", err)
	}

	// Next at boundary returns false
	if _, ok := s.Next(); ok {
		t.Error("expected false for Next() at end")
	}

	// StepTo invalid
	if _, err := s.StepTo(-1); err == nil {
		t.Error("expected error for negative index")
	}
	if _, err := s.StepTo(4); err == nil {
		t.Error("expected error for out of bounds index")
	}

	// Step back
	prevSnap, ok := s.Prev()
	if !ok || s.CurrentIndex() != 2 || prevSnap.Index != 2 {
		t.Errorf("Prev() failed, index=%d, ok=%v", s.CurrentIndex(), ok)
	}
}

func TestStepperSeekTimestamp(t *testing.T) {
	snaps := createMultiSnapshotSeries()
	s, _ := NewStepper(snaps)

	// Seek exact match for sample 2: 14:00:10
	target := time.Date(2026, 10, 5, 14, 0, 10, 0, time.UTC)
	snap, idx := s.SeekTimestamp(target)
	if idx != 2 || snap.Index != 2 {
		t.Errorf("SeekTimestamp exact got idx=%d, want 2", idx)
	}

	// Seek near sample 1 (14:00:05): 14:00:06 should match sample 1
	near1 := time.Date(2026, 10, 5, 14, 0, 6, 0, time.UTC)
	_, idx = s.SeekTimestamp(near1)
	if idx != 1 {
		t.Errorf("SeekTimestamp near got idx=%d, want 1", idx)
	}

	// Seek before start (matches 0)
	early := time.Date(2026, 10, 5, 13, 50, 0, 0, time.UTC)
	_, idx = s.SeekTimestamp(early)
	if idx != 0 {
		t.Errorf("SeekTimestamp early got idx=%d, want 0", idx)
	}

	// Seek after end (matches 3)
	late := time.Date(2026, 10, 5, 14, 10, 0, 0, time.UTC)
	_, idx = s.SeekTimestamp(late)
	if idx != 3 {
		t.Errorf("SeekTimestamp late got idx=%d, want 3", idx)
	}
}

func TestStepperSeekTimestampString(t *testing.T) {
	snaps := createMultiSnapshotSeries()
	s, _ := NewStepper(snaps)

	// RFC3339 layout
	_, idx, err := s.SeekTimestampString("2026-10-05T14:00:15Z")
	if err != nil || idx != 3 {
		t.Errorf("SeekTimestampString RFC3339 got idx=%d, err=%v", idx, err)
	}

	// Time-only layout
	_, idx, err = s.SeekTimestampString("14:00:05")
	if err != nil || idx != 1 {
		t.Errorf("SeekTimestampString time-only got idx=%d, err=%v", idx, err)
	}

	// Invalid string
	_, _, err = s.SeekTimestampString("not-a-timestamp")
	if err == nil {
		t.Error("expected error for invalid timestamp string")
	}
}

func TestStepperResolveTarget(t *testing.T) {
	snaps := createMultiSnapshotSeries()
	s, _ := NewStepper(snaps)

	// Sample 0: worker PID is 200
	pids, err := s.ResolveTarget(model.Target{Type: model.TargetPID, Value: "200"}, false)
	if err != nil || len(pids) != 1 || pids[0] != 200 {
		t.Fatalf("ResolveTarget PID 200 failed: %v", err)
	}

	// Sample 0: search by name exact
	pids, err = s.ResolveTarget(model.Target{Type: model.TargetName, Value: "sshd"}, true)
	if err != nil || len(pids) != 1 || pids[0] != 100 {
		t.Fatalf("ResolveTarget Name sshd failed: %v", err)
	}

	// Sample 0: search by name substring
	pids, err = s.ResolveTarget(model.Target{Type: model.TargetName, Value: "work"}, false)
	if err != nil || len(pids) != 1 || pids[0] != 200 {
		t.Fatalf("ResolveTarget Name substring work failed: %v", err)
	}

	// Search port 22
	pids, err = s.ResolveTarget(model.Target{Type: model.TargetPort, Value: "22"}, false)
	if err != nil || len(pids) != 1 || pids[0] != 100 {
		t.Fatalf("ResolveTarget Port 22 failed: %v", err)
	}

	// Search locked file
	pids, err = s.ResolveTarget(model.Target{Type: model.TargetFile, Value: "/tmp/worker.lock"}, false)
	if err != nil || len(pids) != 1 || pids[0] != 200 {
		t.Fatalf("ResolveTarget File failed: %v", err)
	}

	// Nonexistent PID
	_, err = s.ResolveTarget(model.Target{Type: model.TargetPID, Value: "9999"}, false)
	if err == nil {
		t.Error("expected error for missing PID")
	}

	// Step forward to sample 2 (worker is 202)
	_, _ = s.StepTo(2)
	pids, err = s.ResolveTarget(model.Target{Type: model.TargetName, Value: "worker"}, true)
	if err != nil || len(pids) != 1 || pids[0] != 202 {
		t.Fatalf("ResolveTarget after StepTo(2) got %v, want [202]", pids)
	}
}

func TestStepperResolveAncestry(t *testing.T) {
	snaps := createMultiSnapshotSeries()
	s, _ := NewStepper(snaps)

	// Resolve ancestry for worker (PID 200): chain should be systemd(1) -> sshd(100) -> worker(200)
	chain, err := s.ResolveAncestry(200)
	if err != nil {
		t.Fatalf("ResolveAncestry error: %v", err)
	}

	if len(chain) != 3 {
		t.Fatalf("got chain length %d, want 3", len(chain))
	}
	if chain[0].PID != 1 || chain[0].Command != "systemd" {
		t.Errorf("chain[0] = %+v, want PID 1", chain[0])
	}
	if chain[1].PID != 100 || chain[1].Command != "sshd" {
		t.Errorf("chain[1] = %+v, want PID 100", chain[1])
	}
	if chain[2].PID != 200 || chain[2].Command != "worker" {
		t.Errorf("chain[2] = %+v, want PID 200", chain[2])
	}

	// Test missing parent marks ParentExited
	orphanSnap := &model.Snapshot{
		Version: CurrentSnapshotVersion,
		Processes: []model.Process{
			{PID: 50, PPID: 40, Command: "orphan"}, // PPID 40 is not in snapshot
		},
	}
	os, _ := NewStepper([]*model.Snapshot{orphanSnap})
	orphanChain, err := os.ResolveAncestry(50)
	if err != nil {
		t.Fatalf("ResolveAncestry orphan: %v", err)
	}
	if len(orphanChain) != 1 || !orphanChain[0].ParentExited {
		t.Errorf("orphan process should have ParentExited=true: %+v", orphanChain)
	}

	// Nonexistent PID
	if _, err := s.ResolveAncestry(9999); err == nil {
		t.Error("expected error for nonexistent PID")
	}
}
