//go:build linux || darwin

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/SanCognition/witr/internal/batch"
	"github.com/SanCognition/witr/internal/output"
	"github.com/SanCognition/witr/internal/tui"
)

func runPS(args []string) {
	// Reorder args so flags come before positional arguments
	// This allows: witr ps node --sort cpu (flags after pattern)
	reordered := reorderPSArgs(args)

	fs := flag.NewFlagSet("ps", flag.ExitOnError)
	noColor := fs.Bool("no-color", false, "disable colors")
	sortBy := fs.String("sort", "", "sort by: cpu, mem, age, pid")
	jsonOut := fs.Bool("json", false, "output as JSON")
	watch := fs.Bool("watch", false, "interactive TUI mode with live refresh")
	fs.Parse(reordered)

	if fs.NArg() == 0 {
		fmt.Println("Usage: witr ps <pattern> [flags]")
		fmt.Println()
		fmt.Println("List all processes matching a pattern with detailed info.")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  witr ps node              # All node processes")
		fmt.Println("  witr ps node --sort cpu   # Sorted by CPU usage (highest first)")
		fmt.Println("  witr ps node --sort mem   # Sorted by memory usage")
		fmt.Println("  witr ps node --sort age   # Sorted by age (oldest first)")
		fmt.Println("  witr ps python --json     # JSON output")
		fmt.Println("  witr ps node --watch      # Interactive TUI with live refresh")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
		os.Exit(1)
	}

	pattern := fs.Arg(0)
	colorEnabled := !*noColor

	// Watch mode: launch interactive TUI
	if *watch {
		runWatchMode(pattern, *sortBy)
		return
	}

	start := time.Now()

	// 1. Discover all matching PIDs (fast)
	pids, err := batch.DiscoverPIDs(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(pids) == 0 {
		fmt.Printf("No processes matching %q found\n", pattern)
		os.Exit(0)
	}

	// 2. Create table renderer
	// Note: If sorting or JSON, we buffer all results before printing
	// For JSON or sorting, always buffer (set sortBy to force buffering)
	effectiveSort := *sortBy
	if *jsonOut && effectiveSort == "" {
		effectiveSort = "pid" // Use any sort to force buffering for JSON
	}
	table := output.NewTableRenderer(colorEnabled, effectiveSort)

	// Print header immediately for streaming mode (no sort, no json)
	if *sortBy == "" && !*jsonOut {
		table.PrintHeader()
	}

	// 3. Collect results (stream or buffer based on mode)
	results := batch.AnalyzeAsync(pids, 10) // 10 concurrent workers

	total := 0
	errors := 0
	for summary := range results {
		if summary.Error != nil {
			errors++
			continue
		}
		table.AddRow(summary)
		total++
	}

	elapsed := time.Since(start)

	// 4. Handle JSON output
	if *jsonOut {
		output.PrintBatchJSON(table.GetRows())
		return
	}

	// 5. If sorting, print header then sorted rows
	if *sortBy != "" {
		table.PrintHeader()
		table.Flush()
	}

	// 6. Print footer
	table.PrintFooter(total, errors, elapsed)
}

// reorderPSArgs moves flags before positional arguments
func reorderPSArgs(args []string) []string {
	var flags []string
	var positionals []string

	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Check if this flag needs a value
			if (arg == "-sort" || arg == "--sort") && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
		} else {
			positionals = append(positionals, arg)
		}
		i++
	}

	return append(flags, positionals...)
}

// runWatchMode launches the interactive TUI
func runWatchMode(pattern, sortBy string) {
	model := tui.New(pattern, sortBy)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
