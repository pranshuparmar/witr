package output

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func jsonFixture() model.Result {
	target := model.Process{
		PID:     1234,
		Command: "nginx",
		Cmdline: "/usr/sbin/nginx -g daemon off;",
		Env:     []string{"FOO=bar", "PATH=/usr/bin"},
	}
	return model.Result{
		Target:   model.Target{Type: model.TargetName, Value: "nginx"},
		Process:  target,
		Ancestry: []model.Process{{PID: 1, Command: "systemd"}, target},
		Children: []model.Process{{PID: 5678, Command: "worker"}},
		Source:   model.Source{Type: model.SourceSystemd, Name: "nginx.service"},
		Warnings: []string{"Process is listening on a public interface"},
	}
}

// TestToJSONIsValidAndIncludesKeyFields verifies the full result serializes
// cleanly and includes the top-level fields downstream consumers depend on.
func TestToJSONIsValidAndIncludesKeyFields(t *testing.T) {
	t.Parallel()

	s, err := ToJSON(jsonFixture())
	if err != nil {
		t.Fatalf("ToJSON returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(s), &decoded); err != nil {
		t.Fatalf("ToJSON output is not valid JSON: %v\n%s", err, s)
	}

	for _, key := range []string{"Target", "Process", "Ancestry", "Source", "Warnings"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("ToJSON output missing top-level field %q", key)
		}
	}
}

// TestToJSONOmitsEnv keeps environment variables, which often hold secrets,
// out of the full report: only --env --json prints them.
func TestToJSONOmitsEnv(t *testing.T) {
	t.Parallel()

	s, err := ToJSON(jsonFixture())
	if err != nil {
		t.Fatalf("ToJSON returned error: %v", err)
	}
	if strings.Contains(s, `"Env"`) || strings.Contains(s, "FOO=bar") {
		t.Errorf("ToJSON should not include environment variables; got:\n%s", s)
	}
}

func TestToShortJSON(t *testing.T) {
	t.Parallel()

	s, err := ToShortJSON(jsonFixture())
	if err != nil {
		t.Fatalf("ToShortJSON: %v", err)
	}

	var got []struct {
		PID     int
		Command string
	}
	if err := json.Unmarshal([]byte(s), &got); err != nil {
		t.Fatalf("ToShortJSON not parseable as []shortProcess: %v\n%s", err, s)
	}
	if len(got) != 2 {
		t.Fatalf("ToShortJSON length = %d, want 2", len(got))
	}
	if got[0].Command != "systemd" || got[1].Command != "nginx" {
		t.Errorf("ToShortJSON unexpected ancestry: %+v", got)
	}
}

func TestToTreeJSONIncludesChildren(t *testing.T) {
	t.Parallel()

	s, err := ToTreeJSON(jsonFixture())
	if err != nil {
		t.Fatalf("ToTreeJSON: %v", err)
	}

	var got struct {
		Ancestry []struct{ PID int }
		Children []struct {
			PID     int
			Command string
		}
	}
	if err := json.Unmarshal([]byte(s), &got); err != nil {
		t.Fatalf("ToTreeJSON not parseable: %v\n%s", err, s)
	}
	if len(got.Children) != 1 || got.Children[0].PID != 5678 || got.Children[0].Command != "worker" {
		t.Errorf("ToTreeJSON children = %+v, want one worker pid 5678", got.Children)
	}
}

// TestToTreeJSONOmitsEmptyChildren pins the `omitempty` contract — callers
// scripting against the JSON shouldn't need to special-case "Children":[].
func TestToTreeJSONOmitsEmptyChildren(t *testing.T) {
	t.Parallel()

	fx := jsonFixture()
	fx.Children = nil

	s, err := ToTreeJSON(fx)
	if err != nil {
		t.Fatalf("ToTreeJSON: %v", err)
	}
	if strings.Contains(s, `"Children"`) {
		t.Errorf("ToTreeJSON should omit Children when empty; got:\n%s", s)
	}
}

func TestToWarningsJSON(t *testing.T) {
	t.Parallel()

	s, err := ToWarningsJSON(jsonFixture())
	if err != nil {
		t.Fatalf("ToWarningsJSON: %v", err)
	}

	var got struct {
		PID      int
		Process  string
		Command  string
		Warnings []string
	}
	if err := json.Unmarshal([]byte(s), &got); err != nil {
		t.Fatalf("ToWarningsJSON not parseable: %v\n%s", err, s)
	}
	if got.PID != 1234 || got.Process != "nginx" {
		t.Errorf("ToWarningsJSON identity wrong: %+v", got)
	}
	if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "public interface") {
		t.Errorf("ToWarningsJSON warnings = %v", got.Warnings)
	}
}

// TestToWarningsJSONNilWarningsBecomesEmptyArray ensures downstream
// consumers always see [] and never null — important for jq scripts that do
// `.Warnings | length`.
func TestToWarningsJSONNilWarningsBecomesEmptyArray(t *testing.T) {
	t.Parallel()

	fx := jsonFixture()
	fx.Warnings = nil

	s, err := ToWarningsJSON(fx)
	if err != nil {
		t.Fatalf("ToWarningsJSON: %v", err)
	}
	if !strings.Contains(s, `"Warnings": []`) {
		t.Errorf("ToWarningsJSON should emit Warnings:[] when nil; got:\n%s", s)
	}
}

func TestToEnvJSON(t *testing.T) {
	t.Parallel()

	s, err := ToEnvJSON(jsonFixture())
	if err != nil {
		t.Fatalf("ToEnvJSON: %v", err)
	}

	var got struct {
		PID     int
		Process string
		Command string
		Env     []string
	}
	if err := json.Unmarshal([]byte(s), &got); err != nil {
		t.Fatalf("ToEnvJSON not parseable: %v\n%s", err, s)
	}
	if got.PID != 1234 {
		t.Errorf("ToEnvJSON PID = %d, want 1234", got.PID)
	}
	if len(got.Env) != 2 || got.Env[0] != "FOO=bar" {
		t.Errorf("ToEnvJSON env = %v", got.Env)
	}
}
