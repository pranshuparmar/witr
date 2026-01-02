package source

import "github.com/SanCognition/witr/pkg/model"

var shells = map[string]bool{
	"bash": true,
	"zsh":  true,
	"sh":   true,
	"fish": true,
}

func detectShell(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		if shells[p.Command] {
			return &model.Source{
				Type:       model.SourceShell,
				Name:       p.Command,
				Confidence: 0.5,
			}
		}
	}
	return nil
}
