//go:build !darwin && !linux

package completion

// getRunningPIDs returns a list of all running PIDs.
// Not implemented for this platform.
func getRunningPIDs() []string {
	return nil
}

// getListeningPorts returns a list of all listening TCP ports.
// Not implemented for this platform.
func getListeningPorts() []string {
	return nil
}

// getProcessNames returns a list of unique running process names.
// Not implemented for this platform.
func getProcessNames() []string {
	return nil
}
