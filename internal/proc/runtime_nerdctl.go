package proc

import "github.com/pranshuparmar/witr/pkg/model"

func init() { registerRuntime(nerdctlRuntime{}) }

type nerdctlRuntime struct{}

func (nerdctlRuntime) Name() string                       { return "containerd" }
func (nerdctlRuntime) Available() bool                    { return binAvailable("nerdctl") }
func (nerdctlRuntime) List() []*model.ContainerMatch      { return dockerLikeList("nerdctl", "containerd") }
func (nerdctlRuntime) HostPID(id string) int              { return dockerLikeHostPID("nerdctl", id) }
func (nerdctlRuntime) Enrich(match *model.ContainerMatch) { dockerLikeEnrich("nerdctl", match) }
