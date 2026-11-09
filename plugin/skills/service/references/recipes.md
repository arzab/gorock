# Recipes

Each recipe lists every file to touch. `MODULE` is the module path from `go.mod`. After any recipe: `gofmt -w`, `go build ./... && go vet ./...`, and start the service once if config changed.

## Contents
1. New project
2. HTTP endpoint
3. PostgreSQL repository (service + resource)
4. Redis cache service
5. Events (rockbus)
6. Scheduled jobs (rockcron)
7. Telegram bot (rocktelebot)
8. Custom App
9. Config field / new environment
10. Short-running command (migrations, CLI)

---

## 1. New project

Run the scaffold script, then build:

```sh
bash "${CLAUDE_SKILL_DIR}/scripts/scaffold.sh" github.com/acme/orders ./orders
cd orders && go run ./cmd/main   # GET localhost:8080/api/v1/hello?name=Ann
```

The script copies `templates/skeleton/` (layers, engine, HTTP app, logger, one endpoint, config, AGENTS.md, CLAUDE.md), sets the module path, runs `go get` for the kit modules and `go build`. If the target dir is not empty, it refuses — then create files by hand from `templates/skeleton/`.

## 2. HTTP endpoint

`internal/delivery/endpoints/create_order/endpoint.go`
```go
package create_order

import "github.com/arzab/gorock-kit/rockfiber"

func Endpoint() rockfiber.FiberEndpoint {
	return rockfiber.POST("/orders", handlers()...)
}
```

`internal/delivery/endpoints/create_order/params.go`
```go
package create_order

import (
	"net/http"

	"github.com/arzab/gorock-kit/rockfiber"
	"github.com/gofiber/fiber/v2"
)

type Params struct {
	Item string `json:"item"`
	Qty  int    `json:"qty"`
}

func (p *Params) Validate(*fiber.Ctx) error {
	if p.Item == "" || p.Qty <= 0 {
		return rockfiber.NewError(http.StatusBadRequest, "item and positive qty are required")
	}
	return nil
}

func InitParams() fiber.Handler { return rockfiber.DefaultHandler[Params]() }

func GetParams(c *fiber.Ctx) (*Params, error) {
	return rockfiber.GetFromContext[Params](c, "params")
}
```

`internal/delivery/endpoints/create_order/response.go`
```go
package create_order

import (
	"github.com/arzab/gorock-kit/rockfiber"
	"github.com/gofiber/fiber/v2"

	"MODULE/internal/models"
)

type Response struct {
	Order *models.Order `json:"order"`
}

const responseKey = "response"

func InitResponse() fiber.Handler { return rockfiber.HandlerInitInCtx[Response](responseKey) }

func GetResponse(c *fiber.Ctx) (*Response, error) {
	return rockfiber.GetFromContext[Response](c, responseKey)
}
```

`internal/delivery/endpoints/create_order/handlers.go`
```go
package create_order

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"MODULE/internal/models"
	"MODULE/pkg/services"
)

func handlers() []fiber.Handler {
	return []fiber.Handler{InitParams(), InitResponse(), createOrder(), sendResponse()}
}

func createOrder() fiber.Handler {
	return func(c *fiber.Ctx) error {
		params, err := GetParams(c)
		if err != nil {
			return err
		}
		resp, err := GetResponse(c)
		if err != nil {
			return err
		}
		order := &models.Order{ID: uuid.NewString(), Item: params.Item, Qty: params.Qty}
		if err := services.Repository().Insert(c.UserContext(), order); err != nil {
			return fmt.Errorf("insert order: %w", err) // → 500, message masked
		}
		resp.Order = order
		return c.Next()
	}
}

func sendResponse() fiber.Handler {
	return func(c *fiber.Ctx) error {
		resp, err := GetResponse(c)
		if err != nil {
			return err
		}
		return c.Status(http.StatusCreated).JSON(resp)
	}
}
```

Register in `internal/delivery/endpoints/endpoints.go`: add `create_order.Endpoint()` to `All()`.

Small endpoints may live in one file (see `templates/skeleton/internal/delivery/endpoints/hello/endpoint.go`). Path params: `rockfiber.GET("/orders/:id", ...)` + `ID string \`params:"id"\``. Not found → `rockfiber.NewError(http.StatusNotFound, "order not found")` after `errors.Is(err, repository.ErrNotFound)`.

## 3. PostgreSQL repository

`go get github.com/arzab/gorock-kit/rockbun@latest github.com/uptrace/bun@latest`

`pkg/services/repository/configs.go`
```go
package repository

import "github.com/arzab/gorock-kit/rockbun"

type Configs struct {
	Postgres rockbun.Configs
}
```

`pkg/services/repository/interface.go`
```go
package repository

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("repository: not found")

// Service stores any bun model; it doesn't import internal/models.
type Service interface {
	Init(ctx context.Context) error
	Stop() error
	Ping(ctx context.Context) error
	Insert(ctx context.Context, model any) error
	GetByID(ctx context.Context, model any, id string) error
	Update(ctx context.Context, model any) error
}
```

