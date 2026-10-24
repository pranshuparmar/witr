package app

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/internal/record"
	"github.com/pranshuparmar/witr/pkg/model"
	"github.com/spf13/cobra"
)

func createTestLogFile(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_session.jsonl")

	base := time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)
	snapshots := []*model.Snapshot{
		{
			Version:   1,
			Index:     0,
			Timestamp: base,
			Hostname:  "test-box",
			Processes: []model.Process{
				{PID: 1, PPID: 0, Command: "init", Cmdline: "/sbin/init", StartedAt: base.Add(-1 * time.Hour)},
				{PID: 100, PPID: 1, Command: "nginx", Cmdline: "nginx: master process", StartedAt: base.Add(-30 * time.Minute)},
				{PID: 101, PPID: 100, Command: "nginx", Cmdline: "nginx: worker process", StartedAt: base.Add(-30 * time.Minute)},
			},
			Ports: []model.OpenPort{
				{PID: 100, Port: 80, Protocol: "tcp", State: "LISTEN"},
			},
		},
		{
			Version:   1,
			Index:     1,
			Timestamp: base.Add(5 * time.Second),
			Hostname:  "test-box",
			Processes: []model.Process{
				{PID: 1, PPID: 0, Command: "init", Cmdline: "/sbin/init", StartedAt: base.Add(-1 * time.Hour)},
				{PID: 100, PPID: 1, Command: "nginx", Cmdline: "nginx: master process", StartedAt: base.Add(-30 * time.Minute)},
				{PID: 101, PPID: 100, Command: "nginx", Cmdline: "nginx: worker process", StartedAt: base.Add(-30 * time.Minute)},
				{PID: 200, PPID: 1, Command: "cron", Cmdline: "/usr/sbin/cron -f", StartedAt: base.Add(2 * time.Second)},
			},
			Ports: []model.OpenPort{
				{PID: 100, Port: 80, Protocol: "tcp", State: "LISTEN"},
			},
		},
	}

	if err := record.SaveFile(logPath, snapshots); err != nil {
		t.Fatalf("create test log file: %v", err)
	}
	return logPath
}

func resetCommandFlags(cmd *cobra.Command) {
	for _, c := range append(cmd.Commands(), cmd) {
		for _, f := range []string{"json", "tree", "short", "warnings", "verbose", "exact", "all", "env", "step", "timestamp", "record", "replay"} {
			if flag := c.Flags().Lookup(f); flag != nil {
				_ = flag.Value.Set(flag.DefValue)
				flag.Changed = false
			}
		}
		for _, f := range []string{"pid", "port", "container", "file", "target-file"} {
			if flag := c.Flags().Lookup(f); flag != nil {
				_ = flag.Value.Set("")
				flag.Changed = false
			}
		}
	}
}

