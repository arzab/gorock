---
name: service
description: Build or extend a Go backend service with the GOROCK architecture — gorock-kit core (rockengine, rockconfig, rockservices, rockhandlers) and gorock-catalog (rockhttp, rocklog, rockbun, rockredis, rockbus, rockcron, rocktelebot, rockfiber). Use when creating a new Go service with gorock, or when the project's go.mod requires github.com/arzab/gorock-kit or github.com/arzab/gorock-catalog or it has the cmd/ internal/ pkg/ GOROCK layout — e.g. adding an HTTP endpoint, a request step or middleware, a PostgreSQL/Redis service, an event, a scheduled job, a Telegram bot, a custom App, or a config field.
---

# GOROCK service

GOROCK splits a Go service into three layers. gorock-kit is the core every service needs (apps lifecycle, config, services layer, request handling); gorock-catalog holds technology-specific modules (HTTP, PostgreSQL, Redis, logger…). Your job is to put each piece of code in the right layer and use the contracts exactly.

## Workflow

1. **Orient.** Read the project's `go.mod` and `AGENTS.md` if present. Check the kit version: `v0.3.x` = current model (services via rockservices, routes via rockhandlers, HTTP via rockhttp). A project on `v0.2.x` (resources in the engine, rockfiber endpoints from `gorock-kit/rockfiber`) uses the old model — keep its style unless asked to migrate.
   - No project yet → scaffold one (recipe 1). Don't hand-write the skeleton.
   - Existing project without `AGENTS.md` → copy `${CLAUDE_SKILL_DIR}/templates/AGENTS.md` to the root and offer to add `CLAUDE.md` containing `@AGENTS.md`.
2. **Pick the recipe** for the task in [references/recipes.md](references/recipes.md) and follow it file by file. For exact signatures, open [references/kit-api.md](references/kit-api.md) — don't guess APIs.
3. **Wire it up.** Every new piece is reachable from exactly one place: route → `endpoints.All()` (rockhttp.Endpoint), event → `events.Subscriptions()`, job → `jobs.All()`, service → `services.Register` (`rockservices.Add`), app → `apps.Register`. Config field → layer `Configs` **and** `configs/*.yaml`.
4. **Verify.** `gofmt -l .` prints nothing, `go build ./... && go vet ./...` pass, `go test ./...` for routes you touched (`rockhandlers.Run`). If config or wiring changed, start the service (`go run ./cmd/main`) and confirm it gets past config loading; rockconfig prints every missing or unknown key at once.
5. **Report** what was added and where, and anything the user must provide (env vars, DB schema, tokens).

## Rules that are easy to get wrong

- cmd/main order: `rockconfig` → `infra.Init` → `services.Register(rockservices.Default(), cfg.Services)` → `rockservices.Init(ctx)` → `defer rockservices.Stop()` → `engine.Run()`. The engine runs **apps only**; there is no `RegisterResource`.
- Service = `Name() / Init(ctx) / Stop()`, `const Name` + `func Get() Service` in its package; `NewService` opens no connections. Catalog services (rockbun, rockredis) have no `Name()` — wrap them as child services.
- Route = `rockhandlers.Chain(name, steps...)`; steps are `func(c *rockhandlers.Ctx[*Params, *Result]) error`. Routes know nothing about HTTP: method/path/status live in `rockhttp.Endpoint`. Validation in `Params.Validate()`. Errors are kinds (`rockhandlers.NotFound(...)`), never HTTP codes. Pass `c` (a context.Context) to services.
- Reusable steps that need request data: generic with an interface constraint, models embed the matching struct. Never type-assert `Params`. Middlewares (`func(c rockhandlers.Context) error`) are only for logic that doesn't read request data.
- App = `Init(ctx) error / Exec(ctx) error / Stop() []error`. No `Shutdown`; finishing work = ctx cancellation. `Stop` must not wait for `Exec`. Register the HTTP app last.
- Config: no `json` tags; required unless `omitempty` / `default=` / `-`; `${VAR}` or `${VAR:-x}` (never `$VAR`); durations as `5s`; unknown keys are errors.
- Log with `rocklog` (catalog); never `Fatal`/`log.Fatal` outside `main()`.
- `pkg/` never imports `internal/`; services take `any` or their own interfaces, not `internal/models` types.
- Old API that must not appear in v0.3 projects: `RegisterResource`, `rockengine.Resource`, `rockdelivery`, `github.com/arzab/gorock-kit/rockfiber` (and other non-core kit paths), `configs.InitFromFile`, `config:"default:X"`, `pkg/libs/go-kit`, `logrus`, an App with `Shutdown()`.

## Resources

- [references/recipes.md](references/recipes.md) — step-by-step code for every common task
- [references/kit-api.md](references/kit-api.md) — compact API and config keys of the core and catalog modules
- `${CLAUDE_SKILL_DIR}/templates/AGENTS.md` — project rules for any coding agent
- `${CLAUDE_SKILL_DIR}/templates/skeleton/` — minimal working project (used by `scripts/scaffold.sh`)
- Docs: https://arzab.github.io/gorock/ · Core: https://github.com/arzab/gorock-kit · Catalog: https://github.com/arzab/gorock-catalog
