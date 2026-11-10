package services

import (
	"context"

	"github.com/arzab/gorock-kit/rockservices"

	"__MODULE__/pkg/services/logger"
)

const (
	LoggerName = "logger"
)

// Configs aggregates the configs of every service. Add one field per service.
type Configs struct {
	Logger logger.Configs `config:",omitempty"`
}

var catalog *rockservices.Catalog

// Init creates the services from their configs and starts them in order:
// the logger first, so the others can log while they start.
//
//	RepositoryName: repository.NewService(cfg.Repository),   // in the map
//	catalog.Init(ctx, LoggerName, RepositoryName)            // in the order
func Init(ctx context.Context, cfg Configs) error {
	catalog = rockservices.New(map[string]rockservices.Service{
		LoggerName: logger.NewService(cfg.Logger),
	})
	return catalog.Init(ctx, LoggerName)
}

// Stop stops the services in reverse order.
func Stop() error { return catalog.Stop() }

// Getters, one per service:
//
//	func Repository() repository.Service { return catalog.Get(RepositoryName).(repository.Service) }

// UseCatalog swaps the catalog — for tests. They must not use t.Parallel().
func UseCatalog(c *rockservices.Catalog) (restore func()) {
	prev := catalog
	catalog = c
	return func() { catalog = prev }
}
