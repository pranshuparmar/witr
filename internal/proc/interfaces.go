//go:build darwin

package proc

type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

type FileReader interface {
	ReadFile(path string) ([]byte, error)
}

type CommandRunnerFunc func(name string, args ...string) ([]byte, error)

func (f CommandRunnerFunc) Run(name string, args ...string) ([]byte, error) {
	return f(name, args...)
}

type FileReaderFunc func(path string) ([]byte, error)

func (f FileReaderFunc) ReadFile(path string) ([]byte, error) {
	return f(path)
}
