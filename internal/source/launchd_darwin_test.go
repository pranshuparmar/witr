//go:build darwin

package source

import (
	"errors"
	"testing"

	"github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/internal/proc/mocks"
	"github.com/pranshuparmar/witr/pkg/model"
	"go.uber.org/mock/gomock"
)

func TestDetectLaunchd(t *testing.T) {
	t.Run("no launchd", func(t *testing.T) {
		got := detectLaunchd([]model.Process{{PID: 100, Command: "bash"}})
		if got != nil {
			t.Fatalf("detectLaunchd() = %v, want nil", got)
		}
	})

	t.Run("launchd fallback when details fail", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockExec := mocks.NewMockExecutor(ctrl)
		proc.SetExecutor(mockExec)
		defer proc.ResetExecutor()

		mockExec.EXPECT().Run("launchctl", "blame", "100").Return(nil, errors.New("not found"))

		got := detectLaunchd([]model.Process{{PID: 1, Command: "launchd"}, {PID: 100, Command: "myapp"}})
		if got == nil {
			t.Fatal("detectLaunchd() = nil, want non-nil")
		}
		if got.Type != model.SourceLaunchd || got.Name != "launchd" {
			t.Fatalf("detectLaunchd() = %v, want Type=launchd Name=launchd", got)
		}
	})

	t.Run("launchd with label", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockExec := mocks.NewMockExecutor(ctrl)
		proc.SetExecutor(mockExec)
		defer proc.ResetExecutor()

		mockExec.EXPECT().Run("launchctl", "blame", "200").Return([]byte("system/com.test.service.unique123"), nil)

		got := detectLaunchd([]model.Process{{PID: 1, Command: "launchd"}, {PID: 200, Command: "myapp"}})
		if got == nil {
			t.Fatal("detectLaunchd() = nil, want non-nil")
		}
		if got.Type != model.SourceLaunchd || got.Name != "com.test.service.unique123" {
			t.Fatalf("detectLaunchd() = %v, want Name com.test.service.unique123", got)
		}
		if got.Details["type"] != "Launch Daemon" {
			t.Fatalf("detectLaunchd() type detail = %q, want Launch Daemon", got.Details["type"])
		}
	})
}

func TestDetectSystemd(t *testing.T) {
	got := detectSystemd([]model.Process{{Command: "systemd"}})
	if got != nil {
		t.Error("detectSystemd should always return nil on Darwin")
	}
}

func TestDetectContainerSource(t *testing.T) {
	got := detectContainer([]model.Process{{PID: 0, Command: "bash"}})
	if got != nil {
		t.Fatalf("detectContainer() = %v, want nil", got)
	}
}
