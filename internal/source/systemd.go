//go:build linux

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectSystemd(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		if p.PID == 1 && p.Command == "systemd" {
			return &model.Source{
				Type:       model.SourceSystemd,
				Name:       "systemd",
				Confidence: 0.8,
			}
		}
	}
	return nil
}
