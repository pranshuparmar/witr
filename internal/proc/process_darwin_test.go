//go:build darwin

package proc

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestIsEnvVarName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"HOME", true}, {"PATH", true}, {"MY_VAR", true}, {"lowercase", true},
		{"_UNDERSCORE", true}, {"VAR1", true}, {"a", true},
		{"", false}, {"VAR-NAME", false}, {"VAR.NAME", false}, {"VAR NAME", false},
		{"VAR=VALUE", false}, {"/path/to", false}, {"VAR$NAME", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEnvVarName(tt.name); got != tt.want {
				t.Errorf("isEnvVarName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestReadProcess(t *testing.T) {
	proc, err := ReadProcess(os.Getpid())
	if err != nil {
		t.Fatalf("ReadProcess failed: %v", err)
	}
	if proc.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", proc.PID, os.Getpid())
	}
	now := time.Now()
	if proc.StartedAt.After(now) {
		t.Errorf("StartedAt in future: %v > %v", proc.StartedAt, now)
	}
	if proc.Command == "" {
		t.Error("Command should not be empty")
	}
	if proc.PPID <= 0 {
		t.Errorf("PPID should be positive, got %d", proc.PPID)
	}
}

func TestReadProcessInvalidPID(t *testing.T) {
	_, err := ReadProcess(-1)
	if err == nil {
		t.Error("ReadProcess(-1) should return error")
	}
	_, err = ReadProcess(999999999)
	if err == nil {
		t.Error("ReadProcess(999999999) should return error for non-existent PID")
	}
}

func TestReadUser(t *testing.T) {
	if got := readUser(1); got != "unknown" {
		t.Errorf("readUser() = %v, want unknown", got)
	}
}

func TestReadUserByUID(t *testing.T) {
	if got := readUserByUID(0); got != "root" {
		t.Errorf("readUserByUID(0) = %v, want root", got)
	}
	if got := readUserByUID(os.Getuid()); got == "" {
		t.Error("readUserByUID(self) should return username")
	}
}

func TestResolveUID(t *testing.T) {
	if got := resolveUID(0); got != "root" {
		t.Errorf("resolveUID(0) = %v, want root", got)
	}
	got := resolveUID(99999)
	if got != "99999" {
		t.Errorf("resolveUID(99999) = %v, want 99999", got)
	}
}

func TestAddStateExplanation(t *testing.T) {
	states := []string{"LISTEN", "TIME_WAIT", "CLOSE_WAIT", "FIN_WAIT_1", "FIN_WAIT_2",
		"ESTABLISHED", "SYN_SENT", "SYN_RECEIVED", "CLOSING", "LAST_ACK", "UNKNOWN"}
	for _, s := range states {
		info := &model.SocketInfo{State: s}
		addStateExplanation(info)
		if info.Explanation == "" {
			t.Errorf("addStateExplanation(%s) gave empty explanation", s)
		}
	}
	timeWait := &model.SocketInfo{State: "TIME_WAIT"}
	addStateExplanation(timeWait)
	if timeWait.Workaround == "" {
		t.Error("TIME_WAIT should have workaround")
	}
	closeWait := &model.SocketInfo{State: "CLOSE_WAIT"}
	addStateExplanation(closeWait)
	if closeWait.Workaround == "" {
		t.Error("CLOSE_WAIT should have workaround")
	}
}

func TestGetTIMEWAITRemaining(t *testing.T) {
	if got := GetTIMEWAITRemaining(); got == "" {
		t.Error("GetTIMEWAITRemaining returned empty")
	}
}

func TestGetMSLDuration(t *testing.T) {
	got := GetMSLDuration()
	if got <= 0 {
		t.Errorf("GetMSLDuration() = %d, want > 0", got)
	}
	if got > 120000 {
		t.Errorf("GetMSLDuration() = %d, seems unreasonably high", got)
	}
}

func TestReverse(t *testing.T) {
	input := []model.Process{{PID: 1}, {PID: 2}, {PID: 3}}
	result := reverse(input)
	if result[0].PID != 3 || result[2].PID != 1 {
		t.Errorf("reverse failed: %v", result)
	}
	empty := reverse([]model.Process{})
	if len(empty) != 0 {
		t.Error("reverse empty failed")
	}
	single := reverse([]model.Process{{PID: 5}})
	if single[0].PID != 5 {
		t.Error("reverse single failed")
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if !containsString(slice, "b") {
		t.Error("containsString should find 'b'")
	}
	if containsString(slice, "d") {
		t.Error("containsString should not find 'd'")
	}
	if containsString(nil, "a") {
		t.Error("containsString nil should return false")
	}
	if containsString([]string{}, "a") {
		t.Error("containsString empty should return false")
	}
}

func TestGetSocketStates(t *testing.T) {
	states, err := GetSocketStates(80)
	if err != nil {
		return
	}
	for _, s := range states {
		if s.Port != 80 {
			t.Errorf("GetSocketStates returned wrong port: %d", s.Port)
		}
	}
}

func TestGetSocketStateForPort(t *testing.T) {
	state := GetSocketStateForPort(80)
	if state != nil && state.Port != 80 {
		t.Errorf("GetSocketStateForPort returned wrong port: %d", state.Port)
	}
}

func TestCountSocketsByState(t *testing.T) {
	counts := CountSocketsByState(80)
	if counts == nil {
		t.Error("CountSocketsByState returned nil")
	}
}

func TestResolveAncestry(t *testing.T) {
	chain, err := ResolveAncestry(os.Getpid())
	if err != nil {
		t.Fatalf("ResolveAncestry failed: %v", err)
	}
	if len(chain) == 0 {
		t.Error("ResolveAncestry returned empty chain")
	}
	if chain[len(chain)-1].PID != os.Getpid() {
		t.Error("Last process in ancestry should be self")
	}
	if chain[0].PID != 1 && chain[0].PPID != 0 {
		t.Error("First process should be init/launchd")
	}
}

func TestResolveAncestryInvalidPID(t *testing.T) {
	chain, err := ResolveAncestry(-1)
	if err != nil && len(chain) > 0 {
		t.Error("ResolveAncestry(-1) should return error or empty chain")
	}
}

func TestGetFileContext(t *testing.T) {
	ctx := GetFileContext(os.Getpid())
	if ctx != nil {
		if ctx.OpenFiles < 0 {
			t.Error("OpenFiles should not be negative")
		}
		if ctx.FileLimit < 0 {
			t.Error("FileLimit should not be negative")
		}
	}
}

func TestGetResourceContext(t *testing.T) {
	ctx := GetResourceContext(os.Getpid())
	_ = ctx
}

func TestReadListeningSockets(t *testing.T) {
	sockets, err := readListeningSockets()
	if err != nil {
		return
	}
	for inode, sock := range sockets {
		if sock.Inode != inode {
			t.Errorf("Socket inode mismatch: %s vs %s", sock.Inode, inode)
		}
		if sock.Port <= 0 {
			t.Errorf("Socket port should be positive: %d", sock.Port)
		}
	}
}

func TestSocketsForPID(t *testing.T) {
	inodes := socketsForPID(os.Getpid())
	_ = inodes
}

func TestGetCmdline(t *testing.T) {
	cmd := GetCmdline(os.Getpid())
	if cmd == "" {
		t.Log("GetCmdline returned empty (may be expected)")
	}
}

func TestParseNetstatAddr(t *testing.T) {
	tests := []struct {
		input    string
		wantAddr string
		wantPort int
	}{
		{"127.0.0.1.8080", "127.0.0.1", 8080},
		{"127.0.0.1:8080", "127.0.0.1", 8080},
		{"*.80", "0.0.0.0", 80},
		{"*:80", "0.0.0.0", 80},
		{"[::]:8080", "::", 8080},
		{"[::1]:8080", "::1", 8080},
		{"[]:8080", "", 0},
		{"", "", 0},
		{"invalid", "", 0},
		{"no.port.here", "", 0},
	}
	for _, tt := range tests {
		addr, port := parseNetstatAddr(tt.input)
		if addr != tt.wantAddr || port != tt.wantPort {
			t.Errorf("parseNetstatAddr(%q) = (%q, %d), want (%q, %d)",
				tt.input, addr, port, tt.wantAddr, tt.wantPort)
		}
	}
}

func TestReadListeningSocketsNetstat(t *testing.T) {
	sockets, err := readListeningSocketsNetstat()
	if err != nil {
		return
	}
	for _, sock := range sockets {
		if sock.Port <= 0 {
			t.Errorf("Socket port should be positive: %d", sock.Port)
		}
	}
}

func TestBootTime(t *testing.T) {
	bt := bootTime()
	if bt.IsZero() {
		t.Error("bootTime returned zero")
	}
	now := time.Now()
	if bt.After(now) {
		t.Error("bootTime should not be in future")
	}
	if now.Sub(bt) > 365*24*time.Hour {
		t.Log("System uptime > 1 year (unusual)")
	}
}

func TestTicksPerSecond(t *testing.T) {
	tps := ticksPerSecond()
	if tps <= 0 {
		t.Errorf("ticksPerSecond() = %v, want > 0", tps)
	}
	if tps != 100 {
		t.Logf("ticksPerSecond() = %v (expected 100 on most systems)", tps)
	}
}

func TestGetCommandLine(t *testing.T) {
	cmd := getCommandLine(os.Getpid())
	if cmd == "" {
		t.Log("getCommandLine returned empty")
	}
}

func TestGetEnvironment(t *testing.T) {
	env := getEnvironment(os.Getpid())
	_ = env
}

func TestGetWorkingDirectory(t *testing.T) {
	cwd := getWorkingDirectory(os.Getpid())
	if cwd == "" || cwd == "unknown" {
		t.Log("getWorkingDirectory returned empty/unknown")
	}
	expectedCwd, _ := os.Getwd()
	if cwd != expectedCwd && cwd != "unknown" {
		t.Logf("getWorkingDirectory mismatch: got %q, expected %q", cwd, expectedCwd)
	}
}

func TestDetectContainer(t *testing.T) {
	container := detectContainer(os.Getpid())
	_ = container
}

func TestDetectLaunchdService(t *testing.T) {
	service := detectLaunchdService(os.Getpid())
	_ = service
}

func TestDetectGitInfo(t *testing.T) {
	repo, branch := detectGitInfo("/nonexistent")
	if repo != "" || branch != "" {
		t.Error("detectGitInfo should return empty for nonexistent path")
	}
	detectGitInfo("unknown")
	detectGitInfo("")
	detectGitInfo("/")
}

func TestCheckResourceUsage(t *testing.T) {
	health := checkResourceUsage(os.Getpid(), "healthy")
	if health == "" {
		t.Error("checkResourceUsage returned empty")
	}
	health = checkResourceUsage(-1, "healthy")
	if health != "healthy" {
		t.Error("checkResourceUsage should return original health for invalid PID")
	}
}

func TestGetOpenFileCount(t *testing.T) {
	open, max := getOpenFileCount(os.Getpid())
	if open < 0 {
		t.Error("open file count should not be negative")
	}
	if max < 0 {
		t.Error("max file count should not be negative")
	}
	if open > max && max > 0 {
		t.Errorf("open files (%d) > max (%d)", open, max)
	}
}

func TestGetFileLimit(t *testing.T) {
	limit := getFileLimit(os.Getpid())
	if limit < 0 {
		t.Error("file limit should not be negative")
	}
}

func TestGetLockedFiles(t *testing.T) {
	locked := getLockedFiles(os.Getpid())
	_ = locked
}

func TestCheckPreventsSleep(t *testing.T) {
	prevents := checkPreventsSleep(os.Getpid())
	_ = prevents
}

func TestGetThermalState(t *testing.T) {
	state := getThermalState()
	// Empty string is valid (normal/no throttling)
	// Non-empty should contain "throttling" or "thermal"
	if state != "" && !strings.Contains(strings.ToLower(state), "throttl") && !strings.Contains(strings.ToLower(state), "thermal") {
		t.Errorf("getThermalState returned unexpected value: %q", state)
	}
}

func TestGetEnergyImpact(t *testing.T) {
	impact := GetEnergyImpact(os.Getpid())
	if impact == "" {
		t.Log("GetEnergyImpact returned empty")
	}
}
