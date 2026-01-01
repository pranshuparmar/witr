//go:build linux || darwin

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pranshuparmar/witr/internal/output"
	procpkg "github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/internal/source"
	"github.com/pranshuparmar/witr/internal/target"
	"github.com/pranshuparmar/witr/pkg/model"
	"github.com/spf13/cobra"
)

var (
	version   = ""
	commit    = ""
	buildDate = ""
)

func main() {
	// To embed version, commit, and build date, use:
	// go build -ldflags "-X main.version=v0.1.0 -X main.commit=$(git rev-parse --short HEAD) -X 'main.buildDate=$(date +%Y-%m-%d)'" -o witr ./cmd/witr
	if version == "" {
		version = "v0.0.0-dev"
	}
	if commit == "" {
		commit = "unknown"
	}
	if buildDate == "" {
		buildDate = "unknown"
	}

	rootCmd := &cobra.Command{
		Use:   "witr [process name]",
		Short: "Why is this running?",
		Long:  "witr explains why a process or port is running by tracing its ancestry.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envFlag, _ := cmd.Flags().GetBool("env")
			pidFlag, _ := cmd.Flags().GetString("pid")
			portFlag, _ := cmd.Flags().GetString("port")
			// Show help if no arguments or relevant flags are provided
			if !envFlag && pidFlag == "" && portFlag == "" && len(args) == 0 {
				cmd.Help()
				return nil
			}
			shortFlag, _ := cmd.Flags().GetBool("short")
			treeFlag, _ := cmd.Flags().GetBool("tree")
			jsonFlag, _ := cmd.Flags().GetBool("json")
			warnFlag, _ := cmd.Flags().GetBool("warnings")
			noColorFlag, _ := cmd.Flags().GetBool("no-color")

			if envFlag {
				var t model.Target
				switch {
				case pidFlag != "":
					t = model.Target{Type: model.TargetPID, Value: pidFlag}
				case portFlag != "":
					t = model.Target{Type: model.TargetPort, Value: portFlag}
				case len(args) > 0:
					t = model.Target{Type: model.TargetName, Value: args[0]}
				default:
					return fmt.Errorf("must specify --pid, --port, or a process name")
				}

				pids, err := target.Resolve(t)
				if err != nil {
					return fmt.Errorf("error: %v", err)
				}
				if len(pids) > 1 {
					fmt.Print("Multiple matching processes found:\n\n")
					for i, pid := range pids {
						cmdline := procpkg.GetCmdline(pid)
						fmt.Printf("[%d] PID %d   %s\n", i+1, pid, cmdline)
					}
					fmt.Println("\nRe-run with:")
					fmt.Println("  witr --pid <pid> --env")
					return fmt.Errorf("multiple processes found")
				}
				pid := pids[0]
				procInfo, err := procpkg.ReadProcess(pid)
				if err != nil {
					return fmt.Errorf("error: %v", err)
				}
				if jsonFlag {
					type envOut struct {
						Command string   `json:"Command"`
						Env     []string `json:"Env"`
					}
					out := envOut{Command: procInfo.Cmdline, Env: procInfo.Env}
					enc, _ := json.MarshalIndent(out, "", "  ")
					fmt.Println(string(enc))
				} else {
					output.RenderEnvOnly(procInfo, !noColorFlag)
				}
				return nil
			}

			var t model.Target

			switch {
			case pidFlag != "":
				t = model.Target{Type: model.TargetPID, Value: pidFlag}
			case portFlag != "":
				t = model.Target{Type: model.TargetPort, Value: portFlag}
			case len(args) > 0:
				t = model.Target{Type: model.TargetName, Value: args[0]}
			default:
				return fmt.Errorf("must specify --pid, --port, or a process name")
			}

			pids, err := target.Resolve(t)
			if err != nil {
				errStr := err.Error()
				var errorMsg string
				if strings.Contains(errStr, "socket found but owning process not detected") {
					errorMsg = fmt.Sprintf("%s\n\nA socket was found for the port, but the owning process could not be detected.\nThis may be due to insufficient permissions. Try running with sudo:\n  sudo %s", errStr, strings.Join(os.Args, " "))
				} else {
					errorMsg = fmt.Sprintf("%s\n\nNo matching process or service found. Please check your query or try a different name/port/PID.", errStr)
				}
				return errors.New(errorMsg)
			}

			if len(pids) > 1 {
				fmt.Print("Multiple matching processes found:\n\n")
				for i, pid := range pids {
					cmdline := procpkg.GetCmdline(pid)
					fmt.Printf("[%d] PID %d   %s\n", i+1, pid, cmdline)
				}
				fmt.Println("\nRe-run with:")
				fmt.Println("  witr --pid <pid>")
				return fmt.Errorf("multiple processes found")
			}

			pid := pids[0]

			ancestry, err := procpkg.ResolveAncestry(pid)
			if err != nil {
				fmt.Println()
				fmt.Println("Error:")
				fmt.Printf("  %s\n", err.Error())
				fmt.Println("\nNo matching process or service found. Please check your query or try a different name/port/PID.")
				fmt.Println("For usage and options, run: witr --help")
				os.Exit(1)
			}

			src := source.Detect(ancestry)

			var proc model.Process
			resolvedTarget := "unknown"
			if len(ancestry) > 0 {
				proc = ancestry[len(ancestry)-1]
				resolvedTarget = proc.Command
			}

			// Calculate restart count (consecutive same-command entries)
			restartCount := 0
			lastCmd := ""
			for _, procA := range ancestry {
				if procA.Command == lastCmd {
					restartCount++
				}
				lastCmd = procA.Command
			}

			res := model.Result{
				Target:         t,
				ResolvedTarget: resolvedTarget,
				Process:        proc,
				RestartCount:   restartCount,
				Ancestry:       ancestry,
				Source:         src,
				Warnings:       source.Warnings(ancestry),
			}

			// Add socket state info for port queries
			if t.Type == model.TargetPort {
				portNum := 0
				fmt.Sscanf(t.Value, "%d", &portNum)
				if portNum > 0 {
					res.SocketInfo = procpkg.GetSocketStateForPort(portNum)
				}
			}

			// Add resource context (thermal state, sleep prevention)
			res.ResourceContext = procpkg.GetResourceContext(pid)

			// Add file context (open files, locks)
			res.FileContext = procpkg.GetFileContext(pid)

			if jsonFlag {
				importJSON, _ := output.ToJSON(res)
				fmt.Println(importJSON)
			} else if warnFlag {
				output.RenderWarnings(res.Warnings, !noColorFlag)
			} else if treeFlag {
				output.PrintTree(res.Ancestry, !noColorFlag)
			} else if shortFlag {
				output.RenderShort(res, !noColorFlag)
			} else {
				output.RenderStandard(res, !noColorFlag)
			}
			return nil
		},
	}

	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("witr {{.Version}} (commit %s, built %s)\n", commit, buildDate))

	rootCmd.Flags().String("pid", "", "pid to look up")
	rootCmd.Flags().String("port", "", "port to look up")
	rootCmd.Flags().Bool("short", false, "show only ancestry")
	rootCmd.Flags().Bool("tree", false, "shows only ancestry as a tree")
	rootCmd.Flags().Bool("json", false, "show result as JSON")
	rootCmd.Flags().Bool("warnings", false, "show only warnings")
	rootCmd.Flags().Bool("no-color", false, "disable colorized output")
	rootCmd.Flags().Bool("env", false, "show environment variables for the process")

	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
