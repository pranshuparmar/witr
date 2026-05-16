package proc

import "github.com/pranshuparmar/witr/pkg/model"

func init() { registerRuntime(podmanRuntime{}) }

type podmanRuntime struct{}

func (podmanRuntime) Name() string                       { return "podman" }
func (podmanRuntime) Available() bool                    { return binAvailable("podman") }
func (podmanRuntime) List() []*model.ContainerMatch      { return dockerLikeList("podman", "podman") }
func (podmanRuntime) HostPID(id string) int              { return dockerLikeHostPID("podman", id) }
func (podmanRuntime) Enrich(match *model.ContainerMatch) { dockerLikeEnrich("podman", match) }