`pkg/services/repository/service.go`
```go
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/arzab/gorock-kit/rockbun"
)

type service struct {
	db rockbun.Service
}

func NewService(cfg Configs) Service {
	return &service{db: rockbun.NewService(cfg.Postgres)}
}

func (s *service) Init(ctx context.Context) error { return s.db.Init(ctx) }
func (s *service) Stop() error                    { return s.db.Stop() }
func (s *service) Ping(ctx context.Context) error { return s.db.Ping(ctx) }

func (s *service) Insert(ctx context.Context, model any) error {
	_, err := s.db.DB().NewInsert().Model(model).Exec(ctx)
	return err
}

func (s *service) GetByID(ctx context.Context, model any, id string) error {
	err := s.db.DB().NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *service) Update(ctx context.Context, model any) error {
	res, err := s.db.DB().NewUpdate().Model(model).WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
```

`pkg/services/services.go`
```go
package services

import (
	"github.com/arzab/gorock-kit/rockengine"

	"MODULE/pkg/services/repository"
)

type Configs struct {
	Repository repository.Configs
}

var repo repository.Service

func Repository() repository.Service { return repo }

func Register(e *rockengine.Engine, cfg Configs) {
	repo = repository.NewService(cfg.Repository)
	e.MustRegisterResource("repository", repo)
}
```
Add the section to the config file (all `rockbun.Configs` fields are optional; the password comes from env):

`configs/configs.yaml`
```yaml
services:
  repository:
    postgres:
      host: ${DB_HOST:-localhost}
      user: ${DB_USER:-postgres}
      password: ${DB_PASSWORD}
      database: ${DB_NAME:-app}
```

`internal/models/order.go`
```go
package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Order struct {
	bun.BaseModel `bun:"table:orders" json:"-"`

	ID        string    `bun:"id,pk" json:"id"`
	Item      string    `bun:"item,notnull" json:"item"`
	Qty       int       `bun:"qty,notnull" json:"qty"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
}
```
Schema: SQL files in `migrations/` (applied by your tooling or a short-running command, recipe 10).

## 4. Redis cache service

`go get github.com/arzab/gorock-kit/rockredis@latest`

`pkg/services/cache/` — `configs.go`: `type Configs struct { Redis rockredis.Configs }`; `interface.go`: `Service` with `Init(ctx) error`, `Stop() error` and the functionality you need (`Get(ctx, key) (string, bool, error)`, `Set(ctx, key, value, ttl) error`); `service.go`:

```go
type service struct{ r rockredis.Service }

func NewService(cfg Configs) Service { return &service{r: rockredis.NewService(cfg.Redis)} }

func (s *service) Init(ctx context.Context) error { return s.r.Init(ctx) }
func (s *service) Stop() error                    { return s.r.Stop() }

func (s *service) Get(ctx context.Context, key string) (string, bool, error) {
	v, err := s.r.Get(ctx, key)
	if errors.Is(err, rockredis.ErrNil) {
		return "", false, nil
	}
	return v, err == nil, err
}

func (s *service) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.r.Set(ctx, key, value, ttl)
}
```
Register in `services.Register` with `e.MustRegisterResource("cache", c)`; config section `services.cache.redis.addr: ${REDIS_ADDR:-localhost:6379}`.

## 5. Events (rockbus)

`go get github.com/arzab/gorock-kit/rockbus@latest`

`internal/delivery/events/events.go`
```go
package events

import (
	"context"

	"github.com/arzab/gorock-kit/rockbus"
	"github.com/arzab/gorock-kit/rocklog"

	"MODULE/internal/models"
)

const OrderCreated rockbus.Topic = "order.created"

func Subscriptions() []rockbus.Subscription {
	return []rockbus.Subscription{rockbus.On(OrderCreated, onOrderCreated)}
}

func onOrderCreated(ctx context.Context) error {
	order, err := rockbus.Payload[models.Order](ctx)
	if err != nil {
		return err
	}
	rocklog.FromContext(ctx).Info("order created", rocklog.Str("id", order.ID))
	return nil
}
```

`cmd/apps/apps.go` — add `Bus rockbus.Config \`config:",omitempty"\`` to `Configs`, and in `Register` **before** the HTTP app:
```go
bus := rockbus.NewApp(cfg.Bus, func(ctx context.Context, ev rockbus.Event, err error) {
	rocklog.FromContext(ctx).Error("event failed", rocklog.Str("topic", string(ev.Topic)), rocklog.Err(err))
}, events.Subscriptions()...)
rockbus.SetDefault(bus)
e.MustRegister("bus", bus)
```
Publish from any delivery: `rockbus.PublishAsync(ctx, rockbus.Event{Topic: events.OrderCreated, Payload: *order})`. To carry the trace id into the handler log: `ctx := rocklog.WithContext(c.UserContext(), rocklog.Str("trace_id", rockfiber.TraceIdFromContext(c.UserContext())))`.

