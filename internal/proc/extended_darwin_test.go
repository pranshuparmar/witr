//go:build darwin

package proc

import "testing"

func TestParseLaunchctlLimitLine(t *testing.T) {
	for name, tc := range map[string]struct {
		line  string
		limit uint64
		valid bool
	}{
		"numeric":   {line: "maxfiles    1024            unlimited", limit: 1024, valid: true},
		"unlimited": {line: "maxfiles    unlimited       unlimited", limit: 0, valid: true},
		"invalid":   {line: "maxfiles --", limit: 0, valid: false},
		"short":     {line: "oops", limit: 0, valid: false},
	} {
		name := name
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			limit, ok := parseLaunchctlLimitLine(tc.line)
			if ok != tc.valid || limit != tc.limit {
				t.Fatalf("parseLaunchctlLimitLine(%q) = (%d, %t), want (%d, %t)", tc.line, limit, ok, tc.limit, tc.valid)
			}
		})
	}
}
