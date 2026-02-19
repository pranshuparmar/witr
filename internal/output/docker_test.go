package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestRenderDockerFallback(t *testing.T) {
	match := &model.DockerPortMatch{
		ID:    "abc123",
		Name:  "my-container",
		Image: "nginx:latest",
		Ports: "0.0.0.0:8080->80/tcp",
	}

	var buf bytes.Buffer
	RenderDockerFallback(&buf, "8080", match, false)
	out := buf.String()

	expected := []string{
		"Target      : port 8080",
		"Container   : my-container",
		"Image       : nginx:latest",
		"Ports       : 0.0.0.0:8080->80/tcp",
		"Why It Exists",
		"Docker Desktop",
		"Source      : docker",
		"Note",
	}

	for _, want := range expected {
		if !strings.Contains(out, want) {
			t.Errorf("RenderDockerFallback output missing %q\nGot:\n%s", want, out)
		}
	}
}

func TestRenderDockerFallbackWithCompose(t *testing.T) {
	match := &model.DockerPortMatch{
		ID:             "abc123",
		Name:           "myapp-db-1",
		Image:          "postgres:16",
		Ports:          "0.0.0.0:5432->5432/tcp",
		ComposeProject: "myapp",
		ComposeService: "db",
	}

	var buf bytes.Buffer
	RenderDockerFallback(&buf, "5432", match, false)
	out := buf.String()

	if !strings.Contains(out, "docker-compose: myapp/db") {
		t.Errorf("expected compose source label, got:\n%s", out)
	}
}

func TestRenderDockerFallbackShort(t *testing.T) {
	match := &model.DockerPortMatch{
		ID:    "abc123",
		Name:  "my-container",
		Image: "nginx:latest",
		Ports: "0.0.0.0:8080->80/tcp",
	}

	var buf bytes.Buffer
	RenderDockerFallbackShort(&buf, "8080", match, false)
	out := buf.String()

	if !strings.Contains(out, "my-container") {
		t.Errorf("short output missing container name, got: %s", out)
	}
	if !strings.Contains(out, "nginx:latest") {
		t.Errorf("short output missing image, got: %s", out)
	}
	if !strings.Contains(out, "[docker]") {
		t.Errorf("short output missing source, got: %s", out)
	}
	// Should be a single line
	if strings.Count(out, "\n") != 1 {
		t.Errorf("short output should be single line, got: %s", out)
	}
}

func TestDockerFallbackToJSON(t *testing.T) {
	match := &model.DockerPortMatch{
		ID:             "abc123",
		Name:           "sql-proxy",
		Image:          "gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.13.0",
		Ports:          "127.0.0.1:5432->5432/tcp",
		ComposeProject: "",
		ComposeService: "",
	}

	jsonStr, err := DockerFallbackToJSON("5432", match)
	if err != nil {
		t.Fatalf("DockerFallbackToJSON() error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if result["Target"] != "port 5432" {
		t.Errorf("Target = %v, want %q", result["Target"], "port 5432")
	}
	if result["ContainerName"] != "sql-proxy" {
		t.Errorf("ContainerName = %v, want %q", result["ContainerName"], "sql-proxy")
	}
	if result["Source"] != "docker" {
		t.Errorf("Source = %v, want %q", result["Source"], "docker")
	}
}

func TestDockerFallbackToJSONCompose(t *testing.T) {
	match := &model.DockerPortMatch{
		ID:             "def456",
		Name:           "myapp-db-1",
		Image:          "postgres:16",
		Ports:          "0.0.0.0:5432->5432/tcp",
		ComposeProject: "myapp",
		ComposeService: "db",
	}

	jsonStr, err := DockerFallbackToJSON("5432", match)
	if err != nil {
		t.Fatalf("DockerFallbackToJSON() error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if result["Source"] != "docker-compose: myapp/db" {
		t.Errorf("Source = %v, want %q", result["Source"], "docker-compose: myapp/db")
	}
}

func TestRenderDockerFallbackSanitizesOutput(t *testing.T) {
	// Simulate a malicious container name with ANSI escape sequence
	match := &model.DockerPortMatch{
		ID:    "abc123",
		Name:  "evil\x1b[31mcontainer",
		Image: "evil\x1b[0mimage",
		Ports: "0.0.0.0:80->80/tcp",
	}

	var buf bytes.Buffer
	RenderDockerFallback(&buf, "80", match, false)
	out := buf.String()

	if strings.Contains(out, "\x1b") {
		t.Errorf("output contains raw ANSI escape sequences, sanitization failed:\n%s", out)
	}
}
