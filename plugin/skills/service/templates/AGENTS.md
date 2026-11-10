# AGENTS.md

This is a Go project built with the **GOROCK** architecture: **gorock-kit** core (v0.6.x) + **gorock-catalog**, Go 1.26+.
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
pkg/services/             Services: wrappers over technologies or logic; services.go = catalog (names, Configs, Init, Stop, getters)
internal/handlers/        Handlers: <domain>/<action>.go (Params, Response, handlers, var X = rockhandlers.Chain(...)), steps/, middleware/
internal/models/          domain entities
cmd/apps/                 Apps: apps.go (Configs + Run: which Apps run, in which order), http.go (which chain on which method/path), own Apps
cmd/main/main.go          Main: config → services.Init → apps.Run → services.Stop
pkg/libs/                 your own stateless reusable code
configs/configs.yaml      one file, mirrors cmd/main.Configs
```

Ask in this order: a tool for the business (DB, cache, external API, logger, file logic) → `pkg/services`. A business decision for a request/event/job → `internal/handlers`. How clients reach the project (HTTP, queue, cron, bot) or where a chain is mounted → `cmd/apps`. A domain entity → `internal/models`. A setting that differs between environments → Config.

Dependency direction: `cmd` → `internal` → `pkg`. `pkg/` never imports `internal/`. Handlers never import `cmd/`.

## Contracts

- **Service** (`rockservices.Service`): `Init(ctx) error`, `Stop() error` — no name of its own. `pkg/services/services.go` holds the catalog: name constants (`RepositoryName = "repository"`), `Configs`, `Init(ctx, cfg)` that builds `rockservices.New(map[string]rockservices.Service{...})` and calls `catalog.Init(ctx, LoggerName, RepositoryName)` (this order, logger first), `Stop()` (reverse order) and getters `func Repository() repository.Service { return catalog.Get(RepositoryName).(repository.Service) }`. `cmd/main` starts the services before the Apps and stops them after.
- **Handler** (`rockhandlers`): `func create(c *rockhandlers.Ctx, params *CreateParams, response *CreateResponse) error`, chained with `var Create = rockhandlers.Chain(steps.Bearer, create)` (types inferred; may be empty). `return c.Next()` runs the next handler, `return nil` ends the chain with success, `return err` stops it; `Next` past the end → `rockhandlers.ErrHandlerNotFound`. Params arrive filled (body + headers/path/query by tags) and have `Validate() error`, run first; Response has `Normalize() error`, run after a successful chain.
- **App** (`rockengine.App`): `Init(ctx) error`, `Exec(ctx) error`, `Stop() []error`. No `Shutdown`: ctx cancellation = "finish current work". `Stop` must not wait for `Exec`. `cmd/apps/apps.go`: `rockengine.New(map[string]rockengine.App{...}).Run(ctx, "bus", "http")`. HTTP: `rockhttp.New(cfg, endpoints...).Use(mws...)` with `rockhttp.Endpoint{Method, Path, Handler, Status}` in `cmd/apps/http.go`.

## Rules

- **Services:** interface `Service` (embeds `rockservices.Service`), implementation `service`, constructor `func NewService(cfg Configs) Service` — stores config and creates child services, no connections. A service knows "how", not "when and why". Free form: a repository with domain-specific queries is fine. A child service lives inside its parent; a shared one is created first in `services.Init`, put into the map and passed to the dependent constructor. No use-case layer.
- **Handlers:** never parse requests, write responses or return status codes. Errors: `rockhandlers.Invalid/Unauthorized/Forbidden/NotFound/Conflict/Unavailable(msg)` (`.WithFields` for validation); a plain `fmt.Errorf("...: %w", err)` is an unexpected error with a hidden message. Pass `c` to services. Data between handlers: `c.Set(key, v)` / `c.Get(key)`.
- **Shared steps** (auth, tenant): generic `func Bearer[P TokenParams, R rockhandlers.Response](c *rockhandlers.Ctx, params P, _ R) error` with `TokenParams interface{ rockhandlers.Params; BearerToken() string }`; Params satisfy it by embedding. No type assertions on Params.
- **Middlewares** (`rockhandlers.Middleware` = `func(c *rockhandlers.Ctx, _ rockhandlers.Params, _ rockhandlers.Response) error` that calls `c.Next()`): logging, metrics, timeouts. They run before the params are filled — never read them. App-wide via `app.Use(...)`, per chain via `Create.Use(...)`.
- **Apps:** list the HTTP App last in `Run` (Apps stop in reverse order). Method, path and status only in `cmd/apps/`.
- **Logging:** `rocklog.Info/Warn/Error(msg, rocklog.Str(k, v), rocklog.Err(err))`, request-scoped `rocklog.FromContext(ctx)`. The logger service configures it. Never `log.Fatal` outside `main()`.
- **Tests:** `res, err := tasks.Create.Call(ctx, &tasks.CreateParams{...})` then `res.(*tasks.CreateResponse)`; fake services via `services.UseCatalog(rockservices.New(map[string]rockservices.Service{services.RepositoryName: fake}))` (returns restore). Such tests must not use `t.Parallel()`.

## Config (rockconfig)

- Every service and App declares `type Configs struct`; field names map to snake_case keys (`MaxBodyBytes` → `max_body_bytes`). No `json`/`yaml` tags.
- A field is **required** unless `config:",omitempty"`, `config:",default=X"` or `config:"-"`. Required = the key must be present.
- `${VAR}` (error if unset) or `${VAR:-default}`; secrets only via env. Unknown keys are errors. Durations with units: `5s`.
- After adding a field, add the key to every `configs/*.yaml`.

## Don't

- Don't call `os.Getenv` in code — use a `Configs` field.
- Don't open connections in constructors or start services by hand outside `services.Init`.
- Don't put HTTP details into handlers or business logic into Apps.
- Don't use removed API: `rockservices.NewKey`/`Add`/`SetDefault`/`Default`, `key.Get()`, `rockhandlers.Ctx[P, R]`, `c.Params`/`c.Result`, `rockhandlers.Done`/`Run`/`NewLocal`/`Prepend`/`Route`, chain names, `.With(...)`, `rockengine.Register`/`Default`/`RestartPolicy`, `apps.Register`, `Name()` on services, `RegisterResource`, `rockdelivery`, `internal/delivery`, `pkg/infra`, `github.com/arzab/gorock-kit/rockfiber` and other non-core kit paths, `config:"default:X"`, `$VAR` without braces.
