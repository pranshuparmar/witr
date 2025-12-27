//go:build darwin

package source

import "github.com/pranshuparmar/witr/pkg/model"

func Detect(ancestry []model.Process) model.Source {
	// Skip container detection on macOS (Docker runs in VM)
	if src := detectSupervisor(ancestry); src != nil {
		return *src
	}
	if src := detectLaunchd(ancestry); src != nil {
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

func detectLaunchd(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		if p.PID == 1 && p.Command == "launchd" {
			return &model.Source{
				Type:       model.SourceLaunchd,
				Name:       "launchd",
				Confidence: 0.8,
			}
		}
	}
	return nil
}
