package source

import (
	"os"
	"strconv"
	"strings"

	"github.com/SanCognition/witr/pkg/model"
)

func detectContainer(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		data, err := os.ReadFile("/proc/" + itoa(p.PID) + "/cgroup")
		if err != nil {
			continue
		}
		content := string(data)

		switch {
		case strings.Contains(content, "docker"):
			return &model.Source{
				Type:       model.SourceContainer,
				Name:       "docker",
				Confidence: 0.9,
			}
		case strings.Contains(content, "podman"), strings.Contains(content, "libpod"):
			return &model.Source{
				Type:       model.SourceContainer,
				Name:       "podman",
				Confidence: 0.9,
			}
		case strings.Contains(content, "kubepods"):
			return &model.Source{
				Type:       model.SourceContainer,
				Name:       "kubernetes",
				Confidence: 0.9,
			}
		case strings.Contains(content, "colima"):
			return &model.Source{
				Type:       model.SourceContainer,
				Name:       "colima",
				Confidence: 0.9,
			}
		case strings.Contains(content, "containerd"):
			// Only match containerd if not already matched by docker/kubernetes/colima
			return &model.Source{
				Type:       model.SourceContainer,
				Name:       "containerd",
				Confidence: 0.8,
			}
		}
	}
	return nil
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