func executeArgs(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := Root()
	resetCommandFlags(cmd)

	var out, errOut strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	t.Cleanup(func() {
		cmd.SetArgs(nil)
		resetCommandFlags(cmd)
	})

	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestReplaySubcommandSummary(t *testing.T) {
	logPath := createTestLogFile(t)

	stdout, stderr, err := executeArgs(t, "replay", "-f", logPath)
	if err != nil {
		t.Fatalf("replay execute failed: %v, stderr: %q, stdout: %q", err, stderr, stdout)
	}

	if !strings.Contains(stdout, "Recorded Session Summary") {
		t.Errorf("expected summary in output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Snapshots:") || !strings.Contains(stdout, "2") {
		t.Errorf("expected 2 snapshots in output, got:\n%s", stdout)
	}
}

func TestReplaySubcommandSummaryJSON(t *testing.T) {
	logPath := createTestLogFile(t)

	stdout, _, err := executeArgs(t, "replay", "-f", logPath, "--json")
	if err != nil {
		t.Fatalf("replay execute failed: %v", err)
	}

	var rep record.SessionReport
	if err := json.Unmarshal([]byte(stdout), &rep); err != nil {
		t.Fatalf("failed to parse json summary: %v\noutput: %s", err, stdout)
	}
	if rep.Summary.TotalSamples != 2 {
		t.Errorf("got %d total samples, want 2", rep.Summary.TotalSamples)
	}
}

func TestReplayQueryByName(t *testing.T) {
	logPath := createTestLogFile(t)

	stdout, _, err := executeArgs(t, "replay", "-f", logPath, "cron")
	if err != nil {
		t.Fatalf("replay cron failed: %v", err)
	}

	if !strings.Contains(stdout, "cron") || !strings.Contains(stdout, "200") {
		t.Errorf("expected cron (PID 200) in report, got:\n%s", stdout)
	}
}

func TestReplayQueryByPIDJSON(t *testing.T) {
	logPath := createTestLogFile(t)

	stdout, _, err := executeArgs(t, "replay", "-f", logPath, "--pid", "100", "--json")
	if err != nil {
		t.Fatalf("replay by pid failed: %v", err)
	}

	var res model.Result
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("failed to decode json result: %v\noutput: %s", err, stdout)
	}
	if res.Process.PID != 100 || res.Process.Command != "nginx" {
		t.Errorf("unexpected process in result: %+v", res.Process)
	}
	if len(res.Ancestry) != 2 {
		t.Errorf("expected ancestry length 2, got %d", len(res.Ancestry))
	}
}

func TestReplayQueryByPort(t *testing.T) {
	logPath := createTestLogFile(t)

	stdout, _, err := executeArgs(t, "replay", "-f", logPath, "--port", "80")
	if err != nil {
		t.Fatalf("replay by port failed: %v", err)
	}

	if !strings.Contains(stdout, "nginx") {
		t.Errorf("expected nginx for port 80, got:\n%s", stdout)
	}
}

func TestReplayTreeOutput(t *testing.T) {
	logPath := createTestLogFile(t)

	stdout, _, err := executeArgs(t, "replay", "-f", logPath, "--pid", "100", "--tree")
	if err != nil {
		t.Fatalf("replay tree failed: %v", err)
	}

	if !strings.Contains(stdout, "init") || !strings.Contains(stdout, "nginx") {
		t.Errorf("tree output missing expected nodes:\n%s", stdout)
	}
}

func TestReplayStepSelection(t *testing.T) {
	logPath := createTestLogFile(t)

	// Sample 0 did not have cron (PID 200). Stepping to 0 should result in not found for cron.
	_, _, err := executeArgs(t, "replay", "-f", logPath, "--step", "0", "cron")
	if err == nil {
		t.Error("expected error looking up cron in snapshot step 0")
	}

	// Stepping to sample 1 should find cron
	stdout, _, err := executeArgs(t, "replay", "-f", logPath, "--step", "1", "cron")
	if err != nil {
		t.Fatalf("expected cron in step 1, got error: %v", err)
	}
	if !strings.Contains(stdout, "cron") {
		t.Errorf("expected cron in step 1 output, got:\n%s", stdout)
	}
}

func TestReplayAllSnapshots(t *testing.T) {
	logPath := createTestLogFile(t)

	stdout, _, err := executeArgs(t, "replay", "-f", logPath, "--all", "--pid", "100")
	if err != nil {
		t.Fatalf("replay --all failed: %v", err)
	}

	if !strings.Contains(stdout, "=== Sample [1/2]") || !strings.Contains(stdout, "=== Sample [2/2]") {
		t.Errorf("expected sample headers for all snapshots, got:\n%s", stdout)
	}
}

func TestReplayRootFlag(t *testing.T) {
	logPath := createTestLogFile(t)

	stdout, _, err := executeArgs(t, "--replay", logPath, "--pid", "100")
	if err != nil {
		t.Fatalf("root --replay flag failed: %v", err)
	}

	if !strings.Contains(stdout, "nginx") {
		t.Errorf("expected nginx in report, got:\n%s", stdout)
	}
}

func TestRecordSubcommandIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "rec.jsonl")

	_, _, err := executeArgs(t, "record", "-o", logPath, "-i", "10ms", "-n", "2")
	if err != nil {
		t.Fatalf("record subcommand failed: %v", err)
	}

	snaps, err := record.LoadFile(logPath)
	if err != nil {
		t.Fatalf("LoadFile on recorded file failed: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("got %d snapshots recorded, want 2", len(snaps))
	}
	if len(snaps[0].Processes) == 0 {
		t.Error("expected processes recorded in snapshot 0")
	}
}

func TestReplayNonexistentFile(t *testing.T) {
	_, _, err := executeArgs(t, "replay", "-f", "/path/to/nonexistent/witr.jsonl")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
