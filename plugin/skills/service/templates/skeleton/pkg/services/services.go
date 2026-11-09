package services

import (
	"github.com/arzab/gorock-kit/rockservices"
)

// Configs aggregates the configs of every service. Add one field per service:
//
//	Repository repository.Configs
type Configs struct{}

// Register adds the services to the catalog. The add order is the Init
// order; a constructor may Get services added before it. cmd/main then calls
// rockservices.Init before the apps and rockservices.Stop after them.
//
//	rockservices.Add(c, repository.NewService, cfg.Repository)
//
// Each service package has a getter: repository.Get().
func Register(c *rockservices.Catalog, cfg Configs) {
	_, _ = c, cfg
}
