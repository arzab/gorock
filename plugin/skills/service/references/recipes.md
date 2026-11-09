# Recipes

Each recipe lists every file to touch. `MODULE` is the module path from `go.mod`. After any recipe: `gofmt -w`, `go build ./... && go vet ./...`, and start the service once if config changed.

Core = `github.com/arzab/gorock-kit/<module>` (rockengine, rockconfig, rockservices, rockhandlers). Catalog = `github.com/arzab/gorock-catalog/<module>` (rockhttp, rocklog, rockbun, rockredis, rockbus, rockcron, rocktelebot, rockfiber).

## Contents
1. New project
2. HTTP endpoint (route + endpoint)
3. Reusable step (auth, tenant…) and groups
4. Middleware (logging, metrics, timeouts)
5. PostgreSQL repository (service)
6. Redis cache service
7. Events (rockbus)
8. Scheduled jobs (rockcron)
9. Telegram bot (rocktelebot)
10. Custom App (queue consumer, gRPC…)
11. Config field / new environment
12. Short-running command (migrations, CLI)
13. Tests

---

## 1. New project

Run the scaffold script, then build:

```sh
bash "${CLAUDE_SKILL_DIR}/scripts/scaffold.sh" github.com/acme/orders ./orders
cd orders && go run ./cmd/main   # GET localhost:8080/api/v1/hello?name=Ann
```

The script copies `templates/skeleton/` (layers, services catalog, engine, rockhttp app, logger, log middleware, one route, config, AGENTS.md, CLAUDE.md), sets the module path, runs `go get` for the core and catalog modules and `go build`. If the target dir is not empty, it refuses — then create files by hand from `templates/skeleton/`.

## 2. HTTP endpoint (route + endpoint)

A **route** is a rockhandlers chain — it knows nothing about HTTP. The **endpoint** (method, path, status) mounts it in rockhttp.

`internal/delivery/endpoints/create_order/route.go`
```go
package create_order

import (
	"fmt"
	"strings"

	"github.com/arzab/gorock-kit/rockhandlers"
	"github.com/google/uuid"

	"MODULE/internal/models"
	"MODULE/pkg/services/repository"
)

type Params struct {
	ShopID string `path:"shop"`       // from the path pattern /shops/{shop}/orders
	DryRun bool   `query:"dry_run"`   // ?dry_run=true
	Item   string `json:"item"`       // JSON body
	Qty    int    `json:"qty"`
}

// Validate runs right after decoding, before every step.
func (p *Params) Validate() error {
	fields := map[string]string{}
	if strings.TrimSpace(p.Item) == "" {
		fields["item"] = "required"
	}
	if p.Qty <= 0 {
		fields["qty"] = "must be > 0"
	}
	if len(fields) > 0 {
		return rockhandlers.Invalid("invalid order").WithFields(fields) // → 400 with "fields"
	}
	return nil
}

type Result struct {
	Order *models.Order `json:"order"`
}

type Ctx = rockhandlers.Ctx[*Params, *Result]

var Route = rockhandlers.Chain("orders.create", create)

func create(c *Ctx) error {
	order := &models.Order{ID: uuid.NewString(), ShopID: c.Params.ShopID, Item: c.Params.Item, Qty: c.Params.Qty}
	if !c.Params.DryRun {
		if err := repository.Get().Insert(c, order); err != nil { // c is a context.Context
			return fmt.Errorf("insert order: %w", err) // → 500, message hidden
		}
	}
	c.Result.Order = order
	return nil
}
```

`internal/delivery/endpoints/endpoints.go` — add the endpoint:
```go
{Method: http.MethodPost, Path: "/shops/{shop}/orders", Handler: create_order.Route, Status: http.StatusCreated},
```

- Business errors: `rockhandlers.NotFound("order not found")` → 404, `Conflict` → 409, `Unauthorized` → 401, `Forbidden` → 403, `Unavailable` → 503. Plain errors/panics → 500 with a hidden message.
- Multi-step routes: `rockhandlers.Chain("orders.update", load, checkOwner, update)`; each step calls `c.Next()`; pass data with Locals (`var OrderKey = rockhandlers.NewLocal[*models.Order]("order")`).
- No body: result type `struct{}` → 204. Files: result `*rockhttp.File`; redirect: `*rockhttp.Redirect`; cookies/headers: `rockhttp.SetCookie(c, ...)`, `rockhttp.SetHeader(c, ...)`.
- Uploads: `File *multipart.FileHeader \`form:"file"\``.
- The `/api/v1` prefix comes from config (`apps.http.prefix`), not from the endpoint.

## 3. Reusable step and groups

A step that needs request data declares it with an interface constraint; models satisfy it by embedding. No casts.

