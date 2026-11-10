---
name: service
description: Build or extend a Go project with the GOROCK architecture — gorock-kit core (rockservices, rockhandlers, rockengine, rockconfig) and gorock-catalog (rockhttp, rocklog, rockbun, rockredis, rockbus, rockcron, rocktelebot, rockfiber). Use when creating a new Go backend with gorock, or when the project's go.mod requires github.com/arzab/gorock-kit or github.com/arzab/gorock-catalog, or it has the GOROCK layout (pkg/services, internal/handlers, cmd/apps, cmd/main) — e.g. adding a service (PostgreSQL, Redis, external API, logic), a handler and its HTTP endpoint, a shared step or middleware, a custom App, events, scheduled jobs, or a config field.
---

# GOROCK project

A GOROCK project is one scheme: `[client] ⇄ App ⇄ Handlers ⇄ Services ⇄ [DB, APIs, files, logs]`, assembled by Main and configured by Config. gorock-kit is the core every project needs; gorock-catalog holds ready Apps and services for specific technologies. Your job is to put each piece of code in the right layer and use the contracts exactly.

## Layers (bottom-up)

- **Services** (`pkg/services/`): wrappers over a technology or logic the business uses (DB, cache, external API, logger, file parsing). `Init`/`Stop`, no business decisions ("how", not "when and why"). Registered in one catalog under names given in `pkg/services/services.go`, which also holds the getters. No use-case layer on top.
- **Handlers** (`internal/handlers/`): business logic as rockhandlers chains. Get filled Params (body + request metadata), return a Result or a typed error kind. Know nothing about HTTP/queues. One action = one file (`internal/handlers/<domain>/<action>.go`); shared steps in `steps/`, middlewares in `middleware/`.
- **App** (`cmd/apps/`): the technology clients reach the project through (HTTP, queue, cron, bot). Fills Params, calls handlers, maps the result/kind to its protocol, stops gracefully. Which handler hangs where (method+path, queue) is in `cmd/apps/` (e.g. `http.go`).
- **Main** (`cmd/main/`): the same assembly in every project: config → services → apps → stop in reverse. rockengine runs the Apps only.
- **Config** (`configs/`): the project's variables, constants and env references in one strict file mirroring `cmd/main.Configs`.

## Workflow

1. **Orient.** Read `go.mod` and `AGENTS.md`. Kit `v0.4.x` = current model (services named at registration). `v0.3.x` = services have `Name()`. `v0.2.x` = old model (resources in the engine, rockfiber endpoints, `internal/delivery`, `pkg/infra`) — keep its style unless asked to migrate.
   - No project yet → scaffold (recipe 1). Don't hand-write the skeleton.
   - Existing project without `AGENTS.md` → copy `${CLAUDE_SKILL_DIR}/templates/AGENTS.md` to the root and offer `CLAUDE.md` containing `@AGENTS.md`.
2. **Pick the recipe** in [references/recipes.md](references/recipes.md) and follow it file by file. Exact signatures: [references/kit-api.md](references/kit-api.md) — don't guess APIs.
3. **Wire it up.** Every new piece is reachable from exactly one place: service → `services.Register` + name constant + getter; handler → `cmd/apps/http.go` (or another App's binding); App → `apps.Register`. Config field → layer `Configs` **and** `configs/*.yaml`.
4. **Verify.** `gofmt -l .` prints nothing; `go build ./... && go vet ./...`; `go test ./...` (handlers via `rockhandlers.Run`). If config or wiring changed, run `go run ./cmd/main` and confirm it gets past config loading and service Init.
5. **Report** what was added and where, and what the user must provide (env vars, DB schema, tokens).

## Rules that are easy to get wrong

- Services: `Service` = `Init(ctx) error` + `Stop() error` — **no `Name()`**; `rockservices.Add(c, Name, NewService, cfg)`; constructors open no connections; getters live in `services.go`.
- Handlers never return HTTP codes or write responses; errors are `rockhandlers.Invalid/Unauthorized/Forbidden/NotFound/Conflict/Unavailable(...)`; plain errors become hidden 500s. Pass `c` (a context.Context) to services. `Validate()` on Params runs first.
- Shared steps that need request data are generic with an interface constraint; Params satisfy it by embedding (`steps.BearerAuth`). Never type-assert Params. Middlewares only for logic that doesn't read Params.
- Method, path and status live in `cmd/apps/http.go`, never in a handler. `Status` is optional (default 200).
- App = `Init(ctx) / Exec(ctx) / Stop() []error`; no `Shutdown`; `Stop` must not wait for `Exec`. Register the HTTP App last.
- Config: no `json` tags; required unless `omitempty` / `default=` / `-`; `${VAR}` or `${VAR:-x}`; durations as `5s`; unknown keys are errors. No `os.Getenv` in code.
- Log with `rocklog` (configured by the logger service, registered first); never `Fatal`/`log.Fatal` outside `main()`.
- `pkg/` never imports `internal/`; `internal/handlers` never imports `cmd/`.
- Removed/old API that must not appear in v0.4 projects: `Name()` on services, `RegisterResource`, `rockengine.Resource`, `rockdelivery`, `internal/delivery`, `pkg/infra`, `github.com/arzab/gorock-kit/rockfiber` (and other non-core kit paths), `configs.InitFromFile`, `config:"default:X"`, `pkg/libs/go-kit`, `logrus`.

## Resources

- [references/recipes.md](references/recipes.md) — step-by-step code for every common task
- [references/kit-api.md](references/kit-api.md) — compact API and config keys of the core and catalog
- `${CLAUDE_SKILL_DIR}/templates/AGENTS.md` — project rules for any coding agent
- `${CLAUDE_SKILL_DIR}/templates/skeleton/` — minimal working project (used by `scripts/scaffold.sh`)
- Docs: https://arzab.github.io/gorock/ (architecture, guides per layer) · Core: https://github.com/arzab/gorock-kit · Catalog: https://github.com/arzab/gorock-catalog
