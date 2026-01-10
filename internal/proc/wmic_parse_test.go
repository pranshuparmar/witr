package proc

import (
	"testing"
)

func TestParseWmicServiceName(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantName string
	}{
		{
			name:     "single match",
			output:   "\n\nName=spooler\n\n",
			wantName: "spooler",
		},
		{
			name:     "no match",
			output:   "\n\nNo Instance(s) Available.\n\n",
			wantName: "",
		},
		{
			name:     "multiple lines with match",
			output:   "Node=WORKSTATION\nName=spooler\nState=Running",
			wantName: "spooler",
		},
		{
			name:     "empty output",
			output:   "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseWmicServiceNameInternal(tt.output); got != tt.wantName {
				t.Errorf("parseWmicServiceNameInternal() = %v, want %v", got, tt.wantName)
			}
		})
	}
}
