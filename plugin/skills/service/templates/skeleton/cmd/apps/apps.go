package apps

import (
	"time"

	"github.com/arzab/gorock-kit/rockengine"
	"github.com/arzab/gorock-kit/rockfiber"
	"github.com/arzab/gorock-kit/rocklog"

	"__MODULE__/internal/delivery/endpoints"
)

type Configs struct {
	HTTP rockfiber.Config
}

// Register creates every App and registers it with the engine.
// Apps stop in reverse registration order: register the HTTP server last so it
// stops first, while the apps it depends on (event bus, ...) are still running.
func Register(e *rockengine.Engine, cfg Configs) {
	cfg.HTTP.OnError = func(traceId string, err *rockfiber.ErrorResponse) {
		log := rocklog.With(rocklog.Str("trace_id", traceId), rocklog.Int("code", err.Code))
		if err.Code >= 500 {
			log.Error("http error", rocklog.Str("error", err.Message)) // real message, even when masked for the client
			return
		}
		log.Warn("http client error", rocklog.Str("error", err.Message))
	}

	http := rockfiber.New(cfg.HTTP, endpoints.All()...)
	e.MustRegister("http", http, rockengine.RestartPolicy{MaxRetries: 3, Delay: time.Second, ResetAfter: time.Minute})
}
