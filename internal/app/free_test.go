package app

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/internal/target"
)

func TestFreeCommandKillsEveryOwnerAndVerifiesRelease(t *testing.T) {
	resolveCalls := 0
	var killed []int
	runner := freePortRunner{
		resolve: func(port int) ([]int, error) {
			resolveCalls++
			if port != 3000 {
				t.Fatalf("port = %d, want 3000", port)
			}
			if resolveCalls == 1 {
				return []int{42, 84}, nil
			}
			return nil, target.ErrPortNotBound
		},
		kill: func(pid int) error {
			killed = append(killed, pid)
			return nil
		},
		wait:    func(time.Duration) {},
		retries: 2,
	}
	cmd := newFreeCommand(runner)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"3000"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !reflect.DeepEqual(killed, []int{42, 84}) {
		t.Fatalf("killed = %v, want [42 84]", killed)
	}
	if got := out.String(); !strings.Contains(got, "Port 3000 released") {
		t.Fatalf("output = %q", got)
	}
}

func TestFreeCommandAlreadyFreeIsIdempotent(t *testing.T) {
	cmd := newFreeCommand(freePortRunner{
		resolve: func(int) ([]int, error) { return nil, target.ErrPortNotBound },
		kill:    func(int) error { t.Fatal("kill called"); return nil },
		wait:    func(time.Duration) {},
		retries: 1,
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"8080"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := out.String(); got != "Port 8080 is already free\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestFreeCommandReportsSupervisorRestart(t *testing.T) {
	cmd := newFreeCommand(freePortRunner{
		resolve: func(int) ([]int, error) { return []int{99}, nil },
		kill:    func(int) error { return nil },
		wait:    func(time.Duration) {},
		retries: 2,
	})
	cmd.SetArgs([]string{"3000"})
	err := cmd.Execute()
	var exitErr *exitCodeError
	if !errors.As(err, &exitErr) || exitErr.code != ExitInternalError {
		t.Fatalf("error = %v, want ExitInternalError", err)
	}
	if !strings.Contains(err.Error(), "restarted by a supervisor") {
		t.Fatalf("error = %v", err)
	}
}

func TestFreeCommandAttemptsEveryOwnerAfterFailure(t *testing.T) {
	var attempted []int
	cmd := newFreeCommand(freePortRunner{
		resolve: func(int) ([]int, error) { return []int{10, 20}, nil },
		kill: func(pid int) error {
			attempted = append(attempted, pid)
			if pid == 10 {
				return errors.New("denied")
			}
			return nil
		},
		wait:    func(time.Duration) {},
		retries: 1,
	})
	cmd.SetArgs([]string{"3000"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute() error = nil")
	}
	if want := []int{10, 20}; !reflect.DeepEqual(attempted, want) {
		t.Fatalf("attempted = %v, want %v", attempted, want)
	}
}

func TestFreeCommandRejectsInvalidPort(t *testing.T) {
	cmd := newFreeCommand(freePortRunner{})
	cmd.SetArgs([]string{"70000"})
	err := cmd.Execute()
	var exitErr *exitCodeError
	if !errors.As(err, &exitErr) || exitErr.code != ExitInvalidInput {
		t.Fatalf("error = %v, want ExitInvalidInput", err)
	}
}

func TestFreeCommandRequiresPort(t *testing.T) {
	cmd := newFreeCommand(freePortRunner{})
	cmd.SetArgs(nil)
	err := cmd.Execute()
	var exitErr *exitCodeError
	if !errors.As(err, &exitErr) || exitErr.code != ExitInvalidInput {
		t.Fatalf("error = %v, want ExitInvalidInput", err)
	}
}
