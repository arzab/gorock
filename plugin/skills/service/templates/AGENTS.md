# AGENTS.md

This is a Go service built with the **GOROCK** architecture and **gorock-kit** (v0.2.x, Go 1.26+).
Docs: https://arzab.github.io/gorock/ · Kit: https://github.com/arzab/gorock-kit

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
cmd/main/main.go          Configs (aggregates all layers) + Exec(): load config → infra.Init → engine → Run
cmd/apps/apps.go          Configs of all apps + Register(engine, cfg): creates and registers every App
internal/delivery/        business logic: endpoints/, events/, jobs/ — self-describing entities
internal/models/          domain entities
internal/utils/           stateless project helpers
pkg/services/             wrappers over external systems (DB, cache, APIs); singletons with getters
pkg/infra/                project primitives: logs (rocklog setup), vars
pkg/libs/                 your own stateless reusable code
configs/configs.yaml      one file, mirrors cmd/main.Configs
```

Ask in this order: wraps an external technology → `pkg/services`. Project tool (logger, tracer) → `pkg/infra`. What happens on a request/event/schedule → `internal/delivery`. Domain data → `internal/models`. Startup, transport, orchestration → `cmd/`.

Dependency direction: `cmd` → `internal` → `pkg`. `pkg/` never imports `internal/`. Delivery never imports `cmd/apps`.

## Contracts (gorock-kit)

- **App** (`rockengine.App`): `Init(ctx) error`, `Exec(ctx) error`, `Stop() []error`. No `Shutdown`: "finish current work" = cancellation of the ctx given to `Exec`. Anything `Exec` returns after ctx is cancelled is a clean stop. `Stop` must not wait for `Exec` (it can be called when `Exec` never ran).
- **Resource** (`rockengine.Resource`): `Init(ctx) error`, `Stop() error`. Every service in `pkg/services` is one; register with `e.MustRegisterResource(name, svc)` in `services.Register`. Resources init before all apps and stop after all apps.
- Ready-made apps: HTTP `rockfiber.New(cfg, endpoints...)`, events `rockbus.NewApp(cfg, onError, subs...)`, schedule `rockcron.NewApp(cfg, onError, jobs...)`, Telegram `rocktelebot.NewApp(cfg, middlewares, handlers...)`. Ready-made resources: `rockbun.NewService` (PostgreSQL/bun), `rockredis.NewService`.
- Write a custom App only when no ready-made one fits.

## Rules

- **Services:** interface is `Service`, implementation `service`, constructor `func NewService(cfg Configs) Service`. `NewService` only stores config and creates child services — no connections. Services expose functionality (`Insert(ctx, model any)`), not business methods (`SaveUser`), and don't import `internal/models`. Return package errors (`ErrNotFound`) that delivery checks with `errors.Is`.
- **Delivery (HTTP):** one endpoint = one folder with `endpoint.go` (`rockfiber.POST("/path", handlers()...)`), `handlers.go` (chain, each step calls `c.Next()`), `params.go` (struct + `Validate(*fiber.Ctx) error` + `rockfiber.DefaultHandler[Params]()`), `response.go`. Add it to `endpoints.All()`. The `/api/v1` prefix comes from config, not from the endpoint.
- **Errors are business decisions:** `rockfiber.NewError(code, msg)` for a status the client must see; a plain `fmt.Errorf("...: %w", err)` becomes 500 with a masked message. Pass `c.UserContext()` (deadline + trace id) to services.
- **Logging:** `rocklog.Info/Warn/Error(msg, rocklog.Str(k, v), rocklog.Err(err))`; request-scoped: `ctx = rocklog.WithContext(ctx, fields...)`, `rocklog.FromContext(ctx)`. Never `rocklog.Fatal` or `log.Fatal` outside `main()` — return errors; the engine decides about restarts.
- **Events:** topics and handlers in `internal/delivery/events` (`rockbus.On(topic, handler)`), publish with `rockbus.PublishAsync(ctx, rockbus.Event{Topic: t, Payload: v})`, read with `rockbus.Payload[T](ctx)`. Subscribe before the engine starts.
- **Jobs:** `rockcron.Every(name, interval, handlers...)` / `rockcron.Cron(name, "0 3 * * *", handlers...)` in `internal/delivery/jobs`; handlers respect `ctx`.
- **Registration order:** apps stop in reverse order — register the HTTP server last.

## Config (rockconfig)

- Every layer declares `type Configs struct`; field names map to snake_case keys automatically (`RequestTimeout` → `request_timeout`). No `json`/`yaml` tags.
- A field is **required** unless tagged `config:",omitempty"`, `config:",default=X"` or `config:"-"`. Required = the key must be present (`0`/`false` are fine; strings must be non-empty).
- `config:"-"` for values set in code (funcs, third-party structs). `config:",env=VAR"` binds a field to an env var.
- Env references in the file: `${VAR}` (error if unset) or `${VAR:-default}`. A bare `$` is literal. Secrets only via env.
- Unknown keys are errors (typos are caught). Durations are strings with units: `5s`, `1m30s` — never bare numbers.
- A field with an in-code default must be `omitempty`, otherwise users are forced to set it.
- After adding a config field, add the key to `configs/configs.yaml` (and other environment files).

## Don't

- Don't call `os.Getenv` in business code — put the value in a `Configs` and the config file.
- Don't add `Init`/`Stop` orchestration by hand in `cmd/main` — register with the engine.
- Don't create goroutines that outlive a request without tying them to an App's `ctx`.
- Don't use `$VAR` (without braces) in config files, `config:"default:X"`, `BuildFiberEndpoint`, `pkg/libs/go-kit` — that is the old, removed API.
