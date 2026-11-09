package apps

import (
	"time"

	"github.com/arzab/gorock-catalog/rockhttp"
	"github.com/arzab/gorock-kit/rockengine"

	"__MODULE__/internal/delivery/endpoints"
	"__MODULE__/internal/delivery/middleware"
)

type Configs struct {
	HTTP rockhttp.Config
}

// Register creates every App and registers it with the engine.
// Apps stop in reverse registration order: register the HTTP server last so it
// stops first, while the apps it depends on (event bus, consumers) still run.
func Register(e *rockengine.Engine, cfg Configs) {
	http := rockhttp.New(cfg.HTTP, endpoints.All()...).
		Use(middleware.Log) // rockhandlers middlewares around every route
	e.MustRegister("http", http, rockengine.RestartPolicy{MaxRetries: 3, Delay: time.Second, ResetAfter: time.Minute})
}
