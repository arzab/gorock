# AGENTS.md

This is a Go service built with the **GOROCK** architecture: **gorock-kit** core (v0.3.x) + **gorock-catalog**, Go 1.26+.
Docs: https://arzab.github.io/gorock/ · Core: https://github.com/arzab/gorock-kit · Catalog: https://github.com/arzab/gorock-catalog

## Commands

```sh
go build ./... && go vet ./...          # must pass after every change
gofmt -l .                              # must print nothing
go test ./...
go run ./cmd/main                       # reads ./configs/configs.yaml
go run ./cmd/main ./configs/prod.yaml   # another config file
```

## Where code goes

```
cmd/main/main.go          Configs + Exec(): config → infra.Init → services (rockservices) → engine.Run (apps) → services stop
cmd/apps/apps.go          Configs of all apps + Register(engine, cfg): creates and registers every App
internal/delivery/        business logic: routes (rockhandlers chains), steps/, middleware/, endpoints/endpoints.go (HTTP mounting), events/, jobs/
internal/models/          domain entities
internal/utils/           stateless project helpers
pkg/services/             wrappers over external systems (DB, cache, APIs); added to the rockservices catalog
pkg/infra/                project primitives: logs (rocklog setup), vars
pkg/libs/                 your own stateless reusable code
configs/configs.yaml      one file, mirrors cmd/main.Configs
```

Ask in this order: wraps an external technology → `pkg/services`. Project tool (logger, tracer) → `pkg/infra`. What happens on a request/event/schedule → `internal/delivery`. Domain data → `internal/models`. Startup, transport, orchestration → `cmd/`.

Dependency direction: `cmd` → `internal` → `pkg`. `pkg/` never imports `internal/`. Delivery never imports `cmd/apps`.

## Contracts

- **App** (`rockengine.App`): `Init(ctx) error`, `Exec(ctx) error`, `Stop() []error`. No `Shutdown`: "finish current work" = cancellation of the ctx given to `Exec`. Anything `Exec` returns after ctx is cancelled is a clean stop. `Stop` must not wait for `Exec`. The engine manages apps only.
- **Service** (`rockservices.Service`): `Name() string`, `Init(ctx) error`, `Stop() error`. Added in `services.Register` with `rockservices.Add(c, NewService, cfg)` (add order = Init order); `cmd/main` calls `rockservices.Init` before the engine and `rockservices.Stop` after it. Get one anywhere with the package getter: `repository.Get()`.
- **Route** (`rockhandlers`): `rockhandlers.Chain("orders.create", steps...)`, steps `func(c *rockhandlers.Ctx[*Params, *Result]) error` calling `c.Next()`. Transport-independent: HTTP mounts it with `rockhttp.Endpoint{Method, Path, Handler, Status}` in `internal/delivery/endpoints/endpoints.go`.
- Catalog apps: HTTP `rockhttp.New(cfg, endpoints...).Use(mws...)`; events `rockbus.NewApp`, schedule `rockcron.NewApp`, Telegram `rocktelebot.NewApp` (not on rockhandlers yet). Catalog services `rockbun`, `rockredis` have no `Name()` — use them as child services inside your own.
- Write a custom App only when no catalog app fits; it only decodes into Params (`route.Serve(ctx, fill)`) and maps the result / `rockhandlers.KindOf(err)` to its protocol.

## Rules

- **Services:** interface `Service` (embeds `rockservices.Service`), implementation `service`, `const Name`, getter `func Get() Service`, constructor `func NewService(cfg Configs) Service` — stores config and creates child services, no connections. Services expose functionality (`Insert(ctx, model any)`), not business methods, and don't import `internal/models`. Return package errors (`ErrNotFound`) that delivery checks with `errors.Is`.
- **Routes:** one route = one folder (`route.go`, optionally `params.go`, `result.go`, `steps.go`). `Params` carries tags for the transport (`path`, `query`, `header`, `form`, `json`) and `Validate() error` (runs first). `Result` is pre-allocated — steps fill it. Pass `c` (a context.Context) to services. Data between steps: `rockhandlers.NewLocal[T](name)`.
- **Errors are business decisions:** `rockhandlers.Invalid/Unauthorized/Forbidden/NotFound/Conflict/Unavailable(msg)` (`.WithFields(map)` for validation); a plain `fmt.Errorf("...: %w", err)` becomes Internal with a hidden message. Never HTTP status codes in routes.
- **Reusable steps** (auth, tenant…): generic with an interface constraint (`func Bearer[P TokenParams, R any](c *rockhandlers.Ctx[P, R]) error`); models satisfy it by embedding. No type assertions on Params. Groups are plain generic functions using `rockhandlers.Prepend`.
- **Middlewares** (`func(c rockhandlers.Context) error`) only for logic that doesn't read request data: logging, metrics, timeouts. App-wide via `app.Use(...)`, per route via `route.With(...)`.
- **Logging:** `rocklog.Info/Warn/Error(msg, rocklog.Str(k, v), rocklog.Err(err))`; request-scoped: `rocklog.WithContext(ctx, fields...)`, `rocklog.FromContext(ctx)`. Never `rocklog.Fatal` or `log.Fatal` outside `main()`.
- **Registration order:** apps stop in reverse order — register the HTTP server last.
- **Tests:** `rockhandlers.Run[*Result](ctx, Route, &Params{...})` runs a whole route without HTTP; fake services with `rockservices.New()` + `rockservices.SetDefault`.

## Config (rockconfig)

- Every layer declares `type Configs struct`; field names map to snake_case keys automatically (`RequestTimeout` → `request_timeout`). No `json`/`yaml` tags.
- A field is **required** unless tagged `config:",omitempty"`, `config:",default=X"` or `config:"-"`. Required = the key must be present (`0`/`false` are fine; strings must be non-empty).
- `config:"-"` for values set in code (funcs, third-party structs). `config:",env=VAR"` binds a field to an env var.
- Env references in the file: `${VAR}` (error if unset) or `${VAR:-default}`. A bare `$` is literal. Secrets only via env.
- Unknown keys are errors (typos are caught). Durations are strings with units: `5s`, `1m30s` — never bare numbers.
- After adding a config field, add the key to `configs/configs.yaml` (and other environment files).

## Don't

- Don't call `os.Getenv` in business code — put the value in a `Configs` and the config file.
- Don't start services by hand outside `rockservices`, and don't register them in the engine (`RegisterResource` no longer exists).
- Don't put HTTP details (status codes, headers handling) into routes — tags in Params, status in `rockhttp.Endpoint`.
- Don't create goroutines that outlive a request without tying them to an App's `ctx`.
- Don't use the removed API: `rockdelivery`, `rockengine.Resource`, `github.com/arzab/gorock-kit/rockfiber` & other non-core kit paths, `config:"default:X"`, `$VAR` without braces, `pkg/libs/go-kit`.
