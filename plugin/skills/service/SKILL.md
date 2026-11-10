---
name: service
description: Build or extend a Go project with the GOROCK architecture — gorock-kit core (rockservices, rockhandlers, rockengine, rockconfig) and gorock-catalog (rockhttp, rocklog, rockbun, rockredis, rockbus, rockcron, rocktelebot, rockfiber). Use when creating a new Go backend with gorock, or when the project's go.mod requires github.com/arzab/gorock-kit or github.com/arzab/gorock-catalog, or it has the GOROCK layout (pkg/services, internal/handlers, cmd/apps, cmd/main) — e.g. adding a service (PostgreSQL, Redis, external API, logic), a handler and its HTTP endpoint, a shared step or middleware, a custom App, events, scheduled jobs, or a config field.
---

# GOROCK project

A GOROCK project is one scheme: `[client] ⇄ App ⇄ Handlers ⇄ Services ⇄ [DB, APIs, files, logs]`, assembled by Main and configured by Config. gorock-kit is the core every project needs; gorock-catalog holds ready Apps and services for specific technologies. Your job is to put each piece of code in the right layer and use the contracts exactly.

## Layers (bottom-up)

- **Services** (`pkg/services/`): wrappers over a technology or logic the business uses (DB, cache, external API, logger, file parsing). `Init`/`Stop`, no business decisions ("how", not "when and why"). `pkg/services/services.go` owns the catalog: name constants, `Configs`, `Init(ctx, cfg)` (map of services + start order), `Stop()` and typed getters. No use-case layer on top.
- **Handlers** (`internal/handlers/`): business logic as Fiber-style rockhandlers chains: `func(c *rockhandlers.Ctx, params *P, response *R) error`. Know nothing about HTTP/queues. One action = one file (`internal/handlers/<domain>/<action>.go`); shared steps in `steps/`, middlewares in `middleware/`.
- **App** (`cmd/apps/`): the technology clients reach the project through (HTTP, queue, cron, bot). Fills Params, runs the chain, maps the response/kind to its protocol, stops gracefully. Which chain hangs where (method+path, queue) is in `cmd/apps/` (e.g. `http.go`); `apps.go` has `Run(ctx, cfg)` with the engine.
- **Main** (`cmd/main/`): the same assembly in every project: config → `services.Init` → `defer services.Stop()` → `apps.Run`. rockengine runs the Apps only.
- **Config** (`configs/`): the project's variables, constants and env references in one strict file mirroring `cmd/main.Configs`.

## Workflow

1. **Orient.** Read `go.mod` and `AGENTS.md`. Kit `v0.6.x` = current model: `rockservices.New(map)` + `Init(ctx, names...)`, getters `catalog.Get(name).(T)`; handlers `func(c, params, response) error` with `return c.Next()` / `return nil`; `rockengine.New(map).Run(ctx, names...)`. `v0.5.x` = typed keys (`rockservices.NewKey`, `Add`), `Ctx[P,R]` with `c.Params`, `rockhandlers.Done`, `apps.Register`. `v0.4.x` and older = string names, `Name()` on services, resources in the engine — keep the project's style unless asked to migrate (migration table: gorock-kit `CHANGELOG.md`).
   - No project yet → scaffold (recipe 1). Don't hand-write the skeleton.
   - Existing project without `AGENTS.md` → copy `${CLAUDE_SKILL_DIR}/templates/AGENTS.md` to the root and offer `CLAUDE.md` containing `@AGENTS.md`.
