//go:build windows

package target

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseWindowsBoundPortOutput(t *testing.T) {
	output := `
  TCP    0.0.0.0:3000       0.0.0.0:0          LISTENING       42
  TCP    [::]:3000          [::]:0             LISTENING       84
  TCP    127.0.0.1:52100    127.0.0.1:3000     ESTABLISHED     99
  UDP    0.0.0.0:3000       *:*                                 42
  TCP    0.0.0.0:30000      0.0.0.0:0          LISTENING       100
`
	pids, err := parseWindowsBoundPortOutput(3000, output)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if want := []int{42, 84}; !reflect.DeepEqual(pids, want) {
		t.Fatalf("pids = %v, want %v", pids, want)
	}
}

func TestParseWindowsBoundPortOutputNotBound(t *testing.T) {
	_, err := parseWindowsBoundPortOutput(3000, "")
	if !errors.Is(err, ErrPortNotBound) {
		t.Fatalf("error = %v, want ErrPortNotBound", err)
	}
}
