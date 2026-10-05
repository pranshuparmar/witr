package record

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func sampleSnapshot(index int, pid int) *model.Snapshot {
	now := time.Date(2026, 10, 5, 12, 0, index, 0, time.UTC)
	return &model.Snapshot{
		Version:   CurrentSnapshotVersion,
		Index:     index,
		Timestamp: now,
		Hostname:  "test-host",
		Processes: []model.Process{
			{
				PID:       pid,
				PPID:      1,
				Command:   "daemon",
				Cmdline:   "/usr/bin/daemon --port 8080",
				User:      "root",
				StartedAt: now.Add(-10 * time.Minute),
			},
		},
		Ports: []model.OpenPort{
			{
				PID:      pid,
				Port:     8080,
				Protocol: "tcp",
				State:    "LISTEN",
			},
		},
		Containers: []*model.ContainerMatch{
			{
				ID:   "c123",
				Name: "my-container",
			},
		},
		LockedFiles: []*model.LockedFile{
			{
				PID:  pid,
				Path: "/var/lock/daemon.lock",
				Type: "POSIX",
				Mode: "WRITE",
			},
		},
	}
}

func TestSerializeDeserializeRoundtrip(t *testing.T) {
	orig := sampleSnapshot(0, 100)

	data, err := Serialize(orig)
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}

	decoded, err := Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize error: %v", err)
	}

	if decoded.Index != orig.Index {
		t.Errorf("got index %d, want %d", decoded.Index, orig.Index)
	}
	if decoded.Hostname != orig.Hostname {
		t.Errorf("got hostname %q, want %q", decoded.Hostname, orig.Hostname)
	}
	if len(decoded.Processes) != 1 || decoded.Processes[0].PID != 100 {
		t.Errorf("processes mismatch: %+v", decoded.Processes)
	}
	if len(decoded.Ports) != 1 || decoded.Ports[0].Port != 8080 {
		t.Errorf("ports mismatch: %+v", decoded.Ports)
	}
	if len(decoded.Containers) != 1 || decoded.Containers[0].Name != "my-container" {
		t.Errorf("containers mismatch: %+v", decoded.Containers)
	}
	if len(decoded.LockedFiles) != 1 || decoded.LockedFiles[0].Path != "/var/lock/daemon.lock" {
		t.Errorf("locked files mismatch: %+v", decoded.LockedFiles)
	}
}

func TestSerializeNilSnapshot(t *testing.T) {
	_, err := Serialize(nil)
	if err == nil {
		t.Fatal("expected error on serializing nil snapshot")
	}
}

func TestDeserializeEmptyData(t *testing.T) {
	_, err := Deserialize([]byte("   \n\t"))
	if err == nil {
		t.Fatal("expected error on deserializing empty data")
	}
}

func TestStreamWriterAndReader(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	for i := 0; i < 5; i++ {
		if err := w.Write(sampleSnapshot(i, 100+i)); err != nil {
			t.Fatalf("write sample %d: %v", i, err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	r, err := NewReader(&buf)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}

	snaps, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(snaps) != 5 {
		t.Fatalf("got %d snapshots, want 5", len(snaps))
	}
	for i, snap := range snaps {
		if snap.Index != i {
			t.Errorf("snap[%d] index = %d", i, snap.Index)
		}
		if snap.Processes[0].PID != 100+i {
			t.Errorf("snap[%d] pid = %d, want %d", i, snap.Processes[0].PID, 100+i)
		}
	}
}

func TestReaderJSONArray(t *testing.T) {
	jsonArray := `[
		{"version": 1, "index": 0, "timestamp": "2026-10-05T12:00:00Z", "processes": [{"pid": 1, "command": "init"}]},
		{"version": 1, "index": 1, "timestamp": "2026-10-05T12:00:01Z", "processes": [{"pid": 2, "command": "kthreadd"}]}
	]`

	r, err := NewReader(bytes.NewReader([]byte(jsonArray)))
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}

	snaps, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(snaps) != 2 {
		t.Fatalf("got %d snapshots, want 2", len(snaps))
	}
	if snaps[0].Processes[0].Command != "init" {
		t.Errorf("snap 0 command: %q", snaps[0].Processes[0].Command)
	}
	if snaps[1].Processes[0].Command != "kthreadd" {
		t.Errorf("snap 1 command: %q", snaps[1].Processes[0].Command)
	}
}

func TestReaderGzipStream(t *testing.T) {
	var gzBuf bytes.Buffer
	gzw := gzip.NewWriter(&gzBuf)

	snap1, _ := Serialize(sampleSnapshot(0, 10))
	snap2, _ := Serialize(sampleSnapshot(1, 20))
	_, _ = gzw.Write(snap1)
	_, _ = gzw.Write(snap2)
	_ = gzw.Close()

	r, err := NewReader(&gzBuf)
	if err != nil {
		t.Fatalf("NewReader gzip: %v", err)
	}

	snaps, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll gzip: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("got %d snapshots from gzip, want 2", len(snaps))
	}
	if snaps[0].Processes[0].PID != 10 {
		t.Errorf("snap 0 PID = %d, want 10", snaps[0].Processes[0].PID)
	}
	if snaps[1].Processes[0].PID != 20 {
		t.Errorf("snap 1 PID = %d, want 20", snaps[1].Processes[0].PID)
	}
}

func TestLoadAndSaveFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "snapshots.jsonl")

	input := []*model.Snapshot{
		sampleSnapshot(0, 101),
		sampleSnapshot(1, 102),
	}

	if err := SaveFile(filePath, input); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	loaded, err := LoadFile(filePath)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("loaded %d snapshots, want 2", len(loaded))
	}
	if loaded[0].Processes[0].PID != 101 || loaded[1].Processes[0].PID != 102 {
		t.Errorf("unexpected loaded processes: %+v", loaded)
	}

	// Test missing file
	if _, err := LoadFile(filepath.Join(tmpDir, "nonexistent.jsonl")); err == nil {
		t.Error("expected error loading nonexistent file")
	}

	// Test empty file
	emptyPath := filepath.Join(tmpDir, "empty.jsonl")
	if err := os.WriteFile(emptyPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(emptyPath); err == nil {
		t.Error("expected error on empty file")
	}
}

func TestReaderSingleReadAtEOF(t *testing.T) {
	r, err := NewReader(bytes.NewReader([]byte("")))
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	_, err = r.Read()
	if err != io.EOF {
		t.Errorf("got error %v, want io.EOF", err)
	}
}
