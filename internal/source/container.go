package source

import (
	"os"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
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
				Type: model.SourceContainer,
				Name: "docker",
			}
		case strings.Contains(content, "podman"), strings.Contains(content, "libpod"):
			return &model.Source{
				Type: model.SourceContainer,
				Name: "podman",
			}
		case strings.Contains(content, "kubepods"):
			return &model.Source{
				Type: model.SourceContainer,
				Name: "kubernetes",
			}
		case strings.Contains(content, "colima"):
			return &model.Source{
				Type: model.SourceContainer,
				Name: "colima",
			}
		case strings.Contains(content, "containerd"):
			// Only match containerd if not already matched by docker/kubernetes/colima
			return &model.Source{
				Type: model.SourceContainer,
				Name: "containerd",
			}
		}
	}
	return nil
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