2. **Pick the recipe** in [references/recipes.md](references/recipes.md) and follow it file by file. Exact signatures: [references/kit-api.md](references/kit-api.md) — don't guess APIs.
3. **Wire it up.** Every new piece is reachable from exactly one place: service → name constant + map entry + place in the `catalog.Init` order + getter in `services.go`; handler → `cmd/apps/http.go` (or another App's binding); App → the map and the `Run` order in `cmd/apps/apps.go`. Config field → layer `Configs` **and** `configs/*.yaml`.
4. **Verify.** `gofmt -l .` prints nothing; `go build ./... && go vet ./...`; `go test ./...` (handlers via `Chain(...).Call`). If config or wiring changed, run `go run ./cmd/main` and confirm it gets past config loading and service Init.
5. **Report** what was added and where, and what the user must provide (env vars, DB schema, tokens).

## Rules that are easy to get wrong

- Services: `Service` = `Init(ctx) error` + `Stop() error` — no name inside. In `services.go`: `RepositoryName = "repository"`, map entry `RepositoryName: repository.NewService(cfg.Repository)`, order `catalog.Init(ctx, LoggerName, RepositoryName)` (logger first), getter `func Repository() repository.Service { return catalog.Get(RepositoryName).(repository.Service) }`. A shared dependency: create it first, put it into the map and pass it to the dependent constructor. Constructors open no connections.
- Handlers: `return c.Next()` runs the next one; `return nil` ends the chain with success; `return err` stops it. `Next` past the end → `rockhandlers.ErrHandlerNotFound` (HTTP 501). An empty `Chain()` is a valid skeleton.
- Params must have `Validate() error` (runs before the handlers; a plain error → Invalid with its text); Response must have `Normalize() error` (runs after a successful chain). Both are pointer types.
- Handlers never return HTTP codes or write responses; errors are `rockhandlers.Invalid/Unauthorized/Forbidden/NotFound/Conflict/Unavailable(...)`; plain errors become hidden 500s. Pass `c` (a context.Context) to services.
- Shared steps that read Params are generic: `func Bearer[P TokenParams, R rockhandlers.Response](c *rockhandlers.Ctx, params P, _ R) error` with `TokenParams interface{ rockhandlers.Params; BearerToken() string }`; Params satisfy it by embedding. Never type-assert Params. Middlewares (`rockhandlers.Middleware`, a handler with the general `Params`/`Response`) run before the params are filled — only for logic that doesn't read them. Data between handlers: `c.Set`/`c.Get`.
- Method, path and status live in `cmd/apps/http.go`, never in a handler. `Status` is optional (default 200, 204 for an empty response struct).
- App = `Init(ctx) / Exec(ctx) / Stop() []error`; no `Shutdown`; `Stop` must not wait for `Exec`. In `engine.Run(ctx, names...)` list HTTP last (Apps stop in reverse).
- Config: no `json` tags; required unless `omitempty` / `default=` / `-`; `${VAR}` or `${VAR:-x}`; durations as `5s`; unknown keys are errors. No `os.Getenv` in code.
- Log with `rocklog` (configured by the logger service, started first); never `Fatal`/`log.Fatal` outside `main()`.
- `pkg/` never imports `internal/`; `internal/handlers` never imports `cmd/`.
- Removed/old API that must not appear in v0.6 projects: `rockservices.NewKey`, `rockservices.Add`, `Register`, `SetDefault`, `Default()`, `key.Get()`, package-level `rockservices.Init/Stop`; `rockhandlers.Ctx[P, R]`, `c.Params`, `c.Result`, `rockhandlers.Done`, `rockhandlers.Run`, `NewLocal`, `Prepend`, `Route`, `Serve`, `.With(...)`, chain names in `Chain("name", …)`; `rockengine.Register/MustRegister`, `rockengine.Default`, `StopApp`, `RestartPolicy`, `apps.Register`; string-name lookups `Get[S](name)`, `Name()` on services, `RegisterResource`, `rockdelivery`, `internal/delivery`, `pkg/infra`, `github.com/arzab/gorock-kit/rockfiber` (and other non-core kit paths), `configs.InitFromFile`, `config:"default:X"`, `pkg/libs/go-kit`, `logrus`.

## Resources

- [references/recipes.md](references/recipes.md) — step-by-step code for every common task
- [references/kit-api.md](references/kit-api.md) — compact API and config keys of the core and catalog
- `${CLAUDE_SKILL_DIR}/templates/AGENTS.md` — project rules for any coding agent
- `${CLAUDE_SKILL_DIR}/templates/skeleton/` — minimal working project (used by `scripts/scaffold.sh`)
- Docs: https://arzab.github.io/gorock/ (architecture, guides per layer) · Example: https://github.com/arzab/gorock/tree/main/examples/taskapi · Core: https://github.com/arzab/gorock-kit · Catalog: https://github.com/arzab/gorock-catalog
