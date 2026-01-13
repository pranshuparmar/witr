//go:build windows

package proc

import (
	"testing"

	"github.com/pranshuparmar/witr/internal/proc/mocks"
	"go.uber.org/mock/gomock"
)

func TestGetResourceContextWindows(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	wmicOut := `PercentProcessorTime=25
WorkingSetPrivate=104857600
`
	mockExec.EXPECT().Run("wmic", "path", "Win32_PerfFormattedData_PerfProc_Process", "where", "IDProcess=123", "get", "PercentProcessorTime,WorkingSetPrivate", "/format:list").
		Return([]byte(wmicOut), nil)

	ctx := GetResourceContext(123)
	if ctx == nil {
		t.Fatal("GetResourceContext returned nil")
	}
	if ctx.CPUUsage != 25.0 {
		t.Errorf("CPUUsage = %f, want 25.0", ctx.CPUUsage)
	}
	if ctx.MemoryUsage != 104857600 {
		t.Errorf("MemoryUsage = %d, want 104857600", ctx.MemoryUsage)
	}
}
