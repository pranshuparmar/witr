//go:build !windows

package proc

// dockerBinaryCandidates is a no-op off Windows; PATH is the usual source.
func dockerBinaryCandidates() string { return "" }
