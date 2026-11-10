# Recipes

Each recipe lists every file to touch. `MODULE` is the module path from `go.mod`. After any recipe: `gofmt -w`, `go build ./... && go vet ./...`, `go test ./...`, and start the project once if config changed.

Layers (bottom-up): **Services** (`pkg/services/`) → **Handlers** (`internal/handlers/`) → **App** (`cmd/apps/`) → **Main** (`cmd/main/`) → **Config** (`configs/`).
Core: `github.com/arzab/gorock-kit/<module>` (rockservices, rockhandlers, rockengine, rockconfig). Catalog: `github.com/arzab/gorock-catalog/<module>` (rockhttp, rocklog, rockbun, rockredis, rockbus, rockcron, rocktelebot, rockfiber).
Reference project with all of this working: https://github.com/arzab/gorock/tree/main/examples/taskapi

## Contents
1. New project
2. Service (PostgreSQL repository)
3. Service over another technology (Redis, external API) or logic
4. Handler + mount it on HTTP
5. Shared step (auth)
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

The script copies `templates/skeleton/` (services catalog with the logger service, a `hello` chain, the log middleware, `cmd/apps` with `Run` and `http.go`, Main, config, AGENTS.md, CLAUDE.md), sets the module path, runs `go get` for core + catalog modules and builds. If the target dir is not empty it refuses — then create files by hand from `templates/skeleton/`.

## 2. Service (PostgreSQL repository)

`go get github.com/arzab/gorock-catalog/rockbun@v0.6.0`

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

`pkg/services/services.go` — four places: the name, the `Configs` field, the map entry + start order, the getter:
```go
const (
	LoggerName = "logger"
	OrdersName = "orders"
)

type Configs struct {
	Logger logger.Configs `config:",omitempty"`
	Orders orders.Configs
}

func Init(ctx context.Context, cfg Configs) error {
	catalog = rockservices.New(map[string]rockservices.Service{
		LoggerName: logger.NewService(cfg.Logger),
		OrdersName: orders.NewService(cfg.Orders),
	})
	return catalog.Init(ctx, LoggerName, OrdersName) // logger first
}

func Orders() orders.Service { return catalog.Get(OrdersName).(orders.Service) }
```
Use it anywhere: `services.Orders().Insert(ctx, order)`. A name passed to `catalog.Init` but missing from the map panics at startup; a service in the map but not in the order is never started.

Config: `services.orders.postgres: {host: ${DB_HOST:-localhost}, user: ${DB_USER:-postgres}, password: ${DB_PASSWORD}, database: ${DB_NAME:-app}}`.
Model in `internal/models/`: `bun.BaseModel \`bun:"table:orders" json:"-"\`` + `bun:"id,pk"` tags.

## 3. Service over another technology or logic

Same three files + the four places in `services.go`. Examples:
- **Redis cache** (`rockredis`): `type service struct{ r rockredis.Service }`, `NewService(cfg) Service { return &service{r: rockredis.NewService(cfg.Redis)} }`, `Init`/`Stop` delegate; methods like `Get(ctx, key) (string, bool, error)` (map `rockredis.ErrNil` to `false`).
- **External API**: hold an `*http.Client` and base URL from `Configs`; `Init` may ping the API; methods are what the project needs (`Charge(ctx, amount) (id string, err error)`).
- **Logic** (file parsing, PDF): no connections — `Init` and `Stop` return nil.
- **Shared dependency** (one pool used by several services): create it first, put it into the map and pass it to the dependent constructors; the catalog starts and stops it, the dependents don't:
  ```go
  pg := postgres.NewService(cfg.Postgres)
  catalog = rockservices.New(map[string]rockservices.Service{
  	LoggerName:   logger.NewService(cfg.Logger),
  	PostgresName: pg,
  	OrdersName:   orders.NewService(cfg.Orders, pg), // orders.Init/Stop don't touch pg
  })
  return catalog.Init(ctx, LoggerName, PostgresName, OrdersName) // pg before its users
  ```
- **Two instances** of one service: two names (`OrdersDBName`, `AnalyticsDBName`), two map entries, two getters.

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

