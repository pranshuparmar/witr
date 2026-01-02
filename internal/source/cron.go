package source

import "github.com/SanCognition/witr/pkg/model"

func detectCron(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		if p.Command == "cron" || p.Command == "crond" {
			return &model.Source{
				Type:       model.SourceCron,
				Name:       "cron",
				Confidence: 0.6,
			}
		}
	}
	return nil
}
