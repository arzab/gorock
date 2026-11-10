package services

import (
	"github.com/arzab/gorock-kit/rockservices"

	"__MODULE__/pkg/services/logger"
)

// Names are given at registration and live here, with the getters.
const (
	LoggerName = "logger"
)

// Configs aggregates the configs of every service. Add one field per service.
type Configs struct {
	Logger logger.Configs `config:",omitempty"`
}

// Register adds every service to the catalog. Add order = Init order; the
// logger goes first so the others can log while they start.
//
//	rockservices.Add(c, OrdersName, orders.NewService, cfg.Orders)
//
// and a getter below:
//
//	func Orders() orders.Service { return rockservices.Get[orders.Service](OrdersName) }
func Register(c *rockservices.Catalog, cfg Configs) {
	rockservices.Add(c, LoggerName, logger.NewService, cfg.Logger)
}