func (p *CreateParams) Validate() error { // runs before every handler
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

type CreateResponse struct {
	Order *models.Order `json:"order"`
}

func (r *CreateResponse) Normalize() error { return nil } // runs after a successful chain

var Create = rockhandlers.Chain(steps.Bearer, create) // types inferred from create

func create(c *rockhandlers.Ctx, params *CreateParams, response *CreateResponse) error {
	order := &models.Order{ID: uuid.NewString(), Owner: steps.User(c), Item: params.Item, Qty: params.Qty}
	if err := services.Orders().Insert(c, order); err != nil { // c is a context.Context
		return fmt.Errorf("insert order: %w", err) // → Internal, message hidden
	}
	response.Order = order
	return nil // done
}
```

`cmd/apps/http.go` — add the endpoint:
```go
{Method: http.MethodPost, Path: "/orders", Handler: orders.Create, Status: http.StatusCreated},
```

- Params tags: `json` (body), `header`, `path` (pattern `{id}`), `query`, `form` (`*multipart.FileHeader` for uploads). Embedded structs are bound.
- Errors are kinds: `rockhandlers.Invalid` 400, `Unauthorized` 401, `Forbidden` 403, `NotFound` 404, `Conflict` 409, `Unavailable` 503; plain errors and panics → 500 with a hidden message. Map service errors where the decision is: `if errors.Is(err, orderssvc.ErrNotFound) { return rockhandlers.NotFound("order not found") }`.
- Multi-step: `rockhandlers.Chain(steps.Bearer, load, checkOwner, update)`. `return c.Next()` passes control on (code after it runs after the rest of the chain); `return nil` ends the chain early with success (e.g. a cache hit); data between handlers via `c.Set(key, v)` / `c.Get(key)` (use an unexported key type).
- Skeleton: `var Update = rockhandlers.Chain[*UpdateParams, *UpdateResponse]()` compiles and answers 501 (`ErrHandlerNotFound`) until handlers are added. The same happens when the last handler returns `c.Next()`.
- No body: an empty response struct → 204. Files `*rockhttp.File`, redirects `*rockhttp.Redirect` (as the response type), cookies `rockhttp.SetCookie(c, …)`.
- `Status` is optional (default 200); the `/api/v1` prefix comes from `apps.http.prefix`.

## 5. Shared step (auth)

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

type TokenParams interface {
	rockhandlers.Params
	BearerToken() string
}

type userKey struct{}

// Bearer fits any chain whose Params embed BearerAuth.
func Bearer[P TokenParams, R rockhandlers.Response](c *rockhandlers.Ctx, params P, _ R) error {
	user, ok := verify(params.BearerToken())
	if !ok {
		return rockhandlers.Unauthorized("invalid token")
	}
	c.Set(userKey{}, user)
	return c.Next()
}

// User returns the user Bearer stored.
func User(c *rockhandlers.Ctx) string {
	u, _ := c.Get(userKey{})
	s, _ := u.(string)
	return s
}
```

Use: embed `steps.BearerAuth` in the Params, put `steps.Bearer` first in the chain — the type parameters are inferred. Forgetting the embed is a compile error — never type-assert Params. A "group" is just a helper that returns the handlers to put in front.

## 6. Middleware

Only for logic that doesn't read Params: middlewares run before the params are filled. `internal/handlers/middleware/<name>.go`, a `rockhandlers.Middleware`: `func(c *rockhandlers.Ctx, _ rockhandlers.Params, _ rockhandlers.Response) error` that calls `c.Next()` (see the skeleton's `log.go`). App-wide: `rockhttp.New(cfg.HTTP, httpEndpoints()...).Use(middleware.Log, middleware.Metrics)`. One chain: `var Create = rockhandlers.Chain(create).Use(rockhandlers.Timeout(2*time.Second))` — each `Use` goes in front of the previous ones. Decoding and validation errors pass through middlewares too.

## 7. Custom App (queue consumer, gRPC…)

Only when the catalog has no App for the technology. `cmd/apps/<name>/configs.go` + `app.go`. The App writes only the protocol part — validation, panics and middlewares come from the chain.

