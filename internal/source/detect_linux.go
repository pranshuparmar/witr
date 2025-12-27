//go:build linux

package source

import "github.com/pranshuparmar/witr/pkg/model"

func Detect(ancestry []model.Process) model.Source {
	if src := detectContainer(ancestry); src != nil {
		return *src
	}
	if src := detectSupervisor(ancestry); src != nil {
		return *src
	}
	if src := detectSystemd(ancestry); src != nil {
		return *src
	}
	if src := detectCron(ancestry); src != nil {
		return *src
	}
	if src := detectShell(ancestry); src != nil {
		return *src
	}

	return model.Source{
		Type:       model.SourceUnknown,
		Confidence: 0.2,
	}
}
