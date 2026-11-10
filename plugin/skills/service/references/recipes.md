# Recipes

Each recipe lists every file to touch. `MODULE` is the module path from `go.mod`. After any recipe: `gofmt -w`, `go build ./... && go vet ./...`, `go test ./...`, and start the project once if config changed.

Layers (bottom-up): **Services** (`pkg/services/`) → **Handlers** (`internal/handlers/`) → **App** (`cmd/apps/`) → **Main** (`cmd/main/`) → **Config** (`configs/`).
Core: `github.com/arzab/gorock-kit/<module>` (rockservices, rockhandlers, rockengine, rockconfig). Catalog: `github.com/arzab/gorock-catalog/<module>` (rockhttp, rocklog, rockbun, rockredis, rockbus, rockcron, rocktelebot, rockfiber).

## Contents
1. New project
2. Service (PostgreSQL repository)
3. Service over another technology (Redis, external API) or logic
4. Handler + mount it on HTTP
5. Shared step (auth) and groups
6. Middleware (logging, metrics, timeouts)
7. Custom App (queue consumer, gRPC…)
8. Events (rockbus) and scheduled jobs (rockcron)
9. Config field / new environment
10. One-shot command (migrations, CLI)
11. Tests

---

## 1. New project

```sh
bash "${CLAUDE_SKILL_DIR}/scripts/scaffold.sh" github.com/acme/orders ./orders
cd orders && go run ./cmd/main   # GET localhost:8080/api/v1/hello?name=Ann
```

The script copies `templates/skeleton/` (services catalog with the logger service, a `hello` handler, the log middleware, the HTTP App with `http.go`, Main, config, AGENTS.md, CLAUDE.md), sets the module path, runs `go get` for core + catalog modules and builds. If the target dir is not empty it refuses — then create files by hand from `templates/skeleton/`.

## 2. Service (PostgreSQL repository)

`go get github.com/arzab/gorock-catalog/rockbun@latest`

`pkg/services/orders/configs.go`
```go
package orders

import "github.com/arzab/gorock-catalog/rockbun"

type Configs struct {
	Postgres rockbun.Configs
}
```

`pkg/services/orders/interface.go`
```go
package orders

import (
	"context"
	"errors"

	"github.com/arzab/gorock-kit/rockservices"
)

var ErrNotFound = errors.New("orders: not found") // handlers check it with errors.Is

type Service interface {
	rockservices.Service // Init + Stop
	Insert(ctx context.Context, order any) error
	GetByID(ctx context.Context, order any, id string) error
}
```

`pkg/services/orders/service.go`
```go
package orders

import (
	"context"
	"database/sql"
	"errors"

	"github.com/arzab/gorock-catalog/rockbun"
)

type service struct{ db rockbun.Service } // child service: orders owns it

func NewService(cfg Configs) Service { return &service{db: rockbun.NewService(cfg.Postgres)} } // no connections here

func (s *service) Init(ctx context.Context) error { return s.db.Init(ctx) }
func (s *service) Stop() error                    { return s.db.Stop() }

func (s *service) Insert(ctx context.Context, order any) error {
	_, err := s.db.DB().NewInsert().Model(order).Exec(ctx)
	return err
}

func (s *service) GetByID(ctx context.Context, order any, id string) error {
	err := s.db.DB().NewSelect().Model(order).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
```

`pkg/services/services.go` — the key, the `Configs` field and the registration (after the logger):
```go
var Orders = rockservices.NewKey[orders.Service]("orders") // in the var block with the other keys

// in Configs:  Orders orders.Configs
// in Register: rockservices.Add(c, Orders, orders.NewService, cfg.Orders)
```
Use it anywhere: `services.Orders.Get().Insert(ctx, order)`. A key declared but not added fails `rockservices.Init` at startup.

Config: `services.orders.postgres: {host: ${DB_HOST:-localhost}, user: ${DB_USER:-postgres}, password: ${DB_PASSWORD}, database: ${DB_NAME:-app}}`.
Model in `internal/models/`: `bun.BaseModel \`bun:"table:orders" json:"-"\`` + `bun:"id,pk"` tags.

## 3. Service over another technology or logic

