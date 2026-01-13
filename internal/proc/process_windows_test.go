//go:build windows

package proc

import (
	"fmt"
	"testing"

	"github.com/pranshuparmar/witr/internal/proc/mocks"
	"go.uber.org/mock/gomock"
)

func TestReadProcessWindows(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	pid := 1234
	tasklistOut := "\"myapp.exe\",\"1234\",\"Console\",\"1\",\"10,240 K\""
	cmdLineOut := `CommandLine="C:\myapp.exe" --run
CreationDate=20240101120000.000000+000
ExecutablePath=C:\myapp.exe
ParentProcessId=999
Status=Running
`
	netstatOut := `  TCP    0.0.0.0:8080           0.0.0.0:0              LISTENING       1234`

	// 1. tasklist
	mockExec.EXPECT().Run("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").
		Return([]byte(tasklistOut), nil)

	// 2. wmic
	mockExec.EXPECT().Run("wmic", "process", "where", fmt.Sprintf("processid=%d", pid), "get", "CommandLine,CreationDate,ExecutablePath,ParentProcessId,Status", "/format:list").
		Return([]byte(cmdLineOut), nil)

	// 3. netstat (via GetListeningPortsForPID)
	mockExec.EXPECT().Run("netstat", "-ano").
		Return([]byte(netstatOut), nil)

	p, err := ReadProcess(pid)
	if err != nil {
		t.Fatalf("ReadProcess failed: %v", err)
	}

	if p.PID != pid {
		t.Errorf("PID = %d, want %d", p.PID, pid)
	}
	if p.Command != "myapp.exe" {
		t.Errorf("Command = %q, want myapp.exe", p.Command)
	}
	if p.PPID != 999 {
		t.Errorf("PPID = %d, want 999", p.PPID)
	}
	if p.Cmdline != "\"C:\\myapp.exe\" --run" {
		t.Errorf("Cmdline = %q", p.Cmdline)
	}
	if len(p.ListeningPorts) != 1 || p.ListeningPorts[0] != 8080 {
		t.Errorf("ListeningPorts = %v, want [8080]", p.ListeningPorts)
	}
}

func TestReadProcessWindowsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("tasklist", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]byte("INFO: No tasks are running specified by the criteria."), nil)

	_, err := ReadProcess(9999)
	if err == nil {
		t.Error("ReadProcess should fail if process not found")
	}
}
