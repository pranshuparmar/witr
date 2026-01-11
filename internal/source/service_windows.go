//go:build windows

package source

import (
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func detectWindowsService(ancestry []model.Process) *model.Source {
	// Walk up the ancestry tree
	for _, p := range ancestry {
		cmd := strings.ToLower(p.Command)
		if cmd == "services.exe" {
			return &model.Source{
				Type: model.SourceWindowsService,
				Name: "Service Control Manager",
				Details: map[string]string{
					"manager": "services.exe",
				},
			}
		}
	}
	return nil
}
