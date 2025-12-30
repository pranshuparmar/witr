package proc

import "os/exec"

//go:generate mockgen -destination=mocks/mock_executor.go -package=mocks github.com/pranshuparmar/witr/internal/proc Executor

type Executor interface {
	Run(name string, args ...string) ([]byte, error)
}

type RealExecutor struct{}

func (r *RealExecutor) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

var executor Executor = &RealExecutor{}

func SetExecutor(e Executor) {
	executor = e
}

func ResetExecutor() {
	executor = &RealExecutor{}
}

// Run executes a command using the current executor
func Run(name string, args ...string) ([]byte, error) {
	return executor.Run(name, args...)
}
