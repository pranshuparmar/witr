//go:build darwin

package proc

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/pranshuparmar/witr/internal/proc/mocks"
	"github.com/pranshuparmar/witr/pkg/model"
)

func TestIsEnvVarName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"HOME", true}, {"PATH", true}, {"MY_VAR", true}, {"VAR123", true},
		{"", false}, {"VAR=value", false}, {"path/to/file", false},
	}
	for _, tt := range tests {
		if got := isEnvVarName(tt.input); got != tt.want {
			t.Errorf("isEnvVarName(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestReadProcess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("ps", "-p", "123", "-o", "pid=,ppid=,uid=,lstart=,state=,ucomm=").
		Return([]byte("  123  456  501 Mon Dec 30 10:00:00 2024 S testproc\n"), nil)
	mockExec.EXPECT().Run("ps", "-p", "123", "-o", "args=").
		Return([]byte("/usr/bin/test --flag\n"), nil).Times(2)
	mockExec.EXPECT().Run("ps", "-p", "123", "-E", "-o", "command=").
		Return([]byte("/bin/test HOME=/Users/test\n"), nil)
	mockExec.EXPECT().Run("lsof", "-a", "-p", "123", "-d", "cwd", "-F", "n").
		Return(nil, errors.New("lsof failed"))
	mockExec.EXPECT().Run("launchctl", "blame", "123").
		Return([]byte("com.apple.Safari\n"), nil)
	mockExec.EXPECT().Run("lsof", "-i", "TCP", "-s", "TCP:LISTEN", "-n", "-P", "-F", "pn").
		Return(nil, errors.New("lsof failed"))
	mockExec.EXPECT().Run("netstat", "-an", "-p", "tcp").
		Return([]byte(""), nil)
	mockExec.EXPECT().Run("lsof", "-a", "-p", "123", "-i", "TCP", "-n", "-P", "-F", "n").
		Return(nil, errors.New("lsof failed"))
	mockExec.EXPECT().Run("ps", "-p", "123", "-o", "pcpu=,rss=").
		Return([]byte("95.0 2097152\n"), nil)

	proc, err := ReadProcess(123)
	if err != nil {
		t.Fatalf("ReadProcess failed: %v", err)
	}
	if proc.PID != 123 {
		t.Errorf("PID = %d, want 123", proc.PID)
	}
	if proc.PPID != 456 {
		t.Errorf("PPID = %d, want 456", proc.PPID)
	}
	if proc.Command != "testproc" {
		t.Errorf("Command = %q, want testproc", proc.Command)
	}
	if proc.Cmdline != "/usr/bin/test --flag" {
		t.Errorf("Cmdline = %q, want /usr/bin/test --flag", proc.Cmdline)
	}
	if proc.WorkingDir != "unknown" {
		t.Errorf("WorkingDir = %q, want unknown", proc.WorkingDir)
	}
	if proc.Service != "com.apple.Safari" {
		t.Errorf("Service = %q, want com.apple.Safari", proc.Service)
	}
	if proc.Health != "high-cpu" {
		t.Errorf("Health = %q, want high-cpu", proc.Health)
	}

	// Expect Local time because ps outputs local time
	expectedTime := time.Date(2024, 12, 30, 10, 0, 0, 0, time.Local)
	if !proc.StartedAt.Equal(expectedTime) {
		t.Errorf("StartedAt = %v, want %v", proc.StartedAt, expectedTime)
	}
}

func TestReadProcessInvalidPID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("ps", "-p", "-1", "-o", "pid=,ppid=,uid=,lstart=,state=,ucomm=").
		Return(nil, errors.New("invalid pid"))

	_, err := ReadProcess(-1)
	if err == nil {
		t.Error("ReadProcess(-1) should return error")
	}
}

func TestReadUser(t *testing.T) {
	got := readUser(0)
	if got != "unknown" {
		t.Errorf("readUser(0) = %q, want unknown", got)
	}
}

func TestReadUserByUID(t *testing.T) {
	got := readUserByUID(0)
	if got != "root" {
		t.Errorf("readUserByUID(0) = %q, want root", got)
	}
}

func TestResolveUID(t *testing.T) {
	got := resolveUID(0)
	if got != "root" {
		t.Errorf("resolveUID(0) = %q, want root", got)
	}
}

func TestAddStateExplanation(t *testing.T) {
	tests := []struct {
		state string
	}{
		{"LISTEN"}, {"ESTABLISHED"}, {"TIME_WAIT"}, {"CLOSE_WAIT"},
	}
	for _, tt := range tests {
		info := &model.SocketInfo{State: tt.state}
		addStateExplanation(info)
		if info.Explanation == "" {
			t.Errorf("addStateExplanation(%q) produced empty explanation", tt.state)
		}
		if tt.state == "TIME_WAIT" && info.Workaround == "" {
			t.Error("TIME_WAIT should include workaround guidance")
		}
	}
}

func TestGetTIMEWAITRemaining(t *testing.T) {
	remaining := GetTIMEWAITRemaining()
	if remaining != "up to 60s remaining (macOS default)" {
		t.Errorf("GetTIMEWAITRemaining = %q", remaining)
	}
}

func TestGetMSLDuration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("sysctl", "-n", "net.inet.tcp.msl").Return([]byte("15000\n"), nil)

	duration := GetMSLDuration()
	if duration != 15000 {
		t.Errorf("GetMSLDuration = %d, want 15000", duration)
	}
}

func TestReverse(t *testing.T) {
	input := []model.Process{{PID: 1}, {PID: 2}, {PID: 3}}
	got := reverse(input)
	if len(got) != 3 || got[0].PID != 3 {
		t.Error("reverse failed")
	}
}

