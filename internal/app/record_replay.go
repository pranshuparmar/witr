package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/internal/pipeline"
	"github.com/pranshuparmar/witr/internal/record"
	"github.com/pranshuparmar/witr/internal/tui"
	"github.com/pranshuparmar/witr/pkg/model"
	"github.com/spf13/cobra"
)

var (
	recordOutputFlag   string
	recordIntervalFlag time.Duration
	recordCountFlag    int
	recordDurationFlag time.Duration

	replayFileFlag      string
	replayStepFlag      int
	replayTimestampFlag string
	replayAllFlag       bool
)

func newRecordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "record",
		Short: "Record periodic process snapshots to a file",
		Long:  "Periodically snapshots system state (processes, open ports, containers, locked files) to a file.",
		Example: `  # Record every 2 seconds to witr.jsonl
  witr record -i 2s -o witr.jsonl

  # Record 5 samples and exit
  witr record -i 1s -n 5 -o witr.jsonl

  # Record for 30 seconds
  witr record -i 2s -d 30s -o witr.jsonl`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outPath, _ := cmd.Flags().GetString("output")
			if outPath == "" {
				outPath = "witr.jsonl"
			}
			interval, _ := cmd.Flags().GetDuration("interval")
			count, _ := cmd.Flags().GetInt("count")
			duration, _ := cmd.Flags().GetDuration("duration")
			return executeRecord(cmd, outPath, interval, count, duration)
		},
	}

	cmd.Flags().StringVarP(&recordOutputFlag, "output", "o", "witr.jsonl", "output file destination for recorded stream")
	cmd.Flags().DurationVarP(&recordIntervalFlag, "interval", "i", 2*time.Second, "sample capture interval")
	cmd.Flags().IntVarP(&recordCountFlag, "count", "n", 0, "number of samples to capture (0 for continuous)")
	cmd.Flags().DurationVarP(&recordDurationFlag, "duration", "d", 0, "total recording duration (0 for unlimited)")

	return cmd
}

func newReplayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replay",
		Short: "Replay recorded process snapshots from a file",
		Long:  "Restores and replays captured process snapshots from a file, supporting live query flags.",
		Example: `  # Browse snapshots in interactive TUI
  witr replay -f witr.jsonl

  # Query process by name from recorded snapshots
  witr replay -f witr.jsonl nginx

  # Query process by PID with JSON output
  witr replay -f witr.jsonl --pid 1234 --json

  # Jump to specific snapshot step or timestamp
  witr replay -f witr.jsonl --step 2 nginx
  witr replay -f witr.jsonl --timestamp 2026-10-05T12:00:00Z nginx`,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			step, _ := cmd.Flags().GetInt("step")
			ts, _ := cmd.Flags().GetString("timestamp")
			all, _ := cmd.Flags().GetBool("all")
			return executeReplay(cmd, filePath, args, step, ts, all)
		},
	}

	cmd.Flags().StringVarP(&replayFileFlag, "file", "f", "", "path to recording log file to replay (required)")
	cmd.Flags().IntVar(&replayStepFlag, "step", -1, "0-based step index of snapshot to replay")
	cmd.Flags().StringVar(&replayTimestampFlag, "timestamp", "", "timestamp to seek in replay")
	cmd.Flags().BoolVar(&replayAllFlag, "all", false, "evaluate target across all snapshots")

	// Standard target and display flags
	cmd.Flags().StringSliceP("pid", "p", nil, "pid(s) to look up (repeatable)")
	cmd.Flags().StringSliceP("port", "o", nil, "port(s) to look up (repeatable)")
	cmd.Flags().StringSliceP("container", "c", nil, "container(s) to look up (repeatable)")
	cmd.Flags().StringSlice("target-file", nil, "file(s) held open by a process (repeatable)")
	cmd.Flags().BoolP("short", "s", false, "show only ancestry")
	cmd.Flags().BoolP("tree", "t", false, "show only ancestry as a tree")
	cmd.Flags().Bool("json", false, "show result as JSON")
	cmd.Flags().Bool("warnings", false, "show only warnings")
	cmd.Flags().Bool("no-color", false, "disable colorized output")
	cmd.Flags().Bool("env", false, "show environment variables for the process")
	cmd.Flags().Bool("verbose", false, "show extended process information")
	cmd.Flags().BoolP("exact", "x", false, "use exact name matching")
	cmd.Flags().BoolP("interactive", "i", false, "interactive mode (TUI)")

	return cmd
}

