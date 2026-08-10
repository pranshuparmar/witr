//go:build linux || darwin || freebsd || windows

package app

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/internal/target"
	"github.com/spf13/cobra"
)

type freePortRunner struct {
	resolve func(int) ([]int, error)
	kill    func(int) error
	wait    func(time.Duration)
	retries int
}

func defaultFreePortRunner() freePortRunner {
	return freePortRunner{
		resolve: target.ResolveBoundPort,
		kill: func(pid int) error {
			process, err := os.FindProcess(pid)
			if err != nil {
				return err
			}
			return process.Kill()
		},
		wait:    time.Sleep,
		retries: 10,
	}
}

func newFreeCommand(runner freePortRunner) *cobra.Command {
	return &cobra.Command{
		Use:   "free <port>",
		Short: "Force-release a local port",
		Long:  "Force-release a local port by forcibly terminating every process that owns a TCP listener or bound UDP socket.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return withExitCode(ExitInvalidInput, fmt.Errorf("usage: witr free <port>"))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(strings.TrimSpace(args[0]))
			if err != nil || port < 1 || port > 65535 {
				return withExitCode(ExitInvalidInput, fmt.Errorf("invalid port: must be between 1 and 65535"))
			}

			pids, err := runner.resolve(port)
			if errors.Is(err, target.ErrPortNotBound) {
				fmt.Fprintf(cmd.OutOrStdout(), "Port %d is already free\n", port)
				return nil
			}
			if err != nil {
				code := ExitInternalError
				if errors.Is(err, target.ErrUnsupported) {
					code = ExitInvalidInput
				}
				return withExitCode(code, err)
			}

			var killErrors []error
			permissionDenied := false
			for _, pid := range pids {
				if err := runner.kill(pid); err != nil {
					// A process can exit between lookup and kill. It is already
					// gone, so let verification decide whether the port is free.
					if errors.Is(err, os.ErrProcessDone) || os.IsNotExist(err) {
						continue
					}
					if errors.Is(err, os.ErrPermission) {
						permissionDenied = true
					}
					killErrors = append(killErrors, fmt.Errorf("kill PID %d: %w", pid, err))
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Killed PID %d\n", pid)
			}
			for attempt := 0; attempt < runner.retries; attempt++ {
				runner.wait(50 * time.Millisecond)
				remaining, checkErr := runner.resolve(port)
				if errors.Is(checkErr, target.ErrPortNotBound) {
					if len(killErrors) > 0 {
						break
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Port %d released\n", port)
					return nil
				}
				if checkErr != nil {
					return withExitCode(ExitInternalError, fmt.Errorf("verify port %d: %w", port, checkErr))
				}
				pids = remaining
			}
			if len(killErrors) > 0 {
				code := ExitInternalError
				if permissionDenied {
					code = ExitPermission
				}
				return withExitCode(code, fmt.Errorf("failed to release port %d (retry with elevated privileges if needed): %w", port, errors.Join(killErrors...)))
			}

			return withExitCode(ExitInternalError, fmt.Errorf("port %d is still occupied by PID(s) %s; the process may have been restarted by a supervisor", port, joinPIDs(pids)))
		},
	}
}

func joinPIDs(pids []int) string {
	values := make([]string, len(pids))
	for i, pid := range pids {
		values[i] = strconv.Itoa(pid)
	}
	return strings.Join(values, ", ")
}

func init() {
	rootCmd.AddCommand(newFreeCommand(defaultFreePortRunner()))
}