`internal/delivery/steps/auth.go`
```go
package steps

import (
	"strings"

	"github.com/arzab/gorock-kit/rockhandlers"
)

type BearerAuth struct {
	Authorization string `header:"Authorization" json:"-"`
}

func (a *BearerAuth) BearerToken() string { return strings.TrimPrefix(a.Authorization, "Bearer ") }

type TokenParams interface{ BearerToken() string }

var UserKey = rockhandlers.NewLocal[string]("user")

func Bearer[P TokenParams, R any](c *rockhandlers.Ctx[P, R]) error {
	user, err := verify(c, c.Params.BearerToken())
	if err != nil {
		return rockhandlers.Unauthorized("invalid token")
	}
	UserKey.Set(c, user)
	return c.Next()
}

// Secured is a group: a plain generic function that prepends shared steps.
func Secured[P TokenParams, R any](name string, s ...rockhandlers.Handler[P, R]) rockhandlers.Route {
	return rockhandlers.Chain(name, rockhandlers.Prepend(s, Bearer[P, R])...)
}
```

Use: embed `steps.BearerAuth` in `Params`, then `var Route = steps.Secured("orders.create", create)` or `rockhandlers.Chain("orders.create", steps.Bearer, create)`. Forgetting the embed is a compile error. Read the user in later steps with `steps.UserKey.MustGet(c)`.

## 4. Middleware

For logic that doesn't look at request data. `internal/delivery/middleware/<name>.go`:
```go
func Log(c rockhandlers.Context) error {
	err := c.Next()
	if err != nil {
		rocklog.Error("request failed", rocklog.Str("route", c.Route()), rocklog.Str("kind", rockhandlers.KindOf(err).String()), rocklog.Err(err))
	}
	return err
}
```
App-wide: `rockhttp.New(cfg.HTTP, endpoints.All()...).Use(middleware.Log)`. One route: `Route.With(rockhandlers.Timeout(2*time.Second))`. Decoding errors also pass through middlewares.

## 5. PostgreSQL repository (service)

`go get github.com/arzab/gorock-catalog/rockbun@latest github.com/arzab/gorock-kit/rockservices@latest`

`pkg/services/repository/configs.go`
```go
package repository

import "github.com/arzab/gorock-catalog/rockbun"

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

	"github.com/arzab/gorock-kit/rockservices"
)

const Name = "repository"

var ErrNotFound = errors.New("repository: not found")

type Service interface {
	rockservices.Service
	Ping(ctx context.Context) error
	Insert(ctx context.Context, model any) error
	GetByID(ctx context.Context, model any, id string) error
	Update(ctx context.Context, model any) error
}

func Get() Service { return rockservices.Get[Service](Name) }
```

`pkg/services/repository/service.go`
```go
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/arzab/gorock-catalog/rockbun"
)

type service struct{ db rockbun.Service } // child service: repository owns it

func NewService(cfg Configs) Service { return &service{db: rockbun.NewService(cfg.Postgres)} } // no connections here

func (s *service) Name() string                    { return Name }
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

`pkg/services/services.go` — `Repository repository.Configs` in `Configs`, and in `Register`: `rockservices.Add(c, repository.NewService, cfg.Repository)`.
Config: `services.repository.postgres: {host: ${DB_HOST:-localhost}, user: ${DB_USER:-postgres}, password: ${DB_PASSWORD}, database: ${DB_NAME:-app}}`.
Model (`internal/models`): `bun.BaseModel \`bun:"table:orders"\`` + `bun:"id,pk"` tags. In delivery: `repository.Get().GetByID(c, c.Result, id)`; map `repository.ErrNotFound` to `rockhandlers.NotFound(...)`.

## 6. Redis cache service

`go get github.com/arzab/gorock-catalog/rockredis@latest`

`pkg/services/cache/` — `configs.go`: `type Configs struct { Redis rockredis.Configs }`; `interface.go`: `const Name = "cache"`, `Service` embedding `rockservices.Service` plus `Get(ctx, key) (string, bool, error)`, `Set(ctx, key, value, ttl) error`, and `func Get() Service`; `service.go`:

```go
type service struct{ r rockredis.Service }

func NewService(cfg Configs) Service { return &service{r: rockredis.NewService(cfg.Redis)} }

func (s *service) Name() string                    { return Name }
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
Add in `services.Register`: `rockservices.Add(c, cache.NewService, cfg.Cache)`; config `services.cache.redis.addr: ${REDIS_ADDR:-localhost:6379}`.

## 7. Events (rockbus)

`go get github.com/arzab/gorock-catalog/rockbus@latest` — not on rockhandlers yet; its handlers are `func(ctx) error`.

`internal/delivery/events/events.go`
```go
package events