Same three files + registration. Examples:
- **Redis cache** (`rockredis`): `type service struct{ r rockredis.Service }`, `NewService(cfg) Service { return &service{r: rockredis.NewService(cfg.Redis)} }`, `Init`/`Stop` delegate; methods like `Get(ctx, key) (string, bool, error)` (map `rockredis.ErrNil` to `false`).
- **External API**: hold an `*http.Client` and base URL from `Configs`; `Init` may ping the API; methods are what the project needs (`Charge(ctx, amount) (id string, err error)`).
- **Logic** (file parsing, PDF): no connections — `Init` and `Stop` return nil.
- **Shared dependency** (one pool used by several services): register it first and pass the value `Add` returned:
  ```go
  pg := rockservices.Add(c, Postgres, postgres.NewService, cfg.Postgres)
  rockservices.Add(c, Orders, func(cfg orders.Configs) orders.Service { return orders.NewService(cfg, pg) }, cfg.Orders)
  ```
  Never call `key.Get()` inside a constructor: it reads `Default()`, not the catalog being filled.
- **Two instances** of one service: two keys (`NewKey[postgres.Service]("orders_db")`, `…("analytics_db")`) and two `Add` calls.

## 4. Handler + mount it on HTTP

`internal/handlers/orders/create.go`
```go
package orders

import (
	"fmt"
	"strings"

	"github.com/arzab/gorock-kit/rockhandlers"
	"github.com/google/uuid"

	"MODULE/internal/handlers/steps"
	"MODULE/internal/models"
	"MODULE/pkg/services"
)

type CreateParams struct {
	steps.BearerAuth        // the token, for the Bearer step (recipe 5)
	Item             string `json:"item"`
	Qty              int    `json:"qty"`
}

func (p *CreateParams) Validate() error { // runs before every step
	fields := map[string]string{}
	if strings.TrimSpace(p.Item) == "" {
		fields["item"] = "required"
	}
	if p.Qty <= 0 {
		fields["qty"] = "must be > 0"
	}
	if len(fields) > 0 {
		return rockhandlers.Invalid("invalid order").WithFields(fields)
	}
	return nil
}

type CreateResult struct {
	Order *models.Order `json:"order"`
}

type CreateCtx = rockhandlers.Ctx[*CreateParams, *CreateResult]

var Create = rockhandlers.Chain("orders.create", steps.Bearer, create)

func create(c *CreateCtx) error {
	order := &models.Order{ID: uuid.NewString(), Owner: steps.UserKey.MustGet(c), Item: c.Params.Item, Qty: c.Params.Qty}
	if err := services.Orders.Get().Insert(c, order); err != nil { // c is a context.Context
		return fmt.Errorf("insert order: %w", err) // → Internal, message hidden
	}
	c.Result.Order = order
	return nil
}
```

`cmd/apps/http.go` — add the endpoint:
```go
{Method: http.MethodPost, Path: "/orders", Handler: orders.Create, Status: http.StatusCreated},
```

- Params tags: `json` (body), `header`, `path` (pattern `{id}`), `query`, `form` (`*multipart.FileHeader` for uploads). Embedded structs are bound.
- Errors are kinds: `rockhandlers.Invalid` 400, `Unauthorized` 401, `Forbidden` 403, `NotFound` 404, `Conflict` 409, `Unavailable` 503; plain errors and panics → 500 with a hidden message. Map service errors where the decision is: `if errors.Is(err, orderssvc.ErrNotFound) { return rockhandlers.NotFound("order not found") }`.
- Multi-step: `rockhandlers.Chain("orders.update", steps.Bearer, load, checkOwner, update)`; a step that returns nil passes control on (`c.Next()` is optional — call it to run code after the rest); `return rockhandlers.Done` ends early with success (e.g. a cache hit) — never `return nil` for that; data between steps via `rockhandlers.NewLocal[T](name)`.
- No body: result `struct{}` → 204. Files `*rockhttp.File`, redirects `*rockhttp.Redirect`, cookies `rockhttp.SetCookie(c, …)`.
- `Status` is optional (default 200); the `/api/v1` prefix comes from `apps.http.prefix`.

## 5. Shared step (auth) and groups

`internal/handlers/steps/auth.go`
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
	user, ok := verify(c.Params.BearerToken())
	if !ok {
		return rockhandlers.Unauthorized("invalid token")
	}
	UserKey.Set(c, user)
	return c.Next()
}
```

Use: embed `steps.BearerAuth` in the Params, put `steps.Bearer` in the chain. Forgetting the embed is a compile error — never type-assert Params. Group = plain generic function:
```go
func Secured[P steps.TokenParams, R any](name string, s ...rockhandlers.Handler[P, R]) rockhandlers.Route {
	return rockhandlers.Chain(name, rockhandlers.Prepend(s, steps.Bearer[P, R])...)
}
```

## 6. Middleware

Only for logic that doesn't read Params. `internal/handlers/middleware/<name>.go`, signature `func(c rockhandlers.Context) error` that calls `c.Next()` (see the skeleton's `log.go`). App-wide: `rockhttp.New(cfg.HTTP, httpEndpoints()...).Use(middleware.Log, middleware.Metrics)`. One handler: `Create.With(rockhandlers.Timeout(2*time.Second))`. Decoding errors pass through middlewares too.

## 7. Custom App (queue consumer, gRPC…)

Only when the catalog has no App for the technology. `cmd/apps/<name>/configs.go` + `app.go`. The App writes only the protocol part — validation, panics and middlewares come from the handler chain.

```go
type Consumer struct {
	Queue   string
	Handler rockhandlers.Route // the same handlers HTTP uses
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
			return err // after ctx cancel this is a clean stop
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
		msg.Reject() // bad message: retrying won't help
	}
}

