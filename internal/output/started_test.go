package output

import (
	"testing"
	"time"
)

func TestFormatStartedAt(t *testing.T) {
	if rel, abs := FormatStartedAt(time.Time{}); rel != "unknown" || abs != "" {
		t.Errorf("zero time = (%q, %q), want (\"unknown\", \"\")", rel, abs)
	}

	now := time.Now()
	cases := []struct {
		ago  time.Duration
		want string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5 min ago"},
		{90 * time.Minute, "1 hour ago"},
		{5 * time.Hour, "5 hours ago"},
		{30 * time.Hour, "1 day ago"},
		{72 * time.Hour, "3 days ago"},
	}
	for _, tc := range cases {
		rel, abs := FormatStartedAt(now.Add(-tc.ago))
		if rel != tc.want {
			t.Errorf("FormatStartedAt(-%v) rel = %q, want %q", tc.ago, rel, tc.want)
		}
		if abs == "" {
			t.Errorf("FormatStartedAt(-%v) absolute timestamp should not be empty", tc.ago)
		}
	}
}

// Runtimes report UTC; the absolute time shows in local time like every other.
func TestFormatStartedAtUsesLocalTime(t *testing.T) {
	ts := time.Date(2026, 10, 4, 9, 0, 0, 0, time.FixedZone("far", 13*3600))
	_, abs := FormatStartedAt(ts)
	if want := ts.Local().Format("Mon 2006-01-02 15:04:05 -07:00"); abs != want {
		t.Errorf("FormatStartedAt = %q, want local time %q", abs, want)
	}
}

func TestFormatStartedAtFutureTime(t *testing.T) {
	rel, _ := FormatStartedAt(time.Now().Add(2 * time.Hour))
	if rel == "just now" {
		t.Fatal("future start time reported as just now")
	}
	if want := "in 2h"; rel != want {
		t.Errorf("future start rel = %q, want %q", rel, want)
	}
}
