package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectCron(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		if p.Command == "cron" || p.Command == "crond" {
			return &model.Source{
				Type: model.SourceCron,
				Name: "cron",
			}
		}
	}
	return nil
}
