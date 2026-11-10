# GOROCK API (compact): gorock-kit v0.6.x core + gorock-catalog v0.6.x

Core: `github.com/arzab/gorock-kit/<module>` — rockengine, rockconfig, rockservices, rockhandlers. Modules don't import each other.
Catalog: `github.com/arzab/gorock-catalog/<module>` — rockhttp (on rockhandlers) and the modules moved as-is from kit v0.2.0 (rocklog, rockfiber, rockbus, rockcron, rocktelebot, rockbun, rockredis). Go 1.26+.
Full READMEs: https://github.com/arzab/gorock-kit/tree/main/<module>, https://github.com/arzab/gorock-catalog/tree/main/<module>

# Core (gorock-kit)

## rockengine — Apps lifecycle

```go
type App interface { Init(ctx context.Context) error; Exec(ctx context.Context) error; Stop() []error }

engine := rockengine.New(map[string]rockengine.App{"bus": bus, "http": http}) // panics on empty name / nil App
engine.WithShutdownTimeout(30 * time.Second)                                 // default 30s
engine.WithRestarts(3, time.Second)                                           // optional: restart a failed App n times
err := engine.Run(ctx, "bus", "http")                                         // start order; panics on unknown/repeated names
```
Run: Init in order (failure → stop the initialised ones in reverse) → each Exec in a goroutine → wait for SIGINT/SIGTERM, ctx cancel or an App failure (Exec error/panic while its ctx is alive) → stop one by one in reverse (cancel ctx → Stop → wait Exec) under the timeout. Exec returning nil early just ends that App. Stop is called once per successful Init. Returns the failure joined with Stop errors. The project's `cmd/apps` has `func Run(ctx, cfg Configs) error` that builds the engine.

## rockconfig — config file → struct

```go
cfg, err := rockconfig.InitFromFile[Configs]("./configs/configs.yaml") // .yaml .yml .json
cfg, err := rockconfig.InitFromFile[Configs](path, rockconfig.AllowUnknownKeys())
errors.Is(err, rockconfig.ErrMissingFields); errors.Is(err, rockconfig.ErrUnknownKeys)
```
Tags: `config:"name"`, `config:",omitempty"`, `config:",default=8080"` (must be last option), `config:",env=VAR"`, `config:"-"`. Keys = snake_case of field names. File values: `${VAR}`, `${VAR:-default}`, `$${` = literal. Durations as strings.

## rockservices — services catalog

```go
type Service interface { Init(ctx context.Context) error; Stop() error }

c := rockservices.New(map[string]rockservices.Service{"logger": logger.NewService(cfg.Logger), "repository": repository.NewService(cfg.Repository)})
err := c.Init(ctx, "logger", "repository") // in this order; panics on unknown names; on failure stops what it started, in reverse
c.Get("repository") rockservices.Service    // panics if missing
err = c.Stop()                               // reverse init order; c.Stop("a", "b") in the given order; 30s timeout (WithStopTimeout)
```
The project's `pkg/services/services.go` owns the catalog: name constants, `Configs`, `var catalog`, `Init(ctx, cfg)` (builds the map from constructors, calls `catalog.Init` in order, logger first), `Stop()`, typed getters `func Repository() repository.Service { return catalog.Get(RepositoryName).(repository.Service) }`, and `UseCatalog(c) (restore func())` for tests (no `t.Parallel()`). A child opened outside its parent: create it first, put it into the map and pass it to the parent's constructor.

## rockhandlers — Fiber-style handler chains

```go
type Params interface{ Validate() error }    // filled by the App; runs before the handlers
type Response interface{ Normalize() error } // filled by the handlers; runs after a successful chain
type Handler[P Params, R Response] func(c *rockhandlers.Ctx, params P, response R) error
type Middleware = Handler[Params, Response]  // a handler for any chain

var Create = rockhandlers.Chain(checkAuth, create) // types inferred; mismatch = compile error; may be empty (→ ErrHandlerNotFound, HTTP 501)
Create.Use(mws...)                                 // middlewares in front; each Use goes in front of the previous ones
response, err := Create.Run(ctx, fill)             // Apps: fill func(params Params) error decodes the request
response, err := Create.Call(ctx, &CreateParams{...}) // tests

c.Next() error  // run the next handler; none left → rockhandlers.ErrHandlerNotFound
c.Set(key, v); c.Get(key) (any, bool); c.Context(); c.SetContext(ctx) // Ctx is a context.Context
rockhandlers.Timeout(d) Middleware

rockhandlers.Invalid/Unauthorized/Forbidden/NotFound/Conflict/Unavailable(msg, args...) *Error // .WithFields(map) .Wrap(err)
rockhandlers.KindOf(err) // nil → KindNone; *Error → its kind; Canceled; Timeout; else KindInternal
rockhandlers.MessageOf(err) // "" for Internal; FieldsOf(err); *PanicError; *NormalizeError
```
Chain semantics: `return nil` ends the chain; `return c.Next()` continues; `return err` stops. Run order: middlewares → fill (plain error → Invalid "invalid request") → Validate (plain error → Invalid with its text) → handlers → Normalize (error → *NormalizeError). Panics anywhere → Internal.

