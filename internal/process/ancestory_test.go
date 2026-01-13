//go:build integration

package process

import (
	"os"
	"testing"
)

func TestBuildAncestry(t *testing.T) {
	chain, err := BuildAncestry(os.Getpid())
	if err != nil {
		t.Fatalf("BuildAncestry failed: %v", err)
	}
	if len(chain) == 0 {
		t.Error("BuildAncestry returned empty chain")
	}
}
