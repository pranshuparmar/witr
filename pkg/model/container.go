package model

import "time"

type ContainerMatch struct {
	Runtime           string
	ID                string
	Name              string
	Image             string
	Command           string
	State             string
	Status            string
	Health            string
	CreatedAt         time.Time
	StartedAt         time.Time
	Networks          string
	Mounts            string
	Ports             string
	ComposeProject    string `json:",omitempty"`
	ComposeService    string `json:",omitempty"`
	ComposeConfigFile string `json:",omitempty"`
	ComposeWorkingDir string `json:",omitempty"`
}