# Catalog (gorock-catalog)

## rockhttp — HTTP App for rockhandlers (net/http, experimental)

```go
app := rockhttp.New(cfg rockhttp.Config, endpoints ...rockhttp.Endpoint).Use(mws ...rockhandlers.Middleware)
rockhttp.Endpoint{Method: "POST", Path: "/tasks", Handler: tasks.Create, Status: http.StatusCreated} // Handler: a *rockhandlers.Pipeline; Status default 200; empty response struct → 204; empty chain → 501
rockhttp.SetCookie(c, *http.Cookie); rockhttp.SetHeader(c, k, v); rockhttp.Request(c) *http.Request   // inside handlers
// Response types with special meaning: *rockhttp.File{Name, ContentType, Content io.Reader, ModTime, Download}, *rockhttp.Redirect{URL, Code}
app.Handler() http.Handler   // after Init — for httptest
```
Params tags: `path:"id"` (pattern `{id}`), `query:"x"` (slice = repeated), `header:"X-Y"`, `form:"x"` (`*multipart.FileHeader` / `[]*multipart.FileHeader` for uploads), JSON body via `json` tags. Tags in embedded structs are bound. Order: JSON, form, query, header, path.
Error → status: Invalid 400, Unauthorized 401, Forbidden 403, NotFound 404, Conflict 409, Unavailable 503, Timeout 504, Canceled 499 (no body), Internal 500 (message hidden). Body: `{code, status, kind, message, fields}`.
`rockhttp.Config` keys: `port` (required), `prefix`, `shutdown_timeout` (30s), `read_timeout` (15s), `write_timeout` (0 = none), `max_body_bytes` (10 MiB), `internal_message`. Code only: `OnError(r, route, err)` (response write failures only — log request errors in a rockhandlers middleware), `Middleware []func(http.Handler) http.Handler` (CORS, gzip…).

> The modules below were moved from gorock-kit v0.2.0 as-is (import path `github.com/arzab/gorock-catalog/<module>`). rockbun and rockredis services are `Init` + `Stop`, so they fit the rockservices map directly (`"postgres": rockbun.NewService(cfg.Postgres)`) or as a child inside your own service; never `RegisterResource` (removed from the kit). rocklog is configured by a logger service (see the skeleton). The Apps (rockfiber, rockbus, rockcron, rocktelebot) don't take rockhandlers chains yet; their READMEs still show the old engine registration — use `rockengine.New(map).Run(ctx, names...)`.

## rocklog — logger (log/slog)

```go
rocklog.Init(rocklog.Config{Level: rocklog.LevelInfo, Format: rocklog.FormatJSON, TimeFormat: time.RFC3339, Caller: true})
rocklog.Debug/Info/Warn/Error(msg, fields...)   // Fatal exits the process — only in main()
rocklog.Str(k, v) Int Int64 Float64 Bool Dur Time Err(err) Any(k, v) F(k, v) Stringer
l := rocklog.With(fields...); l = rocklog.Named("db"); l.Info(...)
ctx = rocklog.WithContext(ctx, fields...); rocklog.FromContext(ctx).Info(...)
rocklog.ContextWithLogger(ctx, l); rocklog.New(cfg) Logger; rocklog.SetDefault(l)
```
Levels: `LevelDebug LevelInfo LevelWarn LevelError LevelFatal`. Formats: `FormatText FormatJSON`. `rocklog.Config` is not file-loadable (io.Writer, ints) — the logger service (`pkg/services/logger`) maps its own Configs to it in `Init`.

## rockfiber — HTTP App on Fiber v2 (older model)

Prefer **rockhttp** for new code. rockfiber keeps its own endpoint model (`rockfiber.GET(path, fiber.Handler...)`, `DefaultHandler[Params]()`, `GetFromContext`, `NewError(code, msg)`); see its README if a project already uses it.

