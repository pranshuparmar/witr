package proc

import "testing"

func TestExtractFlagValueEqualsSyntax(t *testing.T) {
	tests := []struct {
		name    string
		cmdline string
		flags   []string
		want    string
	}{
		{"docker name with equals", "docker run --name=myapp nginx", []string{"--name"}, "myapp"},
		{"minikube profile with equals", "minikube start --profile=dev", []string{"-p", "--profile"}, "dev"},
		{"colima short profile with equals", "colima start -p=work", []string{"-p", "--profile"}, "work"},
		{"apple container uuid with equals", "container-runtime-linux start --uuid=web", []string{"--uuid"}, "web"},
		{"space syntax still works", "docker run --name myapp", []string{"--name"}, "myapp"},
		{"empty equals value", "docker run --name=", []string{"--name"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractFlagValue(tt.cmdline, tt.flags...); got != tt.want {
				t.Errorf("extractFlagValue(%q, %v) = %q, want %q", tt.cmdline, tt.flags, got, tt.want)
			}
		})
	}
}