```go
type Consumer struct {
	Queue   string
	Handler *rockhandlers.Pipeline // the same chains HTTP uses
}

type App struct {
	cfg       Configs
	consumers []Consumer
	mws       []rockhandlers.Middleware
	pipelines map[string]*rockhandlers.Pipeline // queue → chain with the App's middlewares
	conn      *queue.Conn
	started   atomic.Bool
}

func NewApp(cfg Configs, consumers ...Consumer) *App { return &App{cfg: cfg, consumers: consumers} }

func (a *App) Use(mws ...rockhandlers.Middleware) *App { a.mws = append(a.mws, mws...); return a }

func (a *App) Init(ctx context.Context) error {
	a.pipelines = map[string]*rockhandlers.Pipeline{}
	for _, c := range a.consumers {
		a.pipelines[c.Queue] = c.Handler.Use(a.mws...) // Use returns a copy: safe on restarts
	}
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
	_, err := a.pipelines[msg.Queue].Run(ctx, func(params rockhandlers.Params) error {
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
Add it to the map in `cmd/apps/apps.go` and to the `Run` order before `"http"`. Full checklist: https://arzab.github.io/gorock/gorock-kit/rockhandlers-transport

## 8. Events (rockbus) and scheduled jobs (rockcron)

These catalog Apps are not on rockhandlers yet: their handlers are `func(ctx) error`.
- **rockbus**: `internal/handlers/events/events.go` with `const OrderCreated rockbus.Topic = "order.created"` and `func Subscriptions() []rockbus.Subscription { return []rockbus.Subscription{rockbus.On(OrderCreated, onOrderCreated)} }`. In `apps.Run`: `bus := rockbus.NewApp(cfg.Bus, onBusError, events.Subscriptions()...); rockbus.SetDefault(bus)`, map entry `"bus": bus`, order `engine.Run(ctx, "bus", "http")`. Publish from a handler: `rockbus.PublishAsync(c, rockbus.Event{Topic: events.OrderCreated, Payload: *order})`.
- **rockcron**: `internal/handlers/jobs/jobs.go` with `func All() []rockcron.Job { return []rockcron.Job{rockcron.Every("cleanup", time.Hour, cleanup)} }`; map entry `"cron": rockcron.NewApp(cfg.Cron, onJobError, jobs.All()...)`, before `"http"` in the order. Config `apps.cron.location`.

## 9. Config field / new environment

1. Add the field to the right `Configs` (snake_case key is derived). Optional → `config:",omitempty"` or `config:",default=X"`; secret → `${VAR}` in the file; set in code only → `config:"-"`.
2. Add the key to `configs/configs.yaml` and every other `configs/*.yaml`.
3. Start the project once: rockconfig lists missing fields and unknown keys together.

New environment: copy `configs/configs.yaml` to `configs/prod.yaml`, secrets as `${VAR}`, run `./app ./configs/prod.yaml`.

## 10. One-shot command

Migrations, imports, CLI commands finish on their own — no Apps, no engine. `cmd/migrate/main.go`:
```go
cfg, err := rockconfig.InitFromFile[Configs](path)
// ...
if err := services.Init(ctx, cfg.Services); err != nil {
	return err
}
defer services.Stop()
return runMigrations(ctx, services.Orders())
```

## 11. Tests

Chain without HTTP, with a fake service (`services.UseCatalog` is in the skeleton):
```go
type fakeOrders struct{ inserted []any }

func (f *fakeOrders) Init(context.Context) error                 { return nil }
func (f *fakeOrders) Stop() error                                { return nil }
func (f *fakeOrders) Insert(_ context.Context, o any) error      { f.inserted = append(f.inserted, o); return nil }
func (f *fakeOrders) GetByID(context.Context, any, string) error { return orderssvc.ErrNotFound }

func TestCreate(t *testing.T) {
	fake := &fakeOrders{}
	t.Cleanup(services.UseCatalog(rockservices.New(map[string]rockservices.Service{
		services.OrdersName: fake,
	})))

	res, err := orders.Create.Call(context.Background(),
		&orders.CreateParams{BearerAuth: steps.BearerAuth{Authorization: "Bearer ann"}, Item: "tea", Qty: 2})
	if err != nil {
		t.Fatal(err)
	}
	if order := res.(*orders.CreateResponse).Order; order.Owner != "ann" || len(fake.inserted) != 1 {
		t.Fatalf("order %+v, inserted %v", order, fake.inserted)
	}
}
```
`Call` runs `Validate`, the handlers and `Normalize`, like a real request. Tests that swap the catalog must not use `t.Parallel()`. Full example: https://github.com/arzab/gorock/tree/main/examples/taskapi/internal/handlers/tasks
HTTP level: `app := rockhttp.New(cfg, httpEndpoints()...)`, `app.Init(ctx)`, then `httptest` against `app.Handler()`.
