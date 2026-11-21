package record

import (
	"testing"
	"time"

	"github.com/pranshuparmar/witr/internal/pipeline"
	"github.com/pranshuparmar/witr/pkg/model"
)

func TestAnalyzeSnapshot(t *testing.T) {
	now := time.Now().UTC()
	snap := &model.Snapshot{
		Version:   CurrentSnapshotVersion,
		Index:     0,
		Timestamp: now,
		Processes: []model.Process{
			{PID: 1, PPID: 0, Command: "systemd", Cmdline: "/sbin/init"},
			{PID: 50, PPID: 1, Command: "dockerd", Cmdline: "/usr/bin/dockerd"},
			{PID: 100, PPID: 50, Command: "nginx", Cmdline: "nginx: master process", ContainerID: "c-nginx"},
			{PID: 101, PPID: 100, Command: "nginx", Cmdline: "nginx: worker process"},
			{PID: 102, PPID: 100, Command: "nginx", Cmdline: "nginx: worker process"},
		},
		Ports: []model.OpenPort{
			{PID: 100, Port: 80, Protocol: "tcp", State: "LISTEN", Address: "0.0.0.0"},
		},
		Containers: []*model.ContainerMatch{
			{ID: "c-nginx", Name: "web-server", Image: "nginx:alpine"},
		},
	}

	cfg := pipeline.AnalyzeConfig{
		PID:     100,
		Verbose: true,
		Tree:    true,
		Target:  model.Target{Type: model.TargetPID, Value: "100"},
	}

	res, err := AnalyzeSnapshot(snap, 100, cfg)
	if err != nil {
		t.Fatalf("AnalyzeSnapshot failed: %v", err)
	}

	if res.Process.PID != 100 || res.Process.Command != "nginx" {
		t.Errorf("Process mismatch: %+v", res.Process)
	}

	if len(res.Ancestry) != 3 {
		t.Fatalf("expected 3 ancestors in chain (systemd -> dockerd -> nginx), got %d", len(res.Ancestry))
	}
	if res.Ancestry[0].Command != "systemd" || res.Ancestry[1].Command != "dockerd" || res.Ancestry[2].Command != "nginx" {
		t.Errorf("ancestry chain mismatch: %+v", res.Ancestry)
	}

	// Verify child processes detected
	if len(res.Children) != 2 {
		t.Fatalf("expected 2 child worker processes, got %d", len(res.Children))
	}
	if res.Children[0].PID != 101 || res.Children[1].PID != 102 {
		t.Errorf("children mismatch: %+v", res.Children)
	}

	// Verify container detected
	if res.Container == nil || res.Container.Name != "web-server" {
		t.Errorf("container match mismatch: %+v", res.Container)
	}

	// Test port target socket enrichment
	portCfg := pipeline.AnalyzeConfig{
		PID:    100,
		Target: model.Target{Type: model.TargetPort, Value: "80"},
	}
	portRes, err := AnalyzeSnapshot(snap, 100, portCfg)
	if err != nil {
		t.Fatalf("AnalyzeSnapshot port failed: %v", err)
	}
	if portRes.SocketInfo == nil || portRes.SocketInfo.Port != 80 {
		t.Errorf("SocketInfo mismatch: %+v", portRes.SocketInfo)
	}
}

func TestAnalyzeSnapshotNilOrMissing(t *testing.T) {
	_, err := AnalyzeSnapshot(nil, 1, pipeline.AnalyzeConfig{})
	if err == nil {
		t.Error("expected error for nil snapshot")
	}

	snap := &model.Snapshot{Version: CurrentSnapshotVersion}
	_, err = AnalyzeSnapshot(snap, 999, pipeline.AnalyzeConfig{})
	if err == nil {
		t.Error("expected error for missing PID in snapshot")
	}
}