import (
	"context"

	"github.com/arzab/gorock-catalog/rockbus"
	"github.com/arzab/gorock-catalog/rocklog"

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
Publish from a route step: `rockbus.PublishAsync(c, rockbus.Event{Topic: events.OrderCreated, Payload: *order})`.

## 8. Scheduled jobs (rockcron)

`go get github.com/arzab/gorock-catalog/rockcron@latest`

`internal/delivery/jobs/jobs.go`
```go
package jobs

import (
	"context"
	"time"

	"github.com/arzab/gorock-catalog/rockcron"

	"MODULE/pkg/services/repository"
)

func All() []rockcron.Job {
	return []rockcron.Job{
		rockcron.Every("ping-db", time.Minute, func(ctx context.Context) error { return repository.Get().Ping(ctx) }),
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

## 9. Telegram bot (rocktelebot)

`go get github.com/arzab/gorock-catalog/rocktelebot@latest gopkg.in/telebot.v3@latest`

`internal/delivery/bot/bot.go`
```go
package bot

import (
	"github.com/arzab/gorock-catalog/rocktelebot"
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

## 10. Custom App (queue consumer, gRPC…)

Only when no catalog app fits. With rockhandlers the App writes only the protocol part: decode into `Params` and map the result / error kind. `cmd/apps/<name>/configs.go` + `app.go`:

```go
package queue_consumer

type Consumer struct {
	Queue   string
	Handler rockhandlers.Route // the same routes HTTP uses
}

type App struct {
	cfg       Configs
	consumers []Consumer
	conn      *queue.Conn
	started   atomic.Bool
}

func NewApp(cfg Configs, consumers ...Consumer) *App { return &App{cfg: cfg, consumers: consumers} }

func (a *App) Init(ctx context.Context) error {
	conn, err := queue.Dial(ctx, a.cfg.URL)
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
			return err // after ctx cancel the engine treats it as a clean stop
		}
		a.handle(ctx, msg)
	}
}

func (a *App) handle(ctx context.Context, msg *queue.Message) {
	_, err := a.route(msg.Queue).Serve(ctx, func(params any) error {
		return json.Unmarshal(msg.Body, params) // plain error → Invalid
	})
	switch rockhandlers.KindOf(err) {
	case rockhandlers.KindNone:
		msg.Ack()
	case rockhandlers.KindUnavailable, rockhandlers.KindInternal, rockhandlers.KindCanceled, rockhandlers.KindTimeout:
		msg.Requeue()
	default:
		msg.Reject() // bad message: retrying won't help → DLQ
	}
}

// Stop must not wait for Exec: it is also called when Exec never ran.
func (a *App) Stop() []error {
	if a.conn == nil || a.started.Load() {
		return nil
	}
	return []error{a.conn.Close()}
}
```
`queue` is your queue client. Register in `apps.Register` before the HTTP app: `e.MustRegister("consumer", queue_consumer.NewApp(cfg.Consumer, queue_consumer.Consumer{Queue: "orders.create", Handler: create_order.Route}))`. Panics, validation and middlewares come from rockhandlers — don't re-implement them.

## 11. Config field / new environment

1. Add the field to the right layer's `Configs` (snake_case key is derived). Optional → `config:",omitempty"` or `config:",default=X"`; secret → env reference in the file; set-in-code only → `config:"-"`.
2. Add the key to `configs/configs.yaml` and every other `configs/*.yaml`.
3. Start the service once: rockconfig lists missing fields and unknown keys together.

New environment: copy `configs/configs.yaml` to `configs/prod.yaml`, put secrets as `${VAR}`, run `./app ./configs/prod.yaml`.

## 12. Short-running command

The engine waits for a signal even after apps finish, so one-shot tasks don't use it. `cmd/migrate/main.go`:
```go
cfg, err := rockconfig.InitFromFile[Configs](path) // reuse the same Configs type or a smaller one
// ...
services.Register(rockservices.Default(), cfg.Services)
if err := rockservices.Init(ctx); err != nil {
	return err
}
defer rockservices.Stop()
return runMigrations(ctx, repository.Get())
```

## 13. Tests

Route without HTTP, with fake services:
```go
func TestCreateOrder(t *testing.T) {
	c := rockservices.New()
	rockservices.Add(c, func(struct{}) repository.Service { return &fakeRepo{} }, struct{}{})
	prev := rockservices.SetDefault(c)
	t.Cleanup(func() { rockservices.SetDefault(prev) })

	res, err := rockhandlers.Run[*create_order.Result](context.Background(), create_order.Route,
		&create_order.Params{ShopID: "s1", Item: "tea", Qty: 1})
	if err != nil || res.Order.Item != "tea" {
		t.Fatalf("res %+v, err %v", res, err)
	}
}
```
HTTP level: `app := rockhttp.New(cfg, endpoints.All()...)`, `app.Init(ctx)`, then `httptest` against `app.Handler()`.
