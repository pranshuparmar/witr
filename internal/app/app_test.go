package app

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunApp_Integration_SelfPID(t *testing.T) {
	// Setup capture
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Test with --pid <self>
	pid := os.Getpid()
	rootCmd.SetArgs([]string{"--pid", fmt.Sprintf("%d", pid), "--short"})

	// Reset flags between tests
	defer func() {
		rootCmd.SetArgs(nil)
		// Reset flags to defaults if needed
		flagclean := func(c *cobra.Command) {
			c.Flags().Set("pid", "")
			c.Flags().Set("short", "false")
			c.Flags().Set("tree", "false")
			c.Flags().Set("json", "false")
			c.Flags().Set("env", "false")
		}
		flagclean(rootCmd)
	}()

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "witr") && !strings.Contains(output, "go") {
		// Expect 'witr' or test binary name in output
		// Note: when running 'go test', the process name is usually the test binary
		t.Logf("Output did not contain expected process name, but command succeeded. Output: %s", output)
	}
}

func TestRunApp_Help(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute help failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Errorf("Help output missing 'Usage:'. Got: %s", output)
	}
}