## 6. Scheduled jobs (rockcron)

`go get github.com/arzab/gorock-kit/rockcron@latest`

`internal/delivery/jobs/jobs.go`
```go
package jobs

import (
	"context"
	"time"

	"github.com/arzab/gorock-kit/rockcron"

	"MODULE/pkg/services"
)

func All() []rockcron.Job {
	return []rockcron.Job{
		rockcron.Every("ping-db", time.Minute, func(ctx context.Context) error { return services.Repository().Ping(ctx) }),
		// rockcron.Cron("daily-report", "0 3 * * *", buildReport, sendReport),
	}
}
```

`cmd/apps/apps.go` — `Cron rockcron.Config \`config:",omitempty"\`` in `Configs`, and before the HTTP app:
```go
e.MustRegister("cron", rockcron.NewApp(cfg.Cron, func(ctx context.Context, job rockcron.Job, err error) {
	rocklog.Error("job failed", rocklog.Str("job", rockcron.JobName(job)), rocklog.Err(err))
}, jobs.All()...))
```
Optional config: `apps.cron.location: Europe/Moscow` (default UTC).

## 7. Telegram bot (rocktelebot)

`go get github.com/arzab/gorock-kit/rocktelebot@latest gopkg.in/telebot.v3@latest`

`internal/delivery/bot/bot.go`
```go
package bot

import (
	"github.com/arzab/gorock-kit/rocktelebot"
	tele "gopkg.in/telebot.v3"
)

func Handlers() []rocktelebot.Handler {
	return []rocktelebot.Handler{
		rocktelebot.Command("/start", onStart).WithDesc("Start the bot"),
		rocktelebot.GetHandler(tele.OnText, onText),
	}
}

func onStart(c tele.Context) error { return c.Send("Hi!") }
func onText(c tele.Context) error  { return c.Send("echo: " + c.Text()) }
```

`cmd/apps/apps.go` — `Bot rocktelebot.Config` in `Configs`, and:
```go
e.MustRegister("bot", rocktelebot.NewApp(cfg.Bot, []tele.MiddlewareFunc{
	rocktelebot.Recovery(func(c tele.Context, r interface{}) { rocklog.Error("bot panic", rocklog.Any("panic", r)) }),
}, bot.Handlers()...))
```
Config: `apps.bot.token: ${TELEGRAM_TOKEN}`. `Init` calls the Telegram API — a wrong token fails startup.

## 8. Custom App

Only when no ready-made app fits (queue consumer, gRPC, ...). `cmd/apps/<name>/configs.go` + `app.go`:

```go
type App struct {
	cfg     Configs
	conn    *client.Conn
	started atomic.Bool
}

func NewApp(cfg Configs /*, delivery entities */) *App { return &App{cfg: cfg} }

func (a *App) Init(ctx context.Context) error {
	conn, err := client.Dial(ctx, a.cfg.URL)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	a.conn = conn
	a.started.Store(false) // Init runs again on restart
	return nil
}

func (a *App) Exec(ctx context.Context) error {
	a.started.Store(true)
	defer a.conn.Close()
	for {
		msg, err := a.conn.Receive(ctx) // returns when ctx is cancelled
		if err != nil {
			return err // after ctx cancellation the engine treats any return as a clean stop
		}
		a.handle(ctx, msg) // recover panics, map errors to ack/nack
	}
}

// Exec reacts to ctx itself, so Stop doesn't need to unblock it. Stop must never
// wait for Exec: it is also called when Exec never ran (another app failed Init).
func (a *App) Stop() []error {
	if a.conn == nil || a.started.Load() {
		return nil
	}
	return []error{a.conn.Close()}
}
```
For a blocking `Exec` that ignores ctx (e.g. a server's `Serve()`), `Stop` calls the server's graceful shutdown instead. Register in `apps.Register` like the others.

## 9. Config field / new environment

1. Add the field to the right layer's `Configs` (snake_case key is derived). Optional → `config:",omitempty"` or `config:",default=X"`; secret → env reference in the file; set-in-code only → `config:"-"`.
2. Add the key to `configs/configs.yaml` and every other `configs/*.yaml`.
3. Start the service once: rockconfig lists missing fields and unknown keys together.

New environment: copy `configs/configs.yaml` to `configs/prod.yaml`, put secrets as `${VAR}`, run `./app ./configs/prod.yaml`.

## 10. Short-running command

The engine waits for a signal even after apps finish, so one-shot tasks don't use it. `cmd/migrate/main.go`:
```go
cfg, err := rockconfig.InitFromFile[Configs](path) // reuse the same Configs type or a smaller one
// ...
repo := repository.NewService(cfg.Services.Repository)
if err := repo.Init(ctx); err != nil {
	return err
}
defer repo.Stop()
return runMigrations(ctx, repo)
```
