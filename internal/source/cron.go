package source

import (
	"path/filepath"

	"github.com/pranshuparmar/witr/pkg/model"
)

func detectCron(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		base := filepath.Base(p.Command)
		if base == "cron" || base == "crond" {
			return &model.Source{
				Type: model.SourceCron,
				Name: "cron",
			}
		}
	}
	return nil
}