## rockbus — in-process event bus (App)

```go
bus := rockbus.NewApp(rockbus.Config{QueueSize: 1024}, onError func(ctx, rockbus.Event, error), subs ...rockbus.Subscription)
rockbus.SetDefault(bus)                 // enables package-level Publish/PublishAsync/Subscribe
rockbus.On(topic rockbus.Topic, handler rockbus.Handler) Subscription   // Handler = func(ctx) error
rockbus.PublishAsync(ctx, rockbus.Event{Topic: t, Payload: v})          // queued; errors → onError
err := rockbus.Publish(ctx, ev)                                         // sync, in caller goroutine
v, err := rockbus.Payload[T](ctx); rockbus.CurrentTopic(ctx)
err := bus.Subscribe(topic, h)          // new topic only before Init (else ErrNoWorker)
```
Errors: `ErrQueueFull`, `ErrAppStopped`, `ErrNoWorker`, `ErrPanic`, `ErrPayload`. One ordered queue per topic; in-memory, at-most-once. With `PublishAsync` the handler ctx keeps the publisher's values (logger, trace id) but not its cancellation.

## rockcron — scheduler (App)

```go
app := rockcron.NewApp(rockcron.Config{Location: "Europe/Moscow"}, onError func(ctx, rockcron.Job, error), jobs ...rockcron.Job)
rockcron.Every(name, 5*time.Minute, handlers ...rockcron.Handler) // Handler = func(ctx) error; min 1s
rockcron.Cron(name, "0 3 * * *", handlers...)                     // 5-field cron
rockcron.JobName(job)
// custom: type Job interface { Schedule() string; Run(ctx) error } + optional JobName() string
```
Handlers in a job run in sequence; first error stops the chain. A job is skipped while its previous run is still running. Invalid schedule/location → error from Init.

## rocktelebot — Telegram bot (App, telebot.v3)

```go
app := rocktelebot.NewApp(rocktelebot.Config{Token: "..."}, []tele.MiddlewareFunc{rocktelebot.Recovery(fn), rocktelebot.Logger(fn)}, handlers...)
rocktelebot.Command("/start", h tele.HandlerFunc, mws...).WithDesc("Start")   // desc → Telegram command menu
rocktelebot.GetHandler(tele.OnText, h, mws...)
kb := rocktelebot.NewInlineKeyboard(); kb.Row(kb.Data("Yes", "yes"), kb.URL("Site", url)).Markup()
rk := rocktelebot.NewReplyKeyboard(); rk.Row(rk.Text("A")).OneTime().Markup(); rocktelebot.RemoveKeyboard()
```
Config keys: `token` (required), `webhook.listen`, `webhook.public_url` (empty listen = long polling).

## rockbun — PostgreSQL (service, use as a child)

```go
db := rockbun.NewService(rockbun.Configs{...})   // Service: Init(ctx) Stop() DB() *bun.DB Ping(ctx) RunInTx(ctx, *sql.TxOptions, func(ctx, bun.Tx) error)
db.DB().NewSelect().Model(&m).Where("id = ?", id).Scan(ctx)   // full bun query builder
```
Config keys (all optional): `dsn` (wins over fields), `host` (localhost), `port` (5432), `user`, `password`, `database`, `ssl_mode` (disable; use verify-full in prod), `max_open_conns` (25), `max_idle_conns` (5), `conn_max_lifetime` (1h), `conn_max_idle_time` (30m). Code only: `QueryHooks []bun.QueryHook`. Credentials are URL-escaped. After Stop, queries return `sql: database is closed`.

## rockredis — Redis (service, use as a child)

```go
r := rockredis.NewService(rockredis.Configs{Addr: "localhost:6379"})
// Service: Init Stop Ping Get Set SetNX Del Exists Expire TTL Incr IncrBy
//          HSet HGet HGetAll HDel HExists LPush RPush LPop RPop LLen LRange
//          SAdd SRem SMembers SIsMember SCard ZAdd ZRem ZScore ZRank ZRange ZCard Publish Subscribe
errors.Is(err, rockredis.ErrNil)        // key missing
rockredis.Client(r).ZRevRank(ctx, k, m) // underlying *redis.Client for anything else (nil before Init)
```
Config keys (all optional): `addr`, `password`, `db`, `dial_timeout` (5s), `read_timeout` (3s), `write_timeout` (3s), `pool_size` (10), `min_idle_conns`, `tls_enabled`.
