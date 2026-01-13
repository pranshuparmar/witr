//go:build darwin

package proc

import (
	"errors"
	"testing"

	"github.com/pranshuparmar/witr/internal/proc/mocks"
	"go.uber.org/mock/gomock"
)

func TestListProcessSnapshot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	psOut := "  123   1 /sbin/launchd\n  456 123 /usr/libexec/xpcproxy\n"

	mockExec.EXPECT().Run("ps", "-axo", "pid=,ppid=,comm=").
		Return([]byte(psOut), nil)

	procs, err := listProcessSnapshot()
	if err != nil {
		t.Fatalf("listProcessSnapshot failed: %v", err)
	}

	if len(procs) != 2 {
		t.Fatalf("Got %d processes, want 2", len(procs))
	}

	if procs[0].PID != 123 || procs[0].PPID != 1 || procs[0].Command != "/sbin/launchd" {
		t.Errorf("Process 0 mismatch: %+v", procs[0])
	}
	if procs[1].PID != 456 || procs[1].PPID != 123 || procs[1].Command != "/usr/libexec/xpcproxy" {
		t.Errorf("Process 1 mismatch: %+v", procs[1])
	}
}

func TestListProcessSnapshotError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("ps", gomock.Any(), gomock.Any()).
		Return(nil, errors.New("ps failed"))

	_, err := listProcessSnapshot()
	if err == nil {
		t.Error("listProcessSnapshot should have failed")
	}
}
