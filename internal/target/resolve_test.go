//go:build darwin

package target

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/internal/proc/mocks"
	"github.com/pranshuparmar/witr/pkg/model"
	"go.uber.org/mock/gomock"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		target  model.Target
		wantErr bool
	}{
		{"valid pid", model.Target{Type: model.TargetPID, Value: "1234"}, false},
		{"invalid pid", model.Target{Type: model.TargetPID, Value: "abc"}, true},
		{"empty pid", model.Target{Type: model.TargetPID, Value: ""}, true},
		{"invalid port", model.Target{Type: model.TargetPort, Value: "http"}, true},
		{"unknown type", model.Target{Type: "invalid", Value: "x"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Resolve(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidServiceLabel(t *testing.T) {
	tests := []struct {
		label string
		want  bool
	}{
		{"com.apple.Safari", true},
		{"org.nginx", true},
		{"my-service_123", true},
		{"a", true},
		{"", false},
		{"invalid/label", false},
		{"has space", false},
		{"has@special", false},
	}
	for _, tt := range tests {
		if got := isValidServiceLabel(tt.label); got != tt.want {
			t.Errorf("isValidServiceLabel(%q) = %v, want %v", tt.label, got, tt.want)
		}
	}
	longLabel := ""
	for i := 0; i < 300; i++ {
		longLabel += "a"
	}
	if isValidServiceLabel(longLabel) {
		t.Error("isValidServiceLabel should reject labels > 256 chars")
	}
}

func TestResolvePortUsesLsofAndReturnsLowestPID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	proc.SetExecutor(mockExec)
	defer proc.ResetExecutor()

	mockExec.EXPECT().Run("lsof", "-i", "TCP:8080", "-s", "TCP:LISTEN", "-n", "-P", "-t").
		Return([]byte("456\n123\n"), nil)

	pids, err := ResolvePort(8080)
	if err != nil {
		t.Fatalf("ResolvePort error = %v", err)
	}
	if len(pids) != 1 || pids[0] != 123 {
		t.Fatalf("ResolvePort returned %v, want [123]", pids)
	}
}

func TestResolvePortFallsBackToNetstat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	proc.SetExecutor(mockExec)
	defer proc.ResetExecutor()

	netstatOut := "tcp4 0 0 *.8080 *.* LISTEN 0 0 4321\n"

	gomock.InOrder(
		mockExec.EXPECT().Run("lsof", "-i", "TCP:8080", "-s", "TCP:LISTEN", "-n", "-P", "-t").
			Return(nil, errors.New("lsof failed")),
		mockExec.EXPECT().Run("netstat", "-anv", "-p", "tcp").
			Return([]byte(netstatOut), nil),
	)

	pids, err := ResolvePort(8080)
	if err != nil {
		t.Fatalf("ResolvePort fallback error = %v", err)
	}
	if len(pids) != 1 || pids[0] != 4321 {
		t.Fatalf("ResolvePort fallback returned %v, want [4321]", pids)
	}
}

func TestResolvePortNoListener(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	proc.SetExecutor(mockExec)
	defer proc.ResetExecutor()

	mockExec.EXPECT().Run("lsof", "-i", "TCP:8080", "-s", "TCP:LISTEN", "-n", "-P", "-t").
		Return([]byte("\n"), nil)

	_, err := ResolvePort(8080)
	if err == nil {
		t.Fatal("ResolvePort should fail when no listener is found")
	}
}

func TestResolvePortFallbackNoMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	proc.SetExecutor(mockExec)
	defer proc.ResetExecutor()

	gomock.InOrder(
		mockExec.EXPECT().Run("lsof", "-i", "TCP:8080", "-s", "TCP:LISTEN", "-n", "-P", "-t").
			Return(nil, errors.New("lsof failed")),
		mockExec.EXPECT().Run("netstat", "-anv", "-p", "tcp").
			Return([]byte("tcp4 0 0 *.22 *.* LISTEN 0 0 22\n"), nil),
	)

	_, err := ResolvePort(8080)
	if err == nil {
		t.Fatal("ResolvePort should fail when netstat finds no listener")
	}
}

func TestResolveNameMatchesProcessAndSkipsGrepAndWitr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	proc.SetExecutor(mockExec)
	defer proc.ResetExecutor()

	self := os.Getpid()
	parent := os.Getppid()
	pid := self + 1000
	if pid == parent {
		pid += 1000
	}
	grepPid := pid + 1
	if grepPid == parent {
		grepPid++
	}
	witrPid := grepPid + 1
	if witrPid == parent {
		witrPid++
	}

	psOut := fmt.Sprintf(" %d myapp /usr/bin/myapp --flag\n %d grep grep myapp\n %d sh /usr/bin/witr myapp\n", pid, grepPid, witrPid)
	psOut = fmt.Sprintf(" %d myapp /usr/bin/myapp --flag\n %d myapp /usr/bin/myapp --self\n %d grep grep myapp\n %d sh /usr/bin/witr myapp\n", pid, self, grepPid, witrPid)

	gomock.InOrder(
		mockExec.EXPECT().Run("ps", "-axo", "pid=,comm=,args=").Return([]byte(psOut), nil),
		mockExec.EXPECT().Run("launchctl", "print", "system/myapp").Return(nil, errors.New("not found")),
		mockExec.EXPECT().Run("launchctl", "print", "system/com.apple.myapp").Return(nil, errors.New("not found")),
		mockExec.EXPECT().Run("launchctl", "print", "system/org.myapp").Return(nil, errors.New("not found")),
		mockExec.EXPECT().Run("launchctl", "print", "system/io.myapp").Return(nil, errors.New("not found")),
	)

	pids, err := ResolveName("myapp")
	if err != nil {
		t.Fatalf("ResolveName error = %v", err)
	}
	if len(pids) != 1 || pids[0] != pid {
		t.Fatalf("ResolveName returned %v, want [%d]", pids, pid)
	}
}

func TestResolveNameServiceOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	proc.SetExecutor(mockExec)
	defer proc.ResetExecutor()

	psOut := " 1 launchd /sbin/launchd\n"
	serviceOut := "pid = 4321\n"

	gomock.InOrder(
		mockExec.EXPECT().Run("ps", "-axo", "pid=,comm=,args=").Return([]byte(psOut), nil),
		mockExec.EXPECT().Run("launchctl", "print", "system/myservice").Return([]byte(serviceOut), nil),
	)

	pids, err := ResolveName("myservice")
	if err != nil {
		t.Fatalf("ResolveName error = %v", err)
	}
	if len(pids) != 1 || pids[0] != 4321 {
		t.Fatalf("ResolveName returned %v, want [4321]", pids)
	}
}

func TestResolveLaunchdServicePIDRejectsInvalidName(t *testing.T) {
	_, err := resolveLaunchdServicePID("invalid/name")
	if err == nil {
		t.Fatal("resolveLaunchdServicePID should reject invalid names")
	}
}

func TestResolveLaunchdServicePIDParsesPID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	proc.SetExecutor(mockExec)
	defer proc.ResetExecutor()

	mockExec.EXPECT().Run("launchctl", "print", "system/myservice").
		Return([]byte("pid = 555\n"), nil)

	pid, err := resolveLaunchdServicePID("myservice")
	if err != nil {
		t.Fatalf("resolveLaunchdServicePID error = %v", err)
	}
	if pid != 555 {
		t.Fatalf("resolveLaunchdServicePID pid = %d, want 555", pid)
	}
}