func executeRecord(cmd *cobra.Command, outPath string, interval time.Duration, count int, duration time.Duration) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	errWriter := cmd.ErrOrStderr()
	fmt.Fprintf(errWriter, "Recording system snapshots every %s to %s (press Ctrl+C to stop)...\n", interval, outPath)

	opts := record.RecorderOptions{
		FilePath: outPath,
		Interval: interval,
		Count:    count,
		Duration: duration,
		OnSample: func(snap *model.Snapshot) {
			fmt.Fprintf(errWriter, "Captured sample %d (%d processes)\n", snap.Index+1, len(snap.Processes))
		},
	}

	if err := record.RunRecorder(ctx, opts); err != nil && err != context.Canceled {
		return withExitCode(ExitInternalError, fmt.Errorf("recording error: %w", err))
	}

	fmt.Fprintf(errWriter, "Recording finished.\n")
	return nil
}

func executeReplay(cmd *cobra.Command, filePath string, args []string, step int, timestamp string, all bool) error {
	if strings.TrimSpace(filePath) == "" {
		return withExitCode(ExitInvalidInput, fmt.Errorf("must specify replay file with -f or --file"))
	}

	snapshots, err := record.LoadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(strings.ToLower(err.Error()), "no such file") {
			cmd.PrintErrln(fmt.Sprintf("error: file %q does not exist", filePath))
			return withExitCode(ExitNotFound, err)
		}
		cmd.PrintErrln(fmt.Sprintf("error loading replay file: %v", err))
		return withExitCode(ExitInvalidInput, err)
	}

	stepper, err := record.NewStepper(snapshots)
	if err != nil {
		return withExitCode(ExitInvalidInput, err)
	}

	flags := appFlags{
		env:     boolFlag(cmd, "env"),
		exact:   boolFlag(cmd, "exact"),
		short:   boolFlag(cmd, "short"),
		tree:    boolFlag(cmd, "tree"),
		json:    boolFlag(cmd, "json"),
		warn:    boolFlag(cmd, "warnings"),
		noColor: boolFlag(cmd, "no-color"),
		verbose: boolFlag(cmd, "verbose"),
	}

	targets := collectReplayTargets(cmd, args)
	interactiveFlag, _ := cmd.Flags().GetBool("interactive")

	// No targets provided: show session summary or launch interactive TUI
	if len(targets) == 0 {
		if interactiveFlag || (isTerminal(os.Stdin) && isTerminal(os.Stdout) && !flags.json && !flags.short && !flags.tree && !flags.warn && !flags.verbose) {
			v := version
			if v == "v0.0.0-dev" {
				v = ""
			}
			return tui.StartReplay(v, snapshots, nil, flags.exact)
		}

		outw := cmd.OutOrStdout()
		if flags.json {
			summaryJSON, err := record.SummaryJSON(record.Summarize(snapshots), snapshots)
			if err != nil {
				return withExitCode(ExitInternalError, err)
			}
			fmt.Fprintln(outw, summaryJSON)
			return nil
		}

		colorEnabled := useColor(flags, outw)
		record.RenderSummary(outw, record.Summarize(snapshots), snapshots, colorEnabled)
		return nil
	}

	outw := cmd.OutOrStdout()
	outp := output.NewPrinter(outw)
	colorEnabled := useColor(flags, outw)

	if all {
		highestExit := ExitOK
		for i := 0; i < stepper.Count(); i++ {
			snap, _ := stepper.StepTo(i)
			if !flags.json {
				if i > 0 {
					outp.Println()
				}
				if colorEnabled {
					outp.Printf("%s=== Sample [%d/%d] @ %s ===%s\n",
						output.ColorCyan, i+1, stepper.Count(),
						snap.Timestamp.Format("2006-01-02 15:04:05"), output.ColorReset)
				} else {
					outp.Printf("=== Sample [%d/%d] @ %s ===\n",
						i+1, stepper.Count(), snap.Timestamp.Format("2006-01-02 15:04:05"))
				}
			}
			code := replaySingleSnapshot(cmd, outw, outp, snap, stepper, targets, flags)
			if exitSeverity[code] > exitSeverity[highestExit] {
				highestExit = code
			}
		}
		if highestExit > ExitOK {
			cmd.SilenceErrors = true
			return withExitCode(highestExit, fmt.Errorf("completed with exit code %d", highestExit))
		}
		return nil
	}

	// Select snapshot by timestamp, step, or default to latest
	var activeSnap *model.Snapshot
	if timestamp != "" {
		s, _, sErr := stepper.SeekTimestampString(timestamp)
		if sErr != nil {
			cmd.PrintErrln(fmt.Sprintf("invalid timestamp: %v", sErr))
			return withExitCode(ExitInvalidInput, sErr)
		}
		activeSnap = s
	} else if step >= 0 {
		s, sErr := stepper.StepTo(step)
		if sErr != nil {
			cmd.PrintErrln(fmt.Sprintf("invalid step index: %v", sErr))
			return withExitCode(ExitInvalidInput, sErr)
		}
		activeSnap = s
	} else {
		activeSnap, _ = stepper.StepTo(stepper.Count() - 1)
	}

	code := replaySingleSnapshot(cmd, outw, outp, activeSnap, stepper, targets, flags)
	if code > ExitOK {
		cmd.SilenceErrors = true
		return withExitCode(code, fmt.Errorf("completed with exit code %d", code))
	}
	return nil
}

