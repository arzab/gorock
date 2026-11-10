package services

import (
	"github.com/arzab/gorock-kit/rockservices"

	"github.com/arzab/gorock/examples/taskapi/pkg/services/logger"
	"github.com/arzab/gorock/examples/taskapi/pkg/services/repository"
)

// Keys: one per service, typed. Handlers use services.Repository.Get().
var (
	Logger     = rockservices.NewKey[logger.Service]("logger")
	Repository = rockservices.NewKey[repository.Service]("repository")
)

type Configs struct {
	Logger     logger.Configs `config:",omitempty"`
	Repository repository.Configs
}

// Register adds every service to the catalog. Add order = Init order: the
// logger goes first so the others can log while they start.
func Register(c *rockservices.Catalog, cfg Configs) {
	rockservices.Add(c, Logger, logger.NewService, cfg.Logger)
	rockservices.Add(c, Repository, repository.NewService, cfg.Repository)
}
