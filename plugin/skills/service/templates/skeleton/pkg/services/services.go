package services

import (
	"github.com/arzab/gorock-kit/rockservices"

	"__MODULE__/pkg/services/logger"
)

// Keys: one per service, typed. Use them anywhere: services.Logger.Get().
// A key declared here but not added in Register fails startup.
var (
	Logger = rockservices.NewKey[logger.Service]("logger")
)

// Configs aggregates the configs of every service. Add one field per service.
type Configs struct {
	Logger logger.Configs `config:",omitempty"`
}

// Register adds every service to the catalog. Add order = Init order; the
// logger goes first so the others can log while they start.
//
//	rockservices.Add(c, Orders, orders.NewService, cfg.Orders)
//
// A shared dependency is passed as the value Add returned:
//
//	pg := rockservices.Add(c, Postgres, postgres.NewService, cfg.Postgres)
//	rockservices.Add(c, Orders, func(cfg orders.Configs) orders.Service { return orders.NewService(cfg, pg) }, cfg.Orders)
func Register(c *rockservices.Catalog, cfg Configs) {
	rockservices.Add(c, Logger, logger.NewService, cfg.Logger)
}
