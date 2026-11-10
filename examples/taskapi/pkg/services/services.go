package services

import (
	"context"

	"github.com/arzab/gorock-kit/rockservices"

	"github.com/arzab/gorock/examples/taskapi/pkg/services/logger"
	"github.com/arzab/gorock/examples/taskapi/pkg/services/repository"
)

const (
	LoggerName     = "logger"
	RepositoryName = "repository"
)

type Configs struct {
	Logger     logger.Configs `config:",omitempty"`
	Repository repository.Configs
}

var catalog *rockservices.Catalog

// Init creates the services from their configs and starts them in order:
// the logger first, so the others can log while they start.
func Init(ctx context.Context, cfg Configs) error {
	catalog = rockservices.New(map[string]rockservices.Service{
		LoggerName:     logger.NewService(cfg.Logger),
		RepositoryName: repository.NewService(cfg.Repository),
	})
	return catalog.Init(ctx, LoggerName, RepositoryName)
}

// Stop stops the services in reverse order.
func Stop() error { return catalog.Stop() }

// Repository is the getter every layer uses.
func Repository() repository.Service { return catalog.Get(RepositoryName).(repository.Service) }

// UseCatalog swaps the catalog — for tests. They must not use t.Parallel().
func UseCatalog(c *rockservices.Catalog) (restore func()) {
	prev := catalog
	catalog = c
	return func() { catalog = prev }
}
