---
name: service
description: Build or extend a Go backend service with the GOROCK architecture and gorock-kit (rockengine, rockconfig, rocklog, rockfiber, rockbus, rockcron, rocktelebot, rockbun, rockredis). Use when creating a new Go service with gorock/gorock-kit, or when the project's go.mod requires github.com/arzab/gorock-kit or it has the cmd/ internal/ pkg/ GOROCK layout — e.g. adding an HTTP endpoint, a PostgreSQL/Redis service, an event, a scheduled job, a Telegram bot, a custom App, or a config field.
---

# GOROCK service

GOROCK splits a Go service into three layers; gorock-kit provides the code that is the same in every project. Your job is to put each piece of code in the right layer and use the kit's contracts exactly.

## Workflow

1. **Orient.** Read the project's `go.mod` and `AGENTS.md` if present. Check the kit version (`github.com/arzab/gorock-kit/... v0.2.x`) and the layout (`cmd/main`, `cmd/apps`, `internal/delivery`, `pkg/services`, `configs/`).
   - No project yet → scaffold one (recipe 1). Don't hand-write the skeleton.
   - Existing project without `AGENTS.md` → copy `${CLAUDE_SKILL_DIR}/templates/AGENTS.md` to the root and offer to add `CLAUDE.md` containing `@AGENTS.md`.
2. **Pick the recipe** for the task in [references/recipes.md](references/recipes.md) and follow it file by file. For exact signatures, open [references/kit-api.md](references/kit-api.md) — don't guess kit APIs.
3. **Wire it up.** Every new piece is reachable from exactly one place: endpoint → `endpoints.All()`, event → `events.Subscriptions()`, job → `jobs.All()`, service → `services.Register` (as a resource), app → `apps.Register`. Config field → layer `Configs` **and** `configs/*.yaml`.
4. **Verify.** `gofmt -l .` prints nothing, `go build ./... && go vet ./...` pass. If config or wiring changed, start the service (`go run ./cmd/main`) and confirm it gets past config loading; rockconfig prints every missing or unknown key at once. Fix and re-run until clean.
5. **Report** what was added and where, and anything the user must provide (env vars, DB schema, tokens).

## Rules that are easy to get wrong

- App = `Init(ctx) error / Exec(ctx) error / Stop() []error`. There is no `Shutdown`; finishing work = ctx cancellation. `Stop` must not wait for `Exec`.
- Services implement `Init(ctx) error / Stop() error` and are registered with `e.MustRegisterResource`, never started by hand in `main`.
- Register the HTTP app last in `apps.Register` (apps stop in reverse order).
- Config: no `json` tags; required unless `omitempty` / `default=` / `-`; `${VAR}` or `${VAR:-x}` (never `$VAR`); durations as `5s`; unknown keys are errors.
- HTTP errors: `rockfiber.NewError(code, msg)` for client-visible statuses; plain wrapped errors become masked 500s. Pass `c.UserContext()` to services.
- Log with `rocklog`; never `Fatal`/`log.Fatal` outside `main()`.
- `pkg/` never imports `internal/`; services take `any` or their own interfaces, not `internal/models` types.
- Old API that must not appear: `configs.InitFromFile`, `config:"default:X"`, `config:"ignore"`, `BuildFiberEndpoint`, `pkg/libs/go-kit`, `logrus`, an App with `Shutdown()`.

## Resources

- [references/recipes.md](references/recipes.md) — step-by-step code for every common task
- [references/kit-api.md](references/kit-api.md) — compact API and config keys of all kit modules
- `${CLAUDE_SKILL_DIR}/templates/AGENTS.md` — project rules for any coding agent
- `${CLAUDE_SKILL_DIR}/templates/skeleton/` — minimal working project (used by `scripts/scaffold.sh`)
- Complete reference project: https://github.com/arzab/gorock/tree/main/examples/taskapi · Docs: https://arzab.github.io/gorock/
