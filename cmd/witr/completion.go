package main

import (
	"fmt"
	"os"

	"github.com/pranshuparmar/witr/internal/completion"
)

// Exit codes for completion subcommands
const (
	exitOK         = 0
	exitInvalidArg = 1 // Invalid argument value (unknown type, unsupported shell)
	exitMissingArg = 2 // Missing required argument (usage error)
)

// handleComplete outputs completion candidates for dynamic completion
func handleComplete(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: witr __complete <pids|ports|processes>")
		os.Exit(exitMissingArg)
	}

	switch args[0] {
	case "pids":
		completion.Complete(completion.CompletePIDs)
	case "ports":
		completion.Complete(completion.CompletePorts)
	case "processes":
		completion.Complete(completion.CompleteProcesses)
	default:
		fmt.Fprintf(os.Stderr, "Unknown completion type: %s\n", args[0])
		os.Exit(exitInvalidArg)
	}
}

func handleCompletion(args []string) {
	if len(args) == 0 {
		printCompletionUsage()
		os.Exit(exitMissingArg)
	}

	shell := args[0]
	var script string

	switch shell {
	case "bash":
		script = completion.Bash()
	case "zsh":
		script = completion.Zsh()
	case "fish":
		script = completion.Fish()
	case "powershell", "pwsh":
		script = completion.PowerShell()
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported shell %q\n", shell)
		fmt.Fprintf(os.Stderr, "Supported shells: bash, zsh, fish, powershell, pwsh\n")
		os.Exit(exitInvalidArg)
	}

	fmt.Print(script)
}

func printCompletionUsage() {
	fmt.Println("Usage: witr completion <shell>")
	fmt.Println()
	fmt.Println("Generate shell completion scripts for witr.")
	fmt.Println()
	fmt.Println("Supported shells:")
	fmt.Println("  bash        Generate bash completion script")
	fmt.Println("  zsh         Generate zsh completion script")
	fmt.Println("  fish        Generate fish completion script")
	fmt.Println("  powershell  Generate PowerShell completion script")
	fmt.Println("  pwsh        Generate PowerShell (Core) completion script")
	fmt.Println()
	fmt.Println("Installation:")
	fmt.Println()
	fmt.Println("  Bash:")
	fmt.Println("    # Add to ~/.bashrc:")
	fmt.Println("    source <(witr completion bash)")
	fmt.Println()
	fmt.Println("  Zsh:")
	fmt.Println("    # Add to ~/.zshrc:")
	fmt.Println("    source <(witr completion zsh)")
	fmt.Println()
	fmt.Println("  Fish:")
	fmt.Println("    witr completion fish > ~/.config/fish/completions/witr.fish")
	fmt.Println()
	fmt.Println("  PowerShell:")
	fmt.Println("    # Add to $PROFILE:")
	fmt.Println("    witr completion powershell | Out-String | Invoke-Expression")
}
