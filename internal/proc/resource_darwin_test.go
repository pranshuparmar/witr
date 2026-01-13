//go:build darwin

package proc

import (
	"errors"
	"testing"

	"github.com/pranshuparmar/witr/internal/proc/mocks"
	"go.uber.org/mock/gomock"
)

func TestGetResourceContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	// Mock checkPreventsSleep
	mockExec.EXPECT().Run("pmset", "-g", "assertions").
		Return([]byte("pid 123(PreventUserIdleDisplaySleep)\n"), nil)

	// Mock getThermalState
	mockExec.EXPECT().Run("pmset", "-g", "therm").
		Return([]byte("CPU_Speed_Limit = 100\n"), nil)

	// Mock getCPUAndMemoryUsage (ps)
	mockExec.EXPECT().Run("ps", "-p", "123", "-o", "%cpu=,rss=").
		Return([]byte("10.5 102400"), nil)

	ctx := GetResourceContext(123)
	if ctx == nil {
		t.Fatal("GetResourceContext returned nil")
	}
	if !ctx.PreventsSleep {
		t.Error("PreventsSleep should be true")
	}
	if ctx.CPUUsage != 10.5 {
		t.Errorf("CPUUsage = %f, want 10.5", ctx.CPUUsage)
	}
	if ctx.MemoryUsage != 102400*1024 {
		t.Errorf("MemoryUsage = %d, want %d", ctx.MemoryUsage, 102400*1024)
	}
}

func TestCheckPreventsSleep(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"match", "pid 123(PreventUserIdleSystemSleep)\n", true},
		{"no match", "pid 456(PreventUserIdleSystemSleep)\n", false},
		{"no assertion", "pid 123(SomeOtherAssertion)\n", false},
		{"error", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "error" {
				mockExec.EXPECT().Run("pmset", "-g", "assertions").
					Return(nil, errors.New("failed"))
			} else {
				mockExec.EXPECT().Run("pmset", "-g", "assertions").
					Return([]byte(tt.output), nil)
			}

			if got := checkPreventsSleep(123); got != tt.want {
				t.Errorf("checkPreventsSleep() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetThermalStateParsing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"heavy throttle", "CPU_Speed_Limit = 40\n", "Heavy throttling"},
		{"moderate throttle", "CPU_Speed_Limit = 60\n", "Moderate throttling"},
		{"light throttle", "CPU_Speed_Limit = 90\n", "Light throttling"},
		{"normal", "CPU_Speed_Limit = 100\n", ""},
		{"thermal level 1", "Thermal_Level = 1\n", "Moderate thermal pressure"},
		{"error", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "error" {
				mockExec.EXPECT().Run("pmset", "-g", "therm").
					Return(nil, errors.New("failed"))
			} else {
				mockExec.EXPECT().Run("pmset", "-g", "therm").
					Return([]byte(tt.output), nil)
			}

			if got := getThermalState(); got != tt.want {
				t.Errorf("getThermalState() = %q, want %q", got, tt.want)
			}
		})
	}
}