func collectReplayTargets(cmd *cobra.Command, args []string) []model.Target {
	var targets []model.Target
	for _, name := range args {
		targets = append(targets, model.Target{Type: model.TargetName, Value: name})
	}
	pidFlags, _ := cmd.Flags().GetStringSlice("pid")
	for _, p := range pidFlags {
		if p = strings.TrimSpace(p); p != "" && p != "[]" {
			targets = append(targets, model.Target{Type: model.TargetPID, Value: p})
		}
	}
	portFlags, _ := cmd.Flags().GetStringSlice("port")
	for _, o := range portFlags {
		if o = strings.TrimSpace(o); o != "" && o != "[]" {
			targets = append(targets, model.Target{Type: model.TargetPort, Value: o})
		}
	}
	containerFlags, _ := cmd.Flags().GetStringSlice("container")
	for _, c := range containerFlags {
		if c = strings.TrimSpace(c); c != "" && c != "[]" {
			targets = append(targets, model.Target{Type: model.TargetContainer, Value: c})
		}
	}
	fileFlags, _ := cmd.Flags().GetStringSlice("target-file")
	for _, f := range fileFlags {
		if f = strings.TrimSpace(f); f != "" && f != "[]" {
			targets = append(targets, model.Target{Type: model.TargetFile, Value: f})
		}
	}
	return targets
}

func replaySingleSnapshot(cmd *cobra.Command, outw io.Writer, outp output.Printer, snap *model.Snapshot, stepper *record.Stepper, targets []model.Target, flags appFlags) int {
	multiMode := len(targets) > 1
	colorEnabled := useColor(flags, outw)
	var jsonResults []string
	highestExit := ExitOK

	for i, t := range targets {
		if multiMode && !flags.json {
			printDivider(outp, t, colorEnabled, i > 0)
		}

		code := replayTarget(cmd, outw, outp, snap, stepper, t, flags, multiMode, &jsonResults)
		if exitSeverity[code] > exitSeverity[highestExit] {
			highestExit = code
		}
	}

	if flags.json && multiMode {
		indented := make([]string, len(jsonResults))
		for i, r := range jsonResults {
			lines := strings.Split(r, "\n")
			for j := range lines {
				if j > 0 {
					lines[j] = "  " + lines[j]
				}
			}
			indented[i] = "  " + strings.Join(lines, "\n")
		}
		fmt.Fprintf(outw, "[\n%s\n]\n", strings.Join(indented, ",\n"))
	}

	return highestExit
}

