package apps

import (
	"context"
	"time"

	"github.com/arzab/gorock-kit/rockbus"
	"github.com/arzab/gorock-kit/rockcron"
	"github.com/arzab/gorock-kit/rockengine"
	"github.com/arzab/gorock-kit/rockfiber"
	"github.com/arzab/gorock-kit/rocklog"

	"github.com/yourname/taskapi/internal/delivery/endpoints"
	"github.com/yourname/taskapi/internal/delivery/events"
	"github.com/yourname/taskapi/internal/delivery/jobs"
)

type Configs struct {
	HTTP rockfiber.Config
	Bus  rockbus.Config  `config:",omitempty"`
	Cron rockcron.Config `config:",omitempty"`
}

// Register creates every App and registers it with the engine.
// Order matters: apps stop in reverse, so the HTTP server (registered last)
// stops first and the bus it publishes to is still alive while it drains.
func Register(e *rockengine.Engine, cfg Configs) {
	bus := rockbus.NewApp(cfg.Bus, onBusError, events.Subscriptions()...)
	rockbus.SetDefault(bus)
	e.MustRegister("bus", bus)

	e.MustRegister("cron", rockcron.NewApp(cfg.Cron, onJobError, jobs.All()...))

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

func onBusError(ctx context.Context, event rockbus.Event, err error) {
	rocklog.FromContext(ctx).Error("event failed", rocklog.Str("topic", string(event.Topic)), rocklog.Err(err))
}

func onJobError(ctx context.Context, job rockcron.Job, err error) {
	rocklog.Error("job failed", rocklog.Str("job", rockcron.JobName(job)), rocklog.Err(err))
}