func TestContainsString(t *testing.T) {
	if !containsString([]string{"a", "b"}, "b") {
		t.Error("containsString should find b")
	}
	if containsString([]string{"a", "b"}, "c") {
		t.Error("containsString should not find c")
	}
}

func TestGetSocketStates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("netstat", "-an", "-p", "tcp").
		Return([]byte("tcp4  0  0  *.8080  *.*  LISTEN\n"), nil)

	states, err := GetSocketStates(8080)
	if err != nil {
		t.Fatalf("GetSocketStates failed: %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("GetSocketStates returned %d entries, want 1", len(states))
	}
	if states[0].State != "LISTEN" {
		t.Errorf("GetSocketStates state = %q, want LISTEN", states[0].State)
	}
	if states[0].LocalAddr != "0.0.0.0" {
		t.Errorf("GetSocketStates local addr = %q, want 0.0.0.0", states[0].LocalAddr)
	}
	if states[0].Explanation == "" {
		t.Error("GetSocketStates should include explanation")
	}
}

func TestGetSocketStateForPort(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("netstat", "-an", "-p", "tcp").
		Return([]byte("tcp4  0  0  *.8080  *.*  LISTEN\n"), nil)

	state := GetSocketStateForPort(8080)
	if state == nil {
		t.Fatal("GetSocketStateForPort returned nil")
	}
	if state.State != "LISTEN" {
		t.Errorf("GetSocketStateForPort state = %q, want LISTEN", state.State)
	}
}

func TestCountSocketsByState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	netstatOutput := `Active Internet connections
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
tcp4       0      0  *.8080               *.*                    LISTEN
tcp4       0      0  *.8080               *.*                    LISTEN
tcp4       0      0  127.0.0.1.8080       127.0.0.1.55555        TIME_WAIT`

	mockExec.EXPECT().Run("netstat", "-an", "-p", "tcp").
		Return([]byte(netstatOutput), nil)

	counts := CountSocketsByState(8080)
	if counts["LISTEN"] != 2 {
		t.Errorf("LISTEN count = %d, want 2", counts["LISTEN"])
	}
	if counts["TIME_WAIT"] != 1 {
		t.Errorf("TIME_WAIT count = %d, want 1", counts["TIME_WAIT"])
	}
}

func TestParseNetstatAddr(t *testing.T) {
	tests := []struct {
		input    string
		wantAddr string
		wantPort int
	}{
		{"127.0.0.1.8080", "127.0.0.1", 8080},
		{"*.80", "0.0.0.0", 80},
		{"[::]:8080", "::", 8080},
		{"", "", 0},
		{"[]:8080", "", 0},
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("netstat", "-an", "-p", "tcp").
		Return([]byte("tcp4  0  0  *.8080  *.*  LISTEN\n"), nil)

	sockets, err := readListeningSocketsNetstat()
	if err != nil {
		t.Fatalf("readListeningSocketsNetstat failed: %v", err)
	}
	if len(sockets) != 1 {
		t.Fatalf("readListeningSocketsNetstat returned %d sockets, want 1", len(sockets))
	}
	for _, s := range sockets {
		if s.Port != 8080 {
			t.Errorf("socket port = %d, want 8080", s.Port)
		}
		if s.Address != "0.0.0.0" {
			t.Errorf("socket address = %q, want 0.0.0.0", s.Address)
		}
	}
}

func TestGetCommandLine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("ps", "-p", "123", "-o", "args=").
		Return([]byte("/usr/bin/test --flag\n"), nil)

	cmdline := getCommandLine(123)
	if cmdline != "/usr/bin/test --flag" {
		t.Errorf("getCommandLine = %q", cmdline)
	}
}

func TestGetEnvironment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("ps", "-p", "123", "-E", "-o", "command=").
		Return([]byte("/bin/test HOME=/Users/test\n"), nil)

	env := getEnvironment(123)
	if len(env) != 1 || env[0] != "HOME=/Users/test" {
		t.Fatalf("getEnvironment = %v, want [HOME=/Users/test]", env)
	}
}

func TestGetWorkingDirectory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("lsof", "-a", "-p", "123", "-d", "cwd", "-F", "n").
		Return([]byte("p123\nn/Users/test/project\n"), nil)

	cwd := getWorkingDirectory(123)
	if cwd != "/Users/test/project" {
		t.Errorf("getWorkingDirectory = %q", cwd)
	}
}

func TestDetectContainer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("ps", "-p", "123", "-o", "args=").
		Return([]byte("/usr/bin/docker run nginx\n"), nil)

	container := detectContainer(123)
	if container != "docker" {
		t.Errorf("detectContainer = %q, want docker", container)
	}
}

func TestDetectLaunchdService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("launchctl", "blame", "123").
		Return([]byte("com.apple.Safari\n"), nil)

	svc := detectLaunchdService(123)
	if svc != "com.apple.Safari" {
		t.Errorf("detectLaunchdService = %q", svc)
	}
}

func TestCheckResourceUsage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("ps", "-p", "123", "-o", "pcpu=,rss=").
		Return([]byte("95.0 2097152\n"), nil)

	health := checkResourceUsage(123, "healthy")
	if health != "high-cpu" {
		t.Errorf("checkResourceUsage = %q, want high-cpu", health)
	}
}

func TestGetThermalState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("pmset", "-g", "therm").
		Return([]byte("CPU_Speed_Limit = 100\n"), nil)

	state := getThermalState()
	if state != "" {
		t.Errorf("getThermalState = %q, want empty", state)
	}
}

func TestGetEnergyImpact(t *testing.T) {
	impact := GetEnergyImpact(123)
	if impact != "" {
		t.Errorf("GetEnergyImpact = %q, want empty", impact)
	}
}
