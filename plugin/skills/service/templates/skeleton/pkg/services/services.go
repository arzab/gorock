package services

import (
	"github.com/arzab/gorock-kit/rockengine"
)

// Configs aggregates the configs of every service. Add one field per service:
//
//	Repository repository.Configs
type Configs struct{}

// Getters for every layer go here:
//
//	var repo repository.Service
//	func Repository() repository.Service { return repo }

// Register creates the services and registers them as engine resources:
// they are initialised before any App and stopped after all of them.
//
//	repo = repository.NewService(cfg.Repository)
//	e.MustRegisterResource("repository", repo)
func Register(e *rockengine.Engine, cfg Configs) {
	_, _ = e, cfg
}
