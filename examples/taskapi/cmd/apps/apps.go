package apps

import (
	"time"

	"github.com/arzab/gorock-catalog/rockhttp"
	"github.com/arzab/gorock-kit/rockengine"

	"github.com/arzab/gorock/examples/taskapi/internal/handlers/middleware"
)

type Configs struct {
	HTTP rockhttp.Config
}

// Register creates every App. Apps stop in reverse order: register HTTP last
// so it stops first, while the apps it depends on still run.
func Register(e *rockengine.Engine, cfg Configs) {
	http := rockhttp.New(cfg.HTTP, httpEndpoints()...).Use(middleware.Log)
	e.MustRegister("http", http, rockengine.RestartPolicy{MaxRetries: 3, Delay: time.Second, ResetAfter: time.Minute})
}
