package proc

import "github.com/pranshuparmar/witr/pkg/model"

func init() { registerRuntime(dockerRuntime{}) }

type dockerRuntime struct{}

func (dockerRuntime) Name() string                       { return "docker" }
func (dockerRuntime) Available() bool                    { return binAvailable("docker") }
func (dockerRuntime) List() []*model.ContainerMatch      { return dockerLikeList("docker", "docker") }
func (dockerRuntime) HostPID(id string) int              { return dockerLikeHostPID("docker", id) }
func (dockerRuntime) Enrich(match *model.ContainerMatch) { dockerLikeEnrich("docker", match) }
