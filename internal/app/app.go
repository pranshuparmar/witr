//go:build linux || darwin || freebsd || windows

package app

import (
	"errors"
	"fmt"
	"os"
	"strconv"
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

// To embed version, commit, and build date, use:

var rootCmd = &cobra.Command{
	Use:   "witr [process name]",
	Short: "Why is this running?",
	Long:  "witr explains why a process or port is running by tracing its ancestry.",
	Args:  cobra.MaximumNArgs(1),
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd:  false,
		DisableDefaultCmd: false,
		DisableNoDescFlag: false,
	},
	Example: _genExamples(),
	RunE:    runApp,
}

func _genExamples() string {

	return `
  # Inspect a running process by name
  witr nginx

  # Look up a process by PID
  witr --pid 1234

  # Find the process listening on a specific port
  witr --port 5432

  # Find the process holding a lock on a file
  witr --file /var/lib/dpkg/lock

  # Inspect a process by name with exact matching (no fuzzy search)
  witr bun --exact

  # Show the full process ancestry (who started whom)
  witr postgres --tree

  # Show only warnings (suspicious env, arguments, parents)
  witr docker --warnings

  # Display only environment variables of the process
  witr node --env

  # Short, single-line output (useful for scripts)
  witr sshd --short

  # Disable colorized output (CI or piping)
  witr redis --no-color

  # Output machine-readable JSON
  witr chrome --json

  # Show extended process information (memory, I/O, file descriptors)
  witr mysql --verbose

  # Combine flags: inspect port, show environment variables, output JSON
  witr --port 8080 --env --json
`
}

func Execute() {

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	if version == "" {
		version = "v0.0.0-dev"
	}
	if commit == "" {
		commit = "unknown"
	}
	if buildDate == "" {
		buildDate = "unknown"
	}

	rootCmd.InitDefaultCompletionCmd()

	rootCmd.Version = version

	rootCmd.SetVersionTemplate(fmt.Sprintf("witr {{.Version}} (commit %s, built %s)\n", commit, buildDate))
	rootCmd.SetErr(output.NewSafeTerminalWriter(os.Stderr))

	rootCmd.Flags().StringP("pid", "p", "", "pid to look up")
	rootCmd.Flags().StringP("port", "o", "", "port to look up")
	rootCmd.Flags().StringP("file", "f", "", "file path to find process for")
	rootCmd.Flags().BoolP("short", "s", false, "show only ancestry")
	rootCmd.Flags().BoolP("tree", "t", false, "show only ancestry as a tree")
	rootCmd.Flags().Bool("json", false, "show result as JSON")
	rootCmd.Flags().Bool("warnings", false, "show only warnings")
	rootCmd.Flags().Bool("no-color", false, "disable colorized output")
	rootCmd.Flags().Bool("env", false, "show environment variables for the process")
	rootCmd.Flags().Bool("verbose", false, "show extended process information")
	rootCmd.Flags().BoolP("exact", "x", false, "use exact name matching (no substring search)")

}

