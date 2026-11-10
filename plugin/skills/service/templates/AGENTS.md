# AGENTS.md

This is a Go project built with the **GOROCK** architecture: **gorock-kit** core (v0.5.x) + **gorock-catalog**, Go 1.26+.
Docs: https://arzab.github.io/gorock/ · Core: https://github.com/arzab/gorock-kit · Catalog: https://github.com/arzab/gorock-catalog

## Commands

```sh
go build ./... && go vet ./...          # must pass after every change
gofmt -l .                              # must print nothing
go test ./...
go run ./cmd/main                       # reads ./configs/configs.yaml
go run ./cmd/main ./configs/prod.yaml   # another config file
```

## The scheme

```
[client, broker, schedule] ⇄ App ⇄ Handlers ⇄ Services ⇄ [DB, APIs, files, logs]
          Main assembles everything · Config holds the project's settings
```

## Where code goes

```
pkg/services/             Services: wrappers over technologies or logic; services.go = typed keys, Configs, Register
internal/handlers/        Handlers: <domain>/<action>.go (Params, Result, steps, var X = rockhandlers.Chain(...)), steps/, middleware/
internal/models/          domain entities
cmd/apps/                 Apps: apps.go (which Apps run + their Configs), http.go (which handler on which method/path), own Apps
cmd/main/main.go          Main: config → services → apps → stop in reverse
pkg/libs/                 your own stateless reusable code
configs/configs.yaml      one file, mirrors cmd/main.Configs
```

Ask in this order: a tool for the business (DB, cache, external API, logger, file logic) → `pkg/services`. A business decision for a request/event/job → `internal/handlers`. How clients reach the project (HTTP, queue, cron, bot) or where a handler is mounted → `cmd/apps`. A domain entity → `internal/models`. A setting that differs between environments → Config.

Dependency direction: `cmd` → `internal` → `pkg`. `pkg/` never imports `internal/`. Handlers never import `cmd/`.

## Contracts

- **Service** (`rockservices.Service`): `Init(ctx) error`, `Stop() error` — no name of its own. Keyed in `pkg/services/services.go`: `var Orders = rockservices.NewKey[orders.Service]("orders")`, registered with `rockservices.Add(c, Orders, orders.NewService, cfg.Orders)` (add order = Init order, logger first), used as `services.Orders.Get()`. A declared but not added key fails startup. `cmd/main` opens the catalog before the Apps and closes it after.
- **Handler** (`rockhandlers`): `var Create = rockhandlers.Chain("orders.create", steps...)`, steps `func(c *rockhandlers.Ctx[*Params, *Result]) error`. A step that returns nil passes control on (calling `c.Next()` is optional — use it to run code after the rest); `return rockhandlers.Done` ends the chain early with success. Params arrive filled (body + headers/path/query by tags), `Params.Validate()` runs first, the Result is pre-allocated. Return a Result or an error kind.
- **App** (`rockengine.App`): `Init(ctx) error`, `Exec(ctx) error`, `Stop() []error`. No `Shutdown`: ctx cancellation = "finish current work". `Stop` must not wait for `Exec`. HTTP: `rockhttp.New(cfg, endpoints...).Use(mws...)` with `rockhttp.Endpoint{Method, Path, Handler, Status}` in `cmd/apps/http.go`.

## Rules

- **Services:** interface `Service` (embeds `rockservices.Service`), implementation `service`, constructor `func NewService(cfg Configs) Service` — stores config and creates child services, no connections. A service knows "how", not "when and why". Free form: a repository with domain-specific queries is fine. A child service lives inside its parent; a shared one is registered first and passed as the value `Add` returned (never `key.Get()` inside a constructor). No use-case layer.
- **Handlers:** never parse requests, write responses or return status codes. Errors: `rockhandlers.Invalid/Unauthorized/Forbidden/NotFound/Conflict/Unavailable(msg)` (`.WithFields` for validation); a plain `fmt.Errorf("...: %w", err)` is an unexpected error with a hidden message. Pass `c` to services. Data between steps: `rockhandlers.NewLocal[T](name)`.
- **Shared steps** (auth, tenant): generic with an interface constraint, Params satisfy it by embedding. No type assertions on Params. Groups are plain generic functions with `rockhandlers.Prepend`.
- **Middlewares** (`func(c rockhandlers.Context) error`): logging, metrics, timeouts — App-wide via `app.Use(...)`, per handler via `handler.With(...)`.
- **Apps:** register the HTTP App last (Apps stop in reverse order). Method, path and status only in `cmd/apps/`.
- **Logging:** `rocklog.Info/Warn/Error(msg, rocklog.Str(k, v), rocklog.Err(err))`, request-scoped `rocklog.FromContext(ctx)`. The logger service configures it. Never `log.Fatal` outside `main()`.
- **Tests:** `rockhandlers.Run[*Result](ctx, Handler, &Params{...})`; fake services with `rockservices.New()` + `rockservices.Add(c, services.Orders, …)` + `rockservices.SetDefault`. Such tests must not use `t.Parallel()`.

## Config (rockconfig)

- Every service and App declares `type Configs struct`; field names map to snake_case keys (`MaxBodyBytes` → `max_body_bytes`). No `json`/`yaml` tags.
- A field is **required** unless `config:",omitempty"`, `config:",default=X"` or `config:"-"`. Required = the key must be present.
- `${VAR}` (error if unset) or `${VAR:-default}`; secrets only via env. Unknown keys are errors. Durations with units: `5s`.
- After adding a field, add the key to every `configs/*.yaml`.

## Don't

- Don't call `os.Getenv` in code — use a `Configs` field.
- Don't open connections in constructors or start services by hand outside the catalog.
- Don't put HTTP details into handlers or business logic into Apps.
- Don't use removed API: string service names (`Add(c, "name", …)`, `Get[S](name)`), `return nil` as an early exit (use `rockhandlers.Done`), `Name()` on services, `RegisterResource`, `rockdelivery`, `internal/delivery`, `pkg/infra`, `github.com/arzab/gorock-kit/rockfiber` and other non-core kit paths, `config:"default:X"`, `$VAR` without braces.
