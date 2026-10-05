//go:build linux || darwin || freebsd || windows

package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/internal/pipeline"
	procpkg "github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/internal/source"
	"github.com/pranshuparmar/witr/internal/target"
	"github.com/pranshuparmar/witr/internal/tui"
	"github.com/pranshuparmar/witr/pkg/model"
	"github.com/spf13/cobra"
)

var (
	version   = "v0.0.0-dev"
	commit    = "unknown"
	buildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "witr [process name...]",
	Short: "Why is this running?",
	Long:  "witr explains why a process or port is running by tracing its ancestry.",
	Args:  cobra.ArbitraryArgs,
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

  # Find the process holding a file open
  witr --file /var/lib/dpkg/lock

  # Inspect a container by name
  witr --container redis

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

  # Multiple inputs
  witr nginx node
  witr --port 8080 --port 3000
  witr --pid 1234 --pid 5678

  # Mixed inputs
  witr nginx --pid 1234 --port 8080
`
}

// Exit codes
const (
	ExitOK           = 0
	ExitWarnings     = 1
	ExitNotFound     = 2
	ExitPermission   = 3
	ExitInvalidInput = 4
	// ExitInternalError is distinct from ExitWarnings so scripts can tell an
	// unexpected witr failure apart from "process found, has warnings".
	ExitInternalError = 5
	// ExitCauseUnknown: the process was found, but what started it can't be
	// traced. Its warnings say why.
	ExitCauseUnknown = 6
)

// exitSeverity ranks exit codes across several targets, where the most
// severe wins. Cause unknown is a finding, like warnings, so it ranks below
// every failure despite its number.
var exitSeverity = map[int]int{
	ExitOK:            0,
	ExitWarnings:      1,
	ExitCauseUnknown:  2,
	ExitNotFound:      3,
	ExitPermission:    4,
	ExitInvalidInput:  5,
	ExitInternalError: 6,
}

// exitCodeError wraps an error with a specific exit code.
type exitCodeError struct {
	code int
	err  error
}

func (e *exitCodeError) Error() string { return e.err.Error() }
func (e *exitCodeError) Unwrap() error { return e.err }

func withExitCode(code int, err error) error {
	return &exitCodeError{code: code, err: err}
}

func Execute() {
	err := rootCmd.Execute()
	if err == nil {
		return
	}

	var ece *exitCodeError
	if errors.As(err, &ece) {
		os.Exit(ece.code)
	}
	// Errors without a code come from cobra's own parsing: an unknown flag, a
	// flag missing its value, a bad argument.
	os.Exit(ExitInvalidInput)
}

func init() {
	rootCmd.InitDefaultCompletionCmd()
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("witr {{.Version}} (commit %s, built %s)\n", commit, buildDate))
	rootCmd.SetErr(output.NewSafeTerminalWriter(os.Stderr))

	rootCmd.Flags().StringSliceP("pid", "p", nil, "pid(s) to look up (repeatable)")
	rootCmd.Flags().StringSliceP("port", "o", nil, "port(s) to look up (repeatable)")
	rootCmd.Flags().StringSliceP("file", "f", nil, "file(s) held open by a process (repeatable)")
	rootCmd.Flags().StringSliceP("container", "c", nil, "container(s) to look up (repeatable)")
	rootCmd.Flags().BoolP("short", "s", false, "show only ancestry")
	rootCmd.Flags().BoolP("tree", "t", false, "show only ancestry as a tree")
	rootCmd.Flags().Bool("json", false, "show result as JSON")
	rootCmd.Flags().Bool("warnings", false, "show only warnings")
	rootCmd.Flags().Bool("no-color", false, "disable colorized output")
	rootCmd.Flags().Bool("env", false, "show environment variables for the process")
	rootCmd.Flags().Bool("verbose", false, "show extended process information")
	rootCmd.Flags().BoolP("exact", "x", false, "use exact name matching (no substring search)")
	rootCmd.Flags().BoolP("interactive", "i", false, "interactive mode (TUI)")

	rootCmd.AddCommand(newRecordCmd())
	rootCmd.AddCommand(newReplayCmd())

	rootCmd.Flags().StringVar(&recordOutputFlag, "record", "", "record system sample stream to file")
	rootCmd.Flags().StringVar(&replayFileFlag, "replay", "", "replay recorded system sample stream from file")
	rootCmd.Flags().DurationVar(&recordIntervalFlag, "interval", 2*time.Second, "sample capture interval for --record")
	rootCmd.Flags().IntVar(&recordCountFlag, "count", 0, "sample count for --record")
	rootCmd.Flags().IntVar(&replayStepFlag, "step", -1, "0-based step index of snapshot to replay")
	rootCmd.Flags().StringVar(&replayTimestampFlag, "timestamp", "", "timestamp to seek in replay")
	rootCmd.Flags().BoolVar(&replayAllFlag, "all", false, "evaluate target across all snapshots")
}

// appFlags holds all parsed CLI flags for convenience.
type appFlags struct {
	short   bool
	tree    bool
	json    bool
	warn    bool
	noColor bool
	verbose bool
	exact   bool
	env     bool
}

func runApp(cmd *cobra.Command, args []string) error {
	if recordPath, _ := cmd.Flags().GetString("record"); recordPath != "" {
		interval, _ := cmd.Flags().GetDuration("interval")
		count, _ := cmd.Flags().GetInt("count")
		return executeRecord(cmd, recordPath, interval, count, 0)
	}

	if replayPath, _ := cmd.Flags().GetString("replay"); replayPath != "" {
		step, _ := cmd.Flags().GetInt("step")
		ts, _ := cmd.Flags().GetString("timestamp")
		all, _ := cmd.Flags().GetBool("all")
		return executeReplay(cmd, replayPath, args, step, ts, all)
	}

	interactiveFlag, _ := cmd.Flags().GetBool("interactive")
	if interactiveFlag {
		return runInteractive(orderedTargets(cmd, os.Args[1:], args), boolFlag(cmd, "exact"))
	}

	envFlag, _ := cmd.Flags().GetBool("env")
	pidFlags, _ := cmd.Flags().GetStringSlice("pid")
	portFlags, _ := cmd.Flags().GetStringSlice("port")
	fileFlags, _ := cmd.Flags().GetStringSlice("file")
	containerFlags, _ := cmd.Flags().GetStringSlice("container")

	// With no target the TUI opens, unless an output mode was asked for: that
	// needs something to explain, and gets the "must specify" error below.
	outputMode := envFlag || boolFlag(cmd, "short") || boolFlag(cmd, "tree") || boolFlag(cmd, "json") || boolFlag(cmd, "warnings") || boolFlag(cmd, "verbose")
	if !outputMode && len(pidFlags) == 0 && len(portFlags) == 0 && len(fileFlags) == 0 && len(containerFlags) == 0 && len(args) == 0 {
		return runInteractive(nil, false)
	}

	flags := appFlags{
		env:     envFlag,
		exact:   boolFlag(cmd, "exact"),
		short:   boolFlag(cmd, "short"),
		tree:    boolFlag(cmd, "tree"),
		json:    boolFlag(cmd, "json"),
		warn:    boolFlag(cmd, "warnings"),
		noColor: boolFlag(cmd, "no-color"),
		verbose: boolFlag(cmd, "verbose"),
	}

	// Collect all targets preserving command-line order
	targets := orderedTargets(cmd, os.Args[1:], args)

	if len(targets) == 0 {
		return withExitCode(ExitInvalidInput, fmt.Errorf("must specify --pid, --port, --file, --container, or a process name"))
	}

	outw := cmd.OutOrStdout()
	outp := output.NewPrinter(outw)
	multiMode := len(targets) > 1
	colorEnabled := useColor(flags, outw)

	// For JSON multi-output, collect all JSON strings and wrap in array
	var jsonResults []string
	highestExit := ExitOK

	for i, t := range targets {
		if multiMode && !flags.json {
			printDivider(outp, t, colorEnabled, i > 0)
		}

		exitCode := processTarget(cmd, outw, outp, t, flags, multiMode, &jsonResults)
		if exitSeverity[exitCode] > exitSeverity[highestExit] {
			highestExit = exitCode
		}
	}

	// Emit JSON array for multi-target
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

	if highestExit > ExitOK {
		cmd.SilenceErrors = true
		return withExitCode(highestExit, fmt.Errorf("completed with exit code %d", highestExit))
	}
	return nil
}

func boolFlag(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

// flagTakesValue reports whether a raw argv token names a non-boolean flag
// whose value is the following token (e.g. "--config foo"). It lets the
// order-preserving parser take flag-arity from cobra's flag set instead of
// assuming every non-target flag is boolean — so a future string-valued flag
// won't have its value mistaken for a target.
func flagTakesValue(cmd *cobra.Command) func(string) bool {
	return func(arg string) bool {
		if strings.Contains(arg, "=") {
			return false // value is attached: --flag=value
		}
		if name, ok := strings.CutPrefix(arg, "--"); ok {
			if f := cmd.Flags().Lookup(name); f != nil {
				return f.NoOptDefVal == ""
			}
		} else if sh, ok := strings.CutPrefix(arg, "-"); ok && len(sh) == 1 {
			if f := cmd.Flags().ShorthandLookup(sh); f != nil {
				return f.NoOptDefVal == ""
			}
		}
		return false
	}
}

// collectTargetsInOrder walks the raw command-line arguments to build a target
// list that preserves the order the user typed them in. takesValue reports
// whether a non-target flag consumes the following token as its value.
func collectTargetsInOrder(rawArgs []string, positionalArgs []string, takesValue func(string) bool) []model.Target {
	var targets []model.Target
	positionalIdx := 0

	// Map flag names to target types
	flagType := map[string]model.TargetType{
		"-p": model.TargetPID, "--pid": model.TargetPID,
		"-o": model.TargetPort, "--port": model.TargetPort,
		"-f": model.TargetFile, "--file": model.TargetFile,
		"-c": model.TargetContainer, "--container": model.TargetContainer,
	}

	// Track which positional args we've placed so we can insert them in order
	// between flag-based targets
	i := 0
	for i < len(rawArgs) {
		arg := rawArgs[i]

		// "--" ends option parsing (POSIX): everything after it is positional,
		// matching how cobra fills positionalArgs.
		if arg == "--" {
			break
		}

		// Check for --flag=value form
		if strings.HasPrefix(arg, "--") {
			if eqIdx := strings.Index(arg, "="); eqIdx >= 0 {
				flagName := arg[:eqIdx]
				flagVal := arg[eqIdx+1:]
				if tt, ok := flagType[flagName]; ok {
					for _, v := range strings.Split(flagVal, ",") {
						v = strings.TrimSpace(v)
						if v != "" {
							targets = append(targets, model.Target{Type: tt, Value: v})
						}
					}
				}
				i++
				continue
			}
		}

		// Check for -f value or --flag value form
		if tt, ok := flagType[arg]; ok {
			if i+1 < len(rawArgs) {
				i++
				for _, v := range strings.Split(rawArgs[i], ",") {
					v = strings.TrimSpace(v)
					if v != "" {
						targets = append(targets, model.Target{Type: tt, Value: v})
					}
				}
			}
			i++
			continue
		}

		// Non-target flag. If it's a value-taking flag in space form
		// (--flag value, not --flag=value), skip its value too so the value
		// isn't mistaken for a positional target.
		if strings.HasPrefix(arg, "-") {
			if takesValue(arg) && i+1 < len(rawArgs) {
				i++ // consume the flag's value
			}
			i++
			continue
		}

		// Positional argument — use it as a name target
		if positionalIdx < len(positionalArgs) {
			targets = append(targets, model.Target{Type: model.TargetName, Value: positionalArgs[positionalIdx]})
			positionalIdx++
		}
		i++
	}

	// Append any remaining positional args that weren't matched
	for positionalIdx < len(positionalArgs) {
		targets = append(targets, model.Target{Type: model.TargetName, Value: positionalArgs[positionalIdx]})
		positionalIdx++
	}

	return targets
}

// orderedTargets returns the targets in the order they were typed. The walk
// over the raw arguments only exists to recover that order, so whenever it
// disagrees with what cobra parsed, cobra's values win and no target is lost.
func orderedTargets(cmd *cobra.Command, rawArgs, args []string) []model.Target {
	ordered := collectTargetsInOrder(expandShortFlags(rawArgs, shortFlagArity(cmd)), args, flagTakesValue(cmd))
	if parsed := parsedTargets(cmd, args); !sameTargets(ordered, parsed) {
		return parsed
	}
	return ordered
}

// shortFlagArity reports whether a one-letter flag exists and takes a value.
func shortFlagArity(cmd *cobra.Command) func(string) (known, takesValue bool) {
	return func(short string) (bool, bool) {
		f := cmd.Flags().ShorthandLookup(short)
		if f == nil {
			return false, false
		}
		return true, f.NoOptDefVal == ""
	}
}

// expandShortFlags rewrites combined and attached short flags into separate
// tokens, the way pflag reads them: -sp 1 becomes -s -p 1, and -p1 or -p=1
// becomes -p 1. Everything after "--" is left alone.
func expandShortFlags(args []string, arity func(string) (known, takesValue bool)) []string {
	out := make([]string, 0, len(args))
	for i, arg := range args {
		if arg == "--" {
			return append(out, args[i:]...)
		}
		if len(arg) < 3 || arg[0] != '-' || arg[1] == '-' {
			out = append(out, arg)
			continue
		}
		rest := arg[1:]
		for rest != "" {
			known, takesValue := arity(rest[:1])
			if !known {
				out = append(out, "-"+rest) // left for cobra to reject
				break
			}
			out = append(out, "-"+rest[:1])
			rest = rest[1:]
			if takesValue {
				if v := strings.TrimPrefix(rest, "="); v != "" {
					out = append(out, v)
				}
				break
			}
		}
	}
	return out
}

// parsedTargets lists the targets cobra parsed, grouped by type.
func parsedTargets(cmd *cobra.Command, args []string) []model.Target {
	var targets []model.Target
	for _, name := range args {
		targets = append(targets, model.Target{Type: model.TargetName, Value: name})
	}
	for _, f := range []struct {
		name string
		typ  model.TargetType
	}{
		{"pid", model.TargetPID},
		{"port", model.TargetPort},
		{"file", model.TargetFile},
		{"container", model.TargetContainer},
	} {
		values, _ := cmd.Flags().GetStringSlice(f.name)
		for _, v := range values {
			if v = strings.TrimSpace(v); v != "" {
				targets = append(targets, model.Target{Type: f.typ, Value: v})
			}
		}
	}
	return targets
}

// sameTargets reports whether a and b hold the same targets in any order.
func sameTargets(a, b []model.Target) bool {
	if len(a) != len(b) {
		return false
	}
	count := map[model.Target]int{}
	for _, t := range a {
		count[t]++
	}
	for _, t := range b {
		if count[t]--; count[t] < 0 {
			return false
		}
	}
	return true
}

// targetLabel returns a human-readable label for the divider.
func targetLabel(t model.Target) string {
	switch t.Type {
	case model.TargetPID:
		return fmt.Sprintf("pid: %s", t.Value)
	case model.TargetPort:
		return fmt.Sprintf("port: %s", t.Value)
	case model.TargetFile:
		return fmt.Sprintf("file: %s", t.Value)
	case model.TargetContainer:
		return fmt.Sprintf("container: %s", t.Value)
	default:
		return fmt.Sprintf("name: %s", t.Value)
	}
}

func printDivider(outp output.Printer, t model.Target, colorEnabled bool, needsNewline bool) {
	label := targetLabel(t)
	if needsNewline {
		outp.Println()
	}
	if colorEnabled {
		outp.Printf("%s----- [%s] -----%s\n", output.ColorCyan, label, output.ColorReset)
	} else {
		outp.Printf("----- [%s] -----\n", label)
	}
}

// jsonErrorEntry returns a JSON string representing a failed target lookup.
func jsonErrorEntry(t model.Target, errMsg string) string {
	return jsonMatchEntry(t, errMsg, nil)
}

// jsonMatchEntry is a failed lookup of an ambiguous target: matches lists the
// candidates, so a script can re-run against the one it wants.
func jsonMatchEntry(t model.Target, errMsg string, matches any) string {
	type errorEntry struct {
		Target  model.Target
		Error   string
		Matches any `json:",omitempty"`
	}
	data, _ := output.MarshalJSON(errorEntry{Error: errMsg, Target: t, Matches: matches})
	return data
}

// emitJSON adds an entry to the multi-target JSON array, or prints it as the
// whole output for a single target.
func emitJSON(outw io.Writer, entry string, multiMode bool, jsonResults *[]string) {
	if multiMode {
		*jsonResults = append(*jsonResults, entry)
		return
	}
	fmt.Fprintln(outw, entry)
}

// jsonError reports a failed target under --json as a {Target, Error} entry,
// so stdout stays JSON for a single target and multi-target arrays alike.
func jsonError(cmd *cobra.Command, t model.Target, msg string, multiMode bool, jsonResults *[]string) {
	emitJSON(cmd.OutOrStdout(), jsonErrorEntry(t, msg), multiMode, jsonResults)
}

// emitJSONResult emits a rendered JSON entry, or an error entry for t when
// rendering failed, so --json output never mixes in plain text.
func emitJSONResult(outw io.Writer, t model.Target, jsonStr string, err error, multiMode bool, jsonResults *[]string) int {
	if err != nil {
		emitJSON(outw, jsonErrorEntry(t, "failed to generate json output: "+err.Error()), multiMode, jsonResults)
		return ExitInternalError
	}
	emitJSON(outw, jsonStr, multiMode, jsonResults)
	return ExitOK
}

// processTarget handles resolving and rendering a single target.
// Returns the exit code for this target.
func processTarget(cmd *cobra.Command, outw io.Writer, outp output.Printer, t model.Target, flags appFlags, multiMode bool, jsonResults *[]string) int {
	colorEnabled := useColor(flags, outw)

	if flags.env {
		return processEnvTarget(cmd, outw, outp, t, flags, multiMode, jsonResults)
	}

	if t.Type == model.TargetContainer {
		return processContainerTarget(cmd, outw, outp, t, flags, multiMode, jsonResults)
	}

	pids, err := target.Resolve(t, flags.exact)
	if err == nil && len(pids) == 0 {
		err = fmt.Errorf("no matching process found")
	}
	if err != nil {
		return handleResolveError(cmd, outw, outp, t, err, flags, multiMode, jsonResults)
	}

	// A port published for a container, through docker-proxy (often one each
	// for IPv4 and IPv6) or Docker Desktop's forwarders, is explained by the
	// container behind it: its main process when that is visible here,
	// otherwise the runtime's view of it.
	if t.Type == model.TargetPort {
		if port, convErr := strconv.Atoi(t.Value); convErr == nil {
			if match, names := procpkg.PublishedContainer(port, pids); match != nil {
				procpkg.EnrichContainer(match)
				if code, ok := analyzeContainer(cmd, outw, outp, t, match, flags, multiMode, jsonResults); ok {
					return code
				}
				return renderContainerMatch(outw, outp, t, "port "+t.Value, match, flags, multiMode, jsonResults, output.PublishedNote(names, pids))
			}
		}
	}

	if len(pids) > 1 {
		if flags.json {
			emitJSON(outw, jsonMatchEntry(t, fmt.Sprintf("multiple processes matched (%d results)", len(pids)), processMatches(pids)), multiMode, jsonResults)
		} else {
			hint := "witr --pid <pid>"
			if flags.env {
				hint = "witr --pid <pid> --env"
			}
			printMultiMatch(outp, pids, colorEnabled, hint)
		}
		return ExitInvalidInput
	}

	pid := pids[0]

	var systemdService string
	if t.Type == model.TargetPort && pid == 1 && source.IsSystemdRunning() {
		if portNum, err := strconv.Atoi(t.Value); err == nil {
			if svc, err := procpkg.ResolveSystemdService(portNum); err == nil && svc != "" {
				systemdService = svc
			}
		}
	}

	res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
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

	if systemdService != "" {
		res.ResolvedTarget = strings.TrimSuffix(systemdService, ".service")
	}

	addSocketInfo(&res, t)
	return renderResult(outw, res, flags, multiMode, jsonResults)
}

// processEnvTarget handles the --env flag for a single target.
func processEnvTarget(cmd *cobra.Command, outw io.Writer, outp output.Printer, t model.Target, flags appFlags, multiMode bool, jsonResults *[]string) int {
	colorEnabled := useColor(flags, outw)

	fail := func(msg string, code int) int {
		switch {
		case flags.json:
			jsonError(cmd, t, msg, multiMode, jsonResults)
		case multiMode:
			outp.Printf("Error: %s\n", msg)
		default:
			outp.Printf("error: %s\n", msg)
		}
		return code
	}

	var pid int
	if t.Type == model.TargetContainer {
		matches := procpkg.ResolveContainer(t.Value, flags.exact)
		switch {
		case len(matches) == 0:
			return fail(fmt.Sprintf("no container found matching %q", t.Value), ExitNotFound)
		case len(matches) > 1:
			if flags.json {
				emitJSON(outw, jsonMatchEntry(t, fmt.Sprintf("multiple containers matched (%d results)", len(matches)), matches), multiMode, jsonResults)
			} else {
				printContainerMultiMatch(outp, matches, colorEnabled)
			}
			return ExitInvalidInput
		}
		match := matches[0]
		pid = procpkg.ResolveContainerHostPID(match.Runtime, match.ID)
		if pid <= 0 || !procpkg.PIDBelongsToContainer(pid, match.ID) {
			return fail(fmt.Sprintf("container %s has no process visible from here", match.Name), ExitNotFound)
		}
	} else {
		pids, err := target.Resolve(t, flags.exact)
		switch {
		case err != nil:
			return fail(err.Error(), classifyError(err))
		case len(pids) == 0:
			return fail("no matching process found", ExitNotFound)
		case len(pids) > 1:
			if flags.json {
				emitJSON(outw, jsonMatchEntry(t, fmt.Sprintf("multiple processes matched (%d results)", len(pids)), processMatches(pids)), multiMode, jsonResults)
			} else {
				printMultiMatch(outp, pids, colorEnabled, "witr --pid <pid> --env")
			}
			return ExitInvalidInput
		}
		pid = pids[0]
	}

	procInfo, err := procpkg.ReadProcess(pid)
	if err != nil {
		return fail(err.Error(), classifyError(err))
	}

	resEnv := model.Result{
		Target:   t,
		Process:  procInfo,
		Ancestry: []model.Process{procInfo},
	}

	if flags.json {
		jsonStr, err := output.ToEnvJSON(resEnv)
		return emitJSONResult(outw, t, jsonStr, err, multiMode, jsonResults)
	}
	output.RenderEnvOnly(outw, resEnv, colorEnabled)
	return ExitOK
}

// handleResolveError handles target resolution errors, including Docker fallback.
func handleResolveError(cmd *cobra.Command, outw io.Writer, outp output.Printer, t model.Target, err error, flags appFlags, multiMode bool, jsonResults *[]string) int {
	errStr := err.Error()

	// Platform-unsupported target (e.g. -f on Windows). Don't tack on the
	// generic "try a different name/port/PID" suffix — the operation isn't a
	// failed lookup, it's unavailable on this OS.
	if errors.Is(err, target.ErrUnsupported) || strings.Contains(errStr, "not supported on") {
		switch {
		case flags.json:
			jsonError(cmd, t, errStr, multiMode, jsonResults)
		case multiMode:
			outp.Printf("Error: %v\n", err)
		default:
			cmd.PrintErrln(errStr)
		}
		return ExitInvalidInput
	}

	ownerUnknown := errors.Is(err, target.ErrSocketOwnerUnknown) || strings.Contains(errStr, "socket found but owning process not detected")
	// A port no visible process holds may still be published for a container:
	// by a runtime in another namespace (Docker Desktop), or through firewall
	// rules (rootful Podman and nerdctl, Docker without its userland proxy).
	if t.Type == model.TargetPort && (ownerUnknown || classifyError(err) == ExitNotFound) {
		if code, ok := portContainer(cmd, outw, outp, t, flags, multiMode, jsonResults); ok {
			return code
		}
	}

	// Nothing is hidden from root, nor on Windows, which reports every
	// socket's owner: a socket no process holds is held from outside this
	// system, and sudo can't help.
	if ownerUnknown && !ownersHidden() {
		short := errStr + " (no process on this system holds it)"
		switch {
		case flags.json:
			jsonError(cmd, t, short, multiMode, jsonResults)
		case multiMode:
			outp.Printf("Error: %s\n", short)
		default:
			cmd.PrintErrln(errStr + "\n\n" + heldOutsideHint())
		}
		return ExitNotFound
	}

	if ownerUnknown {
		short := errStr + " (try sudo)"
		switch {
		case flags.json:
			jsonError(cmd, t, short, multiMode, jsonResults)
		case multiMode:
			outp.Printf("Error: %s\n", short)
		default:
			errorMsg := fmt.Sprintf("%s\n\nA socket was found for the port, but the owning process could not be detected.\nThis may be due to insufficient permissions. Try running with sudo:\n  sudo %s", errStr, strings.Join(os.Args, " "))
			cmd.PrintErrln(errorMsg)
		}
		return ExitPermission
	}

	switch {
	case flags.json:
		jsonError(cmd, t, errStr, multiMode, jsonResults)
	case multiMode:
		outp.Printf("Error: %v\n", err)
	default:
		errorMsg := errorWithHint(err)
		if t.Type == model.TargetFile && runtime.GOOS != "windows" && os.Geteuid() != 0 {
			errorMsg += "\n\nIf the file is held by another user's process, retry with sudo:\n  sudo " + strings.Join(os.Args, " ")
		}
		cmd.PrintErrln(errorMsg)
	}
	return classifyError(err)
}

// renderResult renders a single result in the selected output mode and
// returns the target's exit code.
func renderResult(outw io.Writer, res model.Result, flags appFlags, multiMode bool, jsonResults *[]string) int {
	colorEnabled := useColor(flags, outw)

	switch {
	case flags.json:
		var jsonStr string
		var err error
		switch {
		case flags.short:
			jsonStr, err = output.ToShortJSON(res)
		case flags.tree:
			jsonStr, err = output.ToTreeJSON(res)
		case flags.warn:
			jsonStr, err = output.ToWarningsJSON(res)
		default:
			jsonStr, err = output.ToJSON(res)
		}
		if code := emitJSONResult(outw, res.Target, jsonStr, err, multiMode, jsonResults); code != ExitOK {
			return code
		}
	case flags.warn:
		output.RenderWarnings(outw, res, colorEnabled)
	case flags.tree:
		output.PrintTree(outw, res.Ancestry, res.Children, colorEnabled)
	case flags.short:
		output.RenderShort(outw, res, colorEnabled)
	default:
		output.RenderStandard(outw, res, colorEnabled, flags.verbose)
	}

	switch {
	case source.Untraced(res.Source.Type):
		return ExitCauseUnknown
	case len(res.Warnings) > 0:
		return ExitWarnings
	}
	return ExitOK
}

func Root() *cobra.Command { return rootCmd }

func runInteractive(targets []model.Target, exact bool) error {
	// Without a terminal (a script, a pipe, CI) the TUI would wait forever.
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		return withExitCode(ExitInvalidInput, errors.New("interactive mode needs a terminal; give a target to explain: a process name, --pid, --port, --file or --container"))
	}
	v := version
	if v == "v0.0.0-dev" {
		v = ""
	}
	if err := tui.Start(v, targets, exact); err != nil {
		return withExitCode(ExitInternalError, err)
	}
	return nil
}

// processMatch is one candidate of an ambiguous process target.
type processMatch struct {
	PID     int
	Command string
	Cmdline string
}

func processMatches(pids []int) []processMatch {
	matches := make([]processMatch, 0, len(pids))
	for _, pid := range pids {
		m := processMatch{PID: pid, Command: "unknown"}
		if proc, err := procpkg.ReadProcess(pid); err == nil {
			m.Command, m.Cmdline = proc.Command, proc.Cmdline
		} else {
			m.Cmdline = procpkg.GetCmdline(pid)
		}
		matches = append(matches, m)
	}
	return matches
}

func printMultiMatch(outp output.Printer, pids []int, colorEnabled bool, hint string) {
	outp.Printf("Multiple matching processes found:\n\n")
	for i, m := range processMatches(pids) {
		if colorEnabled {
			outp.Printf("[%d] %s%s%s (%spid %d%s)\n    %s\n",
				i+1, output.ColorGreen, m.Command, output.ColorReset,
				output.ColorDim, m.PID, output.ColorReset,
				m.Cmdline)
		} else {
			outp.Printf("[%d] %s (pid %d)\n    %s\n", i+1, m.Command, m.PID, m.Cmdline)
		}
	}
	outp.Printf("\nRe-run with:\n")
	outp.Printf("  %s\n", hint)
}

func printContainerMultiMatch(outp output.Printer, matches []*model.ContainerMatch, colorEnabled bool) {
	outp.Printf("Multiple matching containers found:\n\n")
	for i, m := range matches {
		name := output.SanitizeTerminal(m.Name)
		image := output.SanitizeTerminal(m.Image)
		status := output.SanitizeTerminal(m.Status)
		ports := output.SanitizeTerminal(m.Ports)
		runtime := output.SanitizeTerminal(m.Runtime)
		if colorEnabled {
			outp.Printf("[%d] %s%s%s (%s%s%s)\n",
				i+1, output.ColorGreen, name, output.ColorReset,
				output.ColorDim, runtime, output.ColorReset)
		} else {
			outp.Printf("[%d] %s (%s)\n", i+1, name, runtime)
		}
		detail := "image: " + image
		if status != "" {
			detail += ", status: " + status
		}
		if ports != "" {
			detail += ", ports: " + ports
		}
		outp.Printf("    %s\n", detail)
	}
	outp.Printf("\nRe-run with the exact container name to disambiguate:\n")
	outp.Println("  witr -c <container-name> --exact")
}

// errorWithHint adds the next step to an error: lookups that found nothing
// get the not-found hint, other failures (such as invalid input) only point
// at --help.
func errorWithHint(err error) string {
	if classifyError(err) == ExitNotFound {
		return err.Error() + "\n\nNo matching process or service found. Please check your query or try a different name/port/PID.\nFor usage and options, run: witr --help"
	}
	return err.Error() + "\n\nFor usage and options, run: witr --help"
}

// classifyError maps common error strings to exit codes.
func classifyError(err error) int {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "operation not permitted") ||
		strings.Contains(msg, "insufficient permissions"):
		return ExitPermission
	case strings.Contains(msg, "no matching") ||
		strings.Contains(msg, "no running process") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "no container found") ||
		strings.Contains(msg, "no process"):
		return ExitNotFound
	case strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "must specify"):
		return ExitInvalidInput
	default:
		return ExitInternalError
	}
}

// processContainerTarget handles `-c/--container` lookups. Resolves against
// every available container runtime, dispatches to the normal pipeline if
// the container's main process is host-visible, otherwise renders the
// runtime-side metadata via the container fallback view.
func processContainerTarget(cmd *cobra.Command, outw io.Writer, outp output.Printer, t model.Target, flags appFlags, multiMode bool, jsonResults *[]string) int {
	colorEnabled := useColor(flags, outw)

	matches := procpkg.ResolveContainer(t.Value, flags.exact)
	if len(matches) == 0 {
		err := fmt.Errorf("no container found matching %q", t.Value)
		return handleResolveError(cmd, outw, outp, t, err, flags, multiMode, jsonResults)
	}

	if len(matches) > 1 {
		if flags.json {
			emitJSON(outw, jsonMatchEntry(t, fmt.Sprintf("multiple containers matched (%d results)", len(matches)), matches), multiMode, jsonResults)
		} else {
			printContainerMultiMatch(outp, matches, colorEnabled)
		}
		return ExitInvalidInput
	}

	match := matches[0]
	procpkg.EnrichContainer(match)
	if code, ok := analyzeContainer(cmd, outw, outp, t, match, flags, multiMode, jsonResults); ok {
		return code
	}
	return renderContainerMatch(outw, outp, t, "container "+match.Name, match, flags, multiMode, jsonResults, "")
}

// analyzeContainer runs the full analysis on match's main process when it is
// visible on this host. ok is false when it isn't, so the caller can fall back
// to the runtime's own view of the container.
func analyzeContainer(cmd *cobra.Command, outw io.Writer, outp output.Printer, t model.Target, match *model.ContainerMatch, flags appFlags, multiMode bool, jsonResults *[]string) (code int, ok bool) {
	pid := procpkg.ResolveContainerHostPID(match.Runtime, match.ID)
	if pid <= 0 || !procpkg.PIDBelongsToContainer(pid, match.ID) {
		return 0, false
	}
	res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
		PID:       pid,
		Verbose:   flags.verbose,
		Tree:      flags.tree,
		Target:    t,
		Container: match,
	})
	if err != nil {
		if flags.json {
			jsonError(cmd, t, err.Error(), multiMode, jsonResults)
		} else {
			outp.Printf("Error: %v\n", err)
		}
		return classifyError(err), true
	}
	res.Process.Container = output.FormatContainerLine(match)
	if len(res.Ancestry) > 0 {
		res.Ancestry[len(res.Ancestry)-1].Container = res.Process.Container
	}
	if res.Container == nil {
		res.Container = match
	}
	addSocketInfo(&res, t)
	return renderResult(outw, res, flags, multiMode, jsonResults), true
}

// ownersHidden reports whether other users' processes are hidden from witr:
// Windows reports every socket's owner, elsewhere only root sees them all. A
// variable for tests.
var ownersHidden = func() bool {
	return runtime.GOOS != "windows" && os.Geteuid() != 0
}

// onWSL reports whether witr runs in a WSL distro. A variable for tests.
var onWSL = func() bool {
	release, err := os.ReadFile("/proc/sys/kernel/osrelease")
	return err == nil && strings.Contains(strings.ToLower(string(release)), "microsoft")
}

// heldOutsideHint explains a socket that no process on this system holds.
func heldOutsideHint() string {
	if onWSL() {
		return "No process in this WSL distro holds the socket. WSL distros share one network,\n" +
			"so it is most likely held by a process in another distro (Docker Desktop runs\n" +
			"in its own), or by the kernel."
	}
	return "No process on this system holds the socket, so it is held from outside it: by a\n" +
		"process in another PID namespace sharing this network (such as a container\n" +
		"runtime's VM), or by the kernel."
}

// containerByPort finds the container publishing a host port; a variable so
// tests needn't run the container runtimes.
var containerByPort = procpkg.ResolveContainerByPort

// portContainer explains a port target by the container that publishes it,
// if any: its main process when that is visible here, otherwise the
// runtime's view of the container.
func portContainer(cmd *cobra.Command, outw io.Writer, outp output.Printer, t model.Target, flags appFlags, multiMode bool, jsonResults *[]string) (int, bool) {
	port, err := strconv.Atoi(t.Value)
	if err != nil {
		return 0, false
	}
	match := containerByPort(port, "")
	if match == nil {
		return 0, false
	}
	procpkg.EnrichContainer(match)
	if code, ok := analyzeContainer(cmd, outw, outp, t, match, flags, multiMode, jsonResults); ok {
		return code, true
	}
	return renderContainerMatch(outw, outp, t, "port "+t.Value, match, flags, multiMode, jsonResults, ""), true
}

// addSocketInfo explains the socket state of a port target.
func addSocketInfo(res *model.Result, t model.Target) {
	if t.Type != model.TargetPort {
		return
	}
	port := 0
	fmt.Sscanf(t.Value, "%d", &port)
	if port > 0 {
		res.SocketInfo = procpkg.GetSocketStateForPort(port)
		source.EnrichSocketInfo(res.SocketInfo)
	}
}

// renderContainerMatch renders a container in the selected output mode.
// proxyPIDs lists the docker-proxy processes publishing it, when the target
// was a port they listen on.
func renderContainerMatch(outw io.Writer, outp output.Printer, t model.Target, label string, match *model.ContainerMatch, flags appFlags, multiMode bool, jsonResults *[]string, note string) int {
	colorEnabled := useColor(flags, outw)
	switch {
	case flags.json:
		jsonStr, err := output.ContainerFallbackToJSON(label, match, note)
		return emitJSONResult(outw, t, jsonStr, err, multiMode, jsonResults)
	case flags.short:
		output.RenderContainerFallbackShort(outw, label, match, colorEnabled)
	case flags.tree:
		output.RenderContainerFallbackTree(outw, match, colorEnabled)
	case flags.warn:
		output.RenderContainerFallbackWarnings(outw, match, colorEnabled)
	case note != "":
		output.RenderProxiedContainer(outw, label, match, colorEnabled, flags.verbose, note)
	default:
		output.RenderContainerFallback(outw, label, match, colorEnabled, flags.verbose)
	}
	return ExitOK
}

func SetVersion(v string, c string, bd string) {
	version = v
	commit = c
	buildDate = bd

	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("witr {{.Version}} (commit %s, built %s)\n", commit, buildDate))
	rootCmd.SilenceUsage = true
}