func replayTarget(cmd *cobra.Command, outw io.Writer, outp output.Printer, snap *model.Snapshot, stepper *record.Stepper, t model.Target, flags appFlags, multiMode bool, jsonResults *[]string) int {
	colorEnabled := useColor(flags, outw)

	pids, err := stepper.ResolveTarget(t, flags.exact)
	if err != nil {
		switch {
		case flags.json:
			jsonError(cmd, t, err.Error(), multiMode, jsonResults)
		case multiMode:
			outp.Printf("Error: %v\n", err)
		default:
			cmd.PrintErrln(errorWithHint(err))
		}
		return classifyError(err)
	}

	if len(pids) > 1 {
		if flags.json {
			emitJSON(outw, jsonMatchEntry(t, fmt.Sprintf("multiple processes matched (%d results)", len(pids)), processMatchesFromSnapshot(snap, pids)), multiMode, jsonResults)
		} else {
			printMultiMatchFromSnapshot(outp, snap, pids, colorEnabled)
		}
		return ExitInvalidInput
	}

	pid := pids[0]
	res, err := record.AnalyzeSnapshot(snap, pid, pipeline.AnalyzeConfig{
		PID:     pid,
		Verbose: flags.verbose,
		Tree:    flags.tree,
		Target:  t,
	})
	if err != nil {
		switch {
		case flags.json:
			jsonError(cmd, t, err.Error(), multiMode, jsonResults)
		case multiMode:
			outp.Printf("Error: %v\n", err)
		default:
			cmd.PrintErrln(errorWithHint(err))
		}
		return classifyError(err)
	}

	if flags.env {
		resEnv := model.Result{
			Target:   t,
			Process:  res.Process,
			Ancestry: []model.Process{res.Process},
		}
		if flags.json {
			jsonStr, err := output.ToEnvJSON(resEnv)
			return emitJSONResult(outw, t, jsonStr, err, multiMode, jsonResults)
		}
		output.RenderEnvOnly(outw, resEnv, colorEnabled)
		return ExitOK
	}

	return renderResult(outw, res, flags, multiMode, jsonResults)
}

func processMatchesFromSnapshot(snap *model.Snapshot, pids []int) []processMatch {
	procMap := make(map[int]model.Process, len(snap.Processes))
	for _, p := range snap.Processes {
		procMap[p.PID] = p
	}

	matches := make([]processMatch, 0, len(pids))
	for _, pid := range pids {
		m := processMatch{PID: pid, Command: "unknown"}
		if p, ok := procMap[pid]; ok {
			m.Command = p.Command
			m.Cmdline = p.Cmdline
		}
		matches = append(matches, m)
	}
	return matches
}

func printMultiMatchFromSnapshot(outp output.Printer, snap *model.Snapshot, pids []int, colorEnabled bool) {
	outp.Printf("Multiple matching processes found:\n\n")
	for i, m := range processMatchesFromSnapshot(snap, pids) {
		if colorEnabled {
			outp.Printf("[%d] %s%s%s (%spid %d%s)\n    %s\n",
				i+1, output.ColorGreen, m.Command, output.ColorReset,
				output.ColorDim, m.PID, output.ColorReset,
				m.Cmdline)
		} else {
			outp.Printf("[%d] %s (pid %d)\n    %s\n", i+1, m.Command, m.PID, m.Cmdline)
		}
	}
	outp.Printf("\nRe-run with exact PID or name to disambiguate.\n")
}
