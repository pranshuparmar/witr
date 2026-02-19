package model

// DockerPortMatch holds information about a Docker container that publishes a specific port.
type DockerPortMatch struct {
	ID             string
	Name           string
	Image          string
	Ports          string
	ComposeProject string
	ComposeService string
}
