package apps

import (
	"context"

	"github.com/arzab/gorock-catalog/rockhttp"
	"github.com/arzab/gorock-kit/rockengine"

	"__MODULE__/internal/handlers/middleware"
)

type Configs struct {
	HTTP rockhttp.Config
}

// Run starts every App and blocks until SIGINT/SIGTERM or an App fails.
// Apps stop in reverse order: list HTTP last so it stops first.
func Run(ctx context.Context, cfg Configs) error {
	engine := rockengine.New(map[string]rockengine.App{
		"http": rockhttp.New(cfg.HTTP, httpEndpoints()...).Use(middleware.Log),
	})
	return engine.Run(ctx, "http")
}
