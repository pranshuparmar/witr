package target

import (
	"fmt"
	"strconv"

	"github.com/SanCognition/witr/pkg/model"
)

func Resolve(t model.Target) ([]int, error) {
	switch t.Type {
	case model.TargetPID:
		pid, err := strconv.Atoi(t.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid pid")
		}
		return []int{pid}, nil

	case model.TargetPort:
		port, err := strconv.Atoi(t.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid port")
		}
		return ResolvePort(port)

	case model.TargetName:
		return ResolveName(t.Value)

	default:
		return nil, fmt.Errorf("unknown target")
	}
}
