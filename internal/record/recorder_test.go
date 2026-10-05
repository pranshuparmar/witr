package record

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestRunRecorderWithCount(t *testing.T) {
	var buf bytes.Buffer
	mockCollector := func(index int) (*model.Snapshot, error) {
		return &model.Snapshot{
			Version:   CurrentSnapshotVersion,
			Index:     index,
			Timestamp: time.Now().UTC(),
			Processes: []model.Process{
				{PID: 10 + index, Command: "testproc"},
			},
		}, nil
	}

	samplesCaptured := 0
	opts := RecorderOptions{
		Output:    &buf,
		Interval:  10 * time.Millisecond,
		Count:     3,
		Collector: mockCollector,
		OnSample: func(snap *model.Snapshot) {
			samplesCaptured++
		},
	}

	ctx := context.Background()
	if err := RunRecorder(ctx, opts); err != nil {
		t.Fatalf("RunRecorder error: %v", err)
	}

	if samplesCaptured != 3 {
		t.Errorf("got %d samples, want 3", samplesCaptured)
	}

	reader, err := NewReader(&buf)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	snaps, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(snaps) != 3 {
		t.Fatalf("got %d snapshots in stream, want 3", len(snaps))
	}
	for i, s := range snaps {
		if s.Index != i {
			t.Errorf("snap[%d] index = %d", i, s.Index)
		}
	}
}

func TestRunRecorderWithCancellation(t *testing.T) {
	var buf bytes.Buffer
	mockCollector := func(index int) (*model.Snapshot, error) {
		return &model.Snapshot{
			Version:   CurrentSnapshotVersion,
			Index:     index,
			Timestamp: time.Now().UTC(),
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	opts := RecorderOptions{
		Output:    &buf,
		Interval:  50 * time.Millisecond,
		Collector: mockCollector,
	}

	go func() {
		time.Sleep(75 * time.Millisecond)
		cancel()
	}()

	err := RunRecorder(ctx, opts)
	if err == nil || err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	reader, err := NewReader(&buf)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	snaps, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(snaps) < 1 {
		t.Errorf("expected at least 1 snapshot recorded before cancellation, got %d", len(snaps))
	}
}

func TestRunRecorderToFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "out.jsonl")

	opts := RecorderOptions{
		FilePath: logFile,
		Interval: 10 * time.Millisecond,
		Count:    2,
		Collector: func(index int) (*model.Snapshot, error) {
			return &model.Snapshot{
				Version:   CurrentSnapshotVersion,
				Index:     index,
				Timestamp: time.Now().UTC(),
			}, nil
		},
	}

	if err := RunRecorder(context.Background(), opts); err != nil {
		t.Fatalf("RunRecorderToFile error: %v", err)
	}

	snaps, err := LoadFile(logFile)
	if err != nil {
		t.Fatalf("LoadFile error: %v", err)
	}
	if len(snaps) != 2 {
		t.Errorf("got %d snapshots in file, want 2", len(snaps))
	}
}