func runApp(cmd *cobra.Command, args []string) error {
	envFlag, _ := cmd.Flags().GetBool("env")
	pidFlag, _ := cmd.Flags().GetString("pid")
	portFlag, _ := cmd.Flags().GetString("port")
	fileFlag, _ := cmd.Flags().GetString("file")
	// Show help if no arguments or relevant flags are provided
	if !envFlag && pidFlag == "" && portFlag == "" && fileFlag == "" && len(args) == 0 {
		cmd.Help()
		return nil
	}
	shortFlag, _ := cmd.Flags().GetBool("short")
	treeFlag, _ := cmd.Flags().GetBool("tree")
	jsonFlag, _ := cmd.Flags().GetBool("json")
	warnFlag, _ := cmd.Flags().GetBool("warnings")
	noColorFlag, _ := cmd.Flags().GetBool("no-color")
	verboseFlag, _ := cmd.Flags().GetBool("verbose")
	exactFlag, _ := cmd.Flags().GetBool("exact")

	outw := cmd.OutOrStdout()
	outp := output.NewPrinter(outw)

	if envFlag {
		var t model.Target
		switch {
		case pidFlag != "":
			t = model.Target{Type: model.TargetPID, Value: pidFlag}
		case portFlag != "":
			t = model.Target{Type: model.TargetPort, Value: portFlag}
		case fileFlag != "":
			t = model.Target{Type: model.TargetFile, Value: fileFlag}
		case len(args) > 0:
			t = model.Target{Type: model.TargetName, Value: args[0]}
		default:
			return fmt.Errorf("must specify --pid, --port, --file, or a process name")
		}

		pids, err := target.Resolve(t, exactFlag)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}
		if len(pids) > 1 {
			cmd.SilenceErrors = true
			outp.Print("Multiple matching processes found:\n\n")
			for i, pid := range pids {
				proc, err := procpkg.ReadProcess(pid)
				var command, cmdline string
				if err != nil {
					command = "unknown"
					cmdline = procpkg.GetCmdline(pid)
				} else {
					command = proc.Command
					cmdline = proc.Cmdline
				}
				if !noColorFlag {
					outp.Printf("[%d] %s%s%s (%spid %d%s)\n    %s\n",
						i+1, output.ColorGreen, command, output.ColorReset,
						output.ColorBold, pid, output.ColorReset,
						cmdline)
				} else {
					outp.Printf("[%d] %s (pid %d)\n    %s\n", i+1, command, pid, cmdline)
				}
			}
			outp.Println("\nRe-run with:")
			outp.Println("  witr --pid <pid> --env")
			return fmt.Errorf("multiple processes found")
		}
		pid := pids[0]
		procInfo, err := procpkg.ReadProcess(pid)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}

		resEnv := model.Result{
			Process:  procInfo,
			Ancestry: []model.Process{procInfo},
		}

		if jsonFlag {
			importJSON, err := output.ToEnvJSON(resEnv)
			if err != nil {
				return fmt.Errorf("failed to generate json output: %w", err)
			}
			fmt.Fprintln(outw, importJSON)
		} else {
			output.RenderEnvOnly(outw, resEnv, !noColorFlag)
		}
		return nil
	}

	var t model.Target

	switch {
	case pidFlag != "":
		t = model.Target{Type: model.TargetPID, Value: pidFlag}
	case portFlag != "":
		t = model.Target{Type: model.TargetPort, Value: portFlag}
	case fileFlag != "":
		t = model.Target{Type: model.TargetFile, Value: fileFlag}
	case len(args) > 0:
		t = model.Target{Type: model.TargetName, Value: args[0]}
	default:
		return fmt.Errorf("must specify --pid, --port, --file, or a process name")
	}

	pids, err := target.Resolve(t, exactFlag)
	if err == nil && len(pids) == 0 {
		err = fmt.Errorf("no matching process found")
	}
	if err != nil {
		errStr := err.Error()
		var errorMsg string
		if strings.Contains(errStr, "socket found but owning process not detected") {
			// Fallback: try Docker CLI to identify which container owns the port
			if t.Type == model.TargetPort {
				if portNum, convErr := strconv.Atoi(t.Value); convErr == nil {
					if match := procpkg.ResolveContainerByPort(portNum); match != nil {
						if jsonFlag {
							importJSON, jsonErr := output.DockerFallbackToJSON(t.Value, match)
							if jsonErr != nil {
								return fmt.Errorf("failed to generate json output: %w", jsonErr)
							}
							fmt.Fprintln(outw, importJSON)
						} else if shortFlag {
							output.RenderDockerFallbackShort(outw, t.Value, match, !noColorFlag)
						} else {
							output.RenderDockerFallback(outw, t.Value, match, !noColorFlag)
						}
						return nil
					}
				}
			}
			errorMsg = fmt.Sprintf("%s\n\nA socket was found for the port, but the owning process could not be detected.\nThis may be due to insufficient permissions. Try running with sudo:\n  sudo %s", errStr, strings.Join(os.Args, " "))
		} else {
			errorMsg = fmt.Sprintf("%s\n\nNo matching process or service found. Please check your query or try a different name/port/PID.\nFor usage and options, run: witr --help", errStr)
		}
		return errors.New(errorMsg)
	}

	if len(pids) > 1 {
		cmd.SilenceErrors = true
		outp.Print("Multiple matching processes found:\n\n")
		for i, pid := range pids {
			proc, err := procpkg.ReadProcess(pid)
			var command, cmdline string
			if err != nil {
				command = "unknown"
				cmdline = procpkg.GetCmdline(pid)
			} else {
				command = proc.Command
				cmdline = proc.Cmdline
			}
			if !noColorFlag {
				outp.Printf("[%d] %s%s%s (%spid %d%s)\n    %s\n",
					i+1, output.ColorGreen, command, output.ColorReset,
					output.ColorBold, pid, output.ColorReset,
					cmdline)
			} else {
				outp.Printf("[%d] %s (pid %d)\n    %s\n", i+1, command, pid, cmdline)
			}
		}
		outp.Println("\nRe-run with:")
		if envFlag {
			outp.Println("  witr --pid <pid> --env")
		} else {
			outp.Println("  witr --pid <pid>")
		}
		return fmt.Errorf("multiple processes found")
	}

	pid := pids[0]

	var systemdService string
	// If we found systemd (PID 1) listening on a port, try to identify the actual service unit.
	if t.Type == model.TargetPort && pid == 1 {
		if portNum, err := strconv.Atoi(t.Value); err == nil {
			if svc, err := procpkg.ResolveSystemdService(portNum); err == nil && svc != "" {
				systemdService = svc
			}
		}
	}

	ancestry, err := procpkg.ResolveAncestry(pid)
	if err != nil {
		errStr := err.Error()
		errorMsg := fmt.Sprintf("%s\n\nNo matching process or service found. Please check your query or try a different name/port/PID.\nFor usage and options, run: witr --help", errStr)
		return errors.New(errorMsg)
	}

	src := source.Detect(ancestry)

	var proc model.Process
	resolvedTarget := "unknown"
	if len(ancestry) > 0 {
		proc = ancestry[len(ancestry)-1]
		resolvedTarget = proc.Command
		if systemdService != "" {
			resolvedTarget = strings.TrimSuffix(systemdService, ".service")
		}
	}

	if verboseFlag && len(ancestry) > 0 {
		memInfo, ioStats, fileDescs, fdCount, fdLimit, children, threadCount, err := procpkg.ReadExtendedInfo(pid)
		if err == nil {
			proc.Memory = memInfo
			proc.IO = ioStats
			proc.FileDescs = fileDescs
			proc.FDCount = fdCount
			proc.FDLimit = fdLimit
			proc.Children = children
			proc.ThreadCount = threadCount
			ancestry[len(ancestry)-1] = proc
		}
	}

	var resCtx *model.ResourceContext
	var fileCtx *model.FileContext
	if verboseFlag {
		resCtx = procpkg.GetResourceContext(pid)
		fileCtx = procpkg.GetFileContext(pid)
	}

	var childProcesses []model.Process
	if (verboseFlag || treeFlag) && proc.PID > 0 {
		if children, err := procpkg.ResolveChildren(proc.PID); err == nil {
			childProcesses = children
		}
	}

	// Calculate restart count
	restartCount := 0
	if src.Type == model.SourceSystemd && src.Name != "" {
		if count, err := procpkg.GetSystemdRestartCount(src.Name); err == nil {
			restartCount = count
		}
	}

	res := model.Result{
		Target:          t,
		ResolvedTarget:  resolvedTarget,
		Process:         proc,
		RestartCount:    restartCount,
		Ancestry:        ancestry,
		Source:          src,
		Warnings:        source.Warnings(ancestry),
		ResourceContext: resCtx,
		FileContext:     fileCtx,
	}
	if len(childProcesses) > 0 {
		res.Children = childProcesses
	}

	// Add socket state info for port queries
	if t.Type == model.TargetPort {
		portNum := 0
		fmt.Sscanf(t.Value, "%d", &portNum)
		if portNum > 0 {
			res.SocketInfo = procpkg.GetSocketStateForPort(portNum)
			source.EnrichSocketInfo(res.SocketInfo)
		}
	}

	if jsonFlag {
		var importJSON string
		var err error

		if shortFlag {
			importJSON, err = output.ToShortJSON(res)
		} else if treeFlag {
			importJSON, err = output.ToTreeJSON(res)
		} else if warnFlag {
			importJSON, err = output.ToWarningsJSON(res)
		} else {
			importJSON, err = output.ToJSON(res)
		}

		if err != nil {
			return fmt.Errorf("failed to generate json output: %w", err)
		}
		fmt.Fprintln(outw, importJSON)
	} else if warnFlag {
		output.RenderWarnings(outw, res, !noColorFlag)
	} else if treeFlag {
		output.PrintTree(outw, res.Ancestry, res.Children, !noColorFlag)
	} else if shortFlag {
		output.RenderShort(outw, res, !noColorFlag)
	} else {
		output.RenderStandard(outw, res, !noColorFlag, verboseFlag)
	}
	return nil
}

func Root() *cobra.Command { return rootCmd }

func SetVersionBuildCommitString(Version string, Commit string, BuildDate string) {
	version = Version
	commit = Commit
	buildDate = BuildDate

	if version == "" {
		version = "v0.0.0-dev"
	}
	if commit == "" {
		commit = "unknown"
	}
	if buildDate == "" {
		buildDate = "unknown"
	}

	rootCmd.Version = version

	rootCmd.SetVersionTemplate(fmt.Sprintf("witr {{.Version}} (commit %s, built %s)\n", commit, buildDate))

	rootCmd.SilenceUsage = true
}
