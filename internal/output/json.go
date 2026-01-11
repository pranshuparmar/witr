package output

import (
	"encoding/json"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ToJSON(r model.Result) (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