func (a *App) Stop() []error { // must not wait for Exec: it is also called when Exec never ran
	if a.conn == nil || a.started.Load() {
		return nil
	}
	return []error{a.conn.Close()}
}
```
Register in `apps.Register` before HTTP. Full checklist: https://arzab.github.io/gorock/gorock-kit/rockhandlers-transport

## 8. Events (rockbus) and scheduled jobs (rockcron)

These catalog Apps are not on rockhandlers yet: their handlers are `func(ctx) error`.
- **rockbus**: `internal/handlers/events/events.go` with `const OrderCreated rockbus.Topic = "order.created"` and `func Subscriptions() []rockbus.Subscription { return []rockbus.Subscription{rockbus.On(OrderCreated, onOrderCreated)} }`; in `apps.Register` before HTTP: `bus := rockbus.NewApp(cfg.Bus, onBusError, events.Subscriptions()...); rockbus.SetDefault(bus); e.MustRegister("bus", bus)`. Publish from a step: `rockbus.PublishAsync(c, rockbus.Event{Topic: events.OrderCreated, Payload: *order})`.
- **rockcron**: `internal/handlers/jobs/jobs.go` with `func All() []rockcron.Job { return []rockcron.Job{rockcron.Every("cleanup", time.Hour, cleanup)} }`; register `rockcron.NewApp(cfg.Cron, onJobError, jobs.All()...)` before HTTP. Config `apps.cron.location`.

## 9. Config field / new environment

1. Add the field to the right `Configs` (snake_case key is derived). Optional → `config:",omitempty"` or `config:",default=X"`; secret → `${VAR}` in the file; set in code only → `config:"-"`.
2. Add the key to `configs/configs.yaml` and every other `configs/*.yaml`.
3. Start the project once: rockconfig lists missing fields and unknown keys together.

New environment: copy `configs/configs.yaml` to `configs/prod.yaml`, secrets as `${VAR}`, run `./app ./configs/prod.yaml`.

## 10. One-shot command

The engine waits for a signal even after apps finish, so one-shot tasks don't use it. `cmd/migrate/main.go`:
```go
cfg, err := rockconfig.InitFromFile[Configs](path)
// ...
services.Register(rockservices.Default(), cfg.Services)
if err := rockservices.Init(ctx); err != nil {
	return err
}
defer rockservices.Stop()
return runMigrations(ctx, services.Orders.Get())
```

## 11. Tests

Handler without HTTP, with a fake service:
```go
type fakeOrders struct{ inserted []any }

func (f *fakeOrders) Init(context.Context) error                 { return nil }
func (f *fakeOrders) Stop() error                                { return nil }
func (f *fakeOrders) Insert(_ context.Context, o any) error      { f.inserted = append(f.inserted, o); return nil }
func (f *fakeOrders) GetByID(context.Context, any, string) error { return orderssvc.ErrNotFound }

func TestCreate(t *testing.T) {
	c := rockservices.New()
	rockservices.Add(c, services.Orders, func(struct{}) orderssvc.Service { return &fakeOrders{} }, struct{}{})
	prev := rockservices.SetDefault(c)
	t.Cleanup(func() { rockservices.SetDefault(prev) })

	res, err := rockhandlers.Run[*orders.CreateResult](context.Background(), orders.Create,
		&orders.CreateParams{BearerAuth: steps.BearerAuth{Authorization: "Bearer ann"}, Item: "tea", Qty: 2})
	if err != nil || res.Order.Owner != "ann" {
		t.Fatalf("res %+v, err %v", res, err)
	}
}
```
Tests that swap the default catalog must not use `t.Parallel()`. Full example: https://github.com/arzab/gorock/tree/main/examples/taskapi/internal/handlers/tasks
HTTP level: `app := rockhttp.New(cfg, httpEndpoints()...)`, `app.Init(ctx)`, then `httptest` against `app.Handler()`.
