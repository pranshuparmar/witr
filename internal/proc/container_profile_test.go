package proc

import "testing"

// A numeric -p value is a port or argument, not a container profile name.
func TestContainerProfileRejectsNumericValue(t *testing.T) {
	t.Parallel()

	if got, want := detectContainerFromCmdline("minikube start -p 8080"), "kubernetes"; got != want {
		t.Errorf("minikube = %q, want %q", got, want)
	}
	if got, want := detectContainerFromCmdline("colima start -p 1234"), "colima: default"; got != want {
		t.Errorf("colima = %q, want %q", got, want)
	}
}
