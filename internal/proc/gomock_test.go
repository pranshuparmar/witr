package proc

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/pranshuparmar/witr/internal/proc/mocks"
)

func TestReadProcessWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().
		Run("ps", "-p", "123", "-o", "pid=,ppid=,uid=,lstart=,state=,ucomm=").
		Return([]byte("  123  456  501 Mon Dec 30 10:00:00 2024 S test\n"), nil)

	mockExec.EXPECT().
		Run("ps", "-p", "123", "-o", "args=").
		Return([]byte("/usr/bin/test --flag\n"), nil)

	mockExec.EXPECT().
		Run("ps", "-p", "123", "-E", "-o", "command=").
		Return([]byte("/usr/bin/test HOME=/Users/test\n"), nil)

	mockExec.EXPECT().
		Run("lsof", "-a", "-p", "123", "-d", "cwd", "-F", "n").
		Return([]byte("p123\nn/Users/test/project\n"), nil)

	mockExec.EXPECT().
		Run("launchctl", "blame", "123").
		Return([]byte(""), nil)

	mockExec.EXPECT().
		Run("lsof", "-i", "TCP", "-s", "TCP:LISTEN", "-n", "-P", "-F", "pn").
		Return([]byte(""), nil)

	mockExec.EXPECT().
		Run("ps", "-p", "123", "-o", "pcpu=,rss=").
		Return([]byte("1.0 10240\n"), nil)

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
	if proc.Command != "test" {
		t.Errorf("Command = %s, want test", proc.Command)
	}
}

func TestReadProcessNotFoundWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().
		Run("ps", "-p", "99999", "-o", "pid=,ppid=,uid=,lstart=,state=,ucomm=").
		Return(nil, errors.New("process not found"))

	_, err := ReadProcess(99999)
	if err == nil {
		t.Error("expected error for nonexistent process")
	}
}

func TestGetSocketStatesWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	netstatOutput := `Active Internet connections
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
tcp4       0      0  *.8080               *.*                    LISTEN
tcp4       0      0  127.0.0.1.8080       127.0.0.1.49152        TIME_WAIT`

	mockExec.EXPECT().
		Run("netstat", "-an", "-p", "tcp").
		Return([]byte(netstatOutput), nil)

	states, err := GetSocketStates(8080)
	if err != nil {
		t.Fatalf("GetSocketStates failed: %v", err)
	}
	_ = states
}

func TestGetMSLDurationWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().
		Run("sysctl", "-n", "net.inet.tcp.msl").
		Return([]byte("15000\n"), nil)

	duration := GetMSLDuration()
	if duration != 15000 {
		t.Errorf("GetMSLDuration = %d, want 15000", duration)
	}
}

func TestGetMSLDurationDefaultWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().
		Run("sysctl", "-n", "net.inet.tcp.msl").
		Return(nil, errors.New("command failed"))

	duration := GetMSLDuration()
	if duration != 30000 {
		t.Errorf("GetMSLDuration = %d, want 30000 (default)", duration)
	}
}

func TestReadListeningSocketsWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	lsofOutput := `p1234
n*:8080
p5678
n127.0.0.1:3000`

	mockExec.EXPECT().
		Run("lsof", "-i", "TCP", "-s", "TCP:LISTEN", "-n", "-P", "-F", "pn").
		Return([]byte(lsofOutput), nil)

	sockets, err := readListeningSockets()
	if err != nil {
		t.Fatalf("readListeningSockets failed: %v", err)
	}
	if len(sockets) == 0 {
		t.Error("expected at least 1 socket")
	}
}

func TestGetCommandLineWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().
		Run("ps", "-p", "123", "-o", "args=").
		Return([]byte("/usr/bin/test --flag --option=value\n"), nil)

	cmdline := getCommandLine(123)
	if cmdline != "/usr/bin/test --flag --option=value" {
		t.Errorf("got %q", cmdline)
	}
}

func TestCheckResourceUsageWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().
		Run("ps", "-p", "123", "-o", "pcpu=,rss=").
		Return([]byte("95.0 2097152\n"), nil)

	health := checkResourceUsage(123, "healthy")
	if health != "high-cpu" {
		t.Errorf("got %q, want high-cpu", health)
	}
}

func TestCheckResourceUsageHighMemWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().
		Run("ps", "-p", "123", "-o", "pcpu=,rss=").
		Return([]byte("1.0 2097152\n"), nil)

	health := checkResourceUsage(123, "healthy")
	if health != "high-mem" {
		t.Errorf("got %q, want high-mem", health)
	}
}

func TestDetectLaunchdServiceWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().
		Run("launchctl", "blame", "123").
		Return([]byte("com.apple.Safari\n"), nil)

	svc := detectLaunchdService(123)
	if svc != "com.apple.Safari" {
		t.Errorf("got %q, want com.apple.Safari", svc)
	}
}
