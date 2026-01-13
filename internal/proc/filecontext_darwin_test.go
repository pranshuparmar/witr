//go:build darwin

package proc

import (
	"errors"
	"strconv"
	"testing"

	"github.com/pranshuparmar/witr/internal/proc/mocks"
	"go.uber.org/mock/gomock"
)

func TestGetFileContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	// Mock getOpenFileCount lsof
	// high file usage to ensure return
	lsofOut := "p123\n"
	for i := 0; i < 600; i++ {
		lsofOut += "f" + strconv.Itoa(i) + "\n"
	}
	mockExec.EXPECT().Run("lsof", "-p", "123").Return([]byte(lsofOut), nil)

	// Mock getFileLimit launchctl
	mockExec.EXPECT().Run("launchctl", "limit", "maxfiles").
		Return([]byte("maxfiles    1000            unlimited"), nil)

	// Mock getLockedFiles lsof -F fn
	mockExec.EXPECT().Run("lsof", "-p", "123", "-F", "fn").
		Return([]byte("p123\nf10\nn/tmp/file.lock\n"), nil)

	// Mock getLockedFiles lsof (second call)
	mockExec.EXPECT().Run("lsof", "-p", "123").
		Return([]byte(""), nil)

	ctx := GetFileContext(123)
	if ctx == nil {
		t.Fatal("GetFileContext returned nil")
	}
	if ctx.OpenFiles != 600 {
		t.Errorf("OpenFiles = %d, want 600", ctx.OpenFiles)
	}
	if ctx.FileLimit != 1000 {
		t.Errorf("FileLimit = %d, want 1000", ctx.FileLimit)
	}
	if len(ctx.LockedFiles) != 1 || ctx.LockedFiles[0] != "/tmp/file.lock" {
		t.Errorf("LockedFiles = %v, want [/tmp/file.lock]", ctx.LockedFiles)
	}
}

func TestGetFileLimitFallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	mockExec.EXPECT().Run("launchctl", "limit", "maxfiles").
		Return(nil, errors.New("failed"))

	limit := getFileLimit(123)
	if limit != 256 {
		t.Errorf("getFileLimit fallback = %d, want 256", limit)
	}
}
