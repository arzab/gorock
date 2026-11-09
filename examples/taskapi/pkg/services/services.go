package services

import (
	"github.com/arzab/gorock-kit/rockengine"

	"github.com/yourname/taskapi/pkg/services/repository"
)

type Configs struct {
	Repository repository.Configs
}

var repo repository.Service

// Repository is the getter used by every layer.
func Repository() repository.Service { return repo }

// Register creates the services and hands their lifecycle to the engine:
// they are initialised before any App and stopped after all of them.
func Register(e *rockengine.Engine, cfg Configs) {
	repo = repository.NewService(cfg.Repository)
	e.MustRegisterResource("repository", repo)
}
