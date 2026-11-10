# GOROCK API (compact): gorock-kit v0.5.x core + gorock-catalog v0.5.x

Core: `github.com/arzab/gorock-kit/<module>` — rockengine, rockconfig, rockservices, rockhandlers. Modules don't import each other.
Catalog: `github.com/arzab/gorock-catalog/<module>` — rockhttp (on rockhandlers) and the modules moved as-is from kit v0.2.0 (rocklog, rockfiber, rockbus, rockcron, rocktelebot, rockbun, rockredis). Go 1.26+.
Full READMEs: https://github.com/arzab/gorock-kit/tree/main/<module>, https://github.com/arzab/gorock-catalog/tree/main/<module>

# Core (gorock-kit)

## rockengine — apps lifecycle

```go
type App interface { Init(ctx context.Context) error; Exec(ctx context.Context) error; Stop() []error }

e := rockengine.NewEngine()                 // .WithShutdownTimeout(d) .WithStopOnFatal(bool) .WithFatalHandler(func(name string, err error))
e.MustRegister(name, app, policy...)        // Register(name, app, policy...) (*Engine, error)
err := e.Run()                              // RunContext(ctx); blocks until SIGINT/SIGTERM, ctx cancel, or an app fails
e.StopApp(name); e.RestartApp(name); e.AppStatus(name); e.AppStatuses()

rockengine.RestartPolicy{
    MaxRetries: 3,             // 0 = none (default), -1 = unlimited
    Delay: time.Second,
    ResetAfter: time.Minute,   // a run that lasted ≥ this resets the retry counter
    RestartOnExit: false,      // restart when Exec returns nil
    OnFatal: func(err error){},
}
```
The engine runs **apps only** (there are no resources any more). Order: apps Init in registration order → each Exec in its goroutine. Shutdown: apps one by one in reverse (cancel ctx → Stop → wait Exec). Default `StopOnFatal(true)`: a permanently failed app stops the engine and `Run` returns its error. Shutdown timeout 30s.
Package-level default engine: `rockengine.Register`, `MustRegister`, `Run`, `SetStopOnFatal`, ...

## rockconfig — config file → struct

```go
cfg, err := rockconfig.InitFromFile[Configs]("./configs/configs.yaml") // .yaml .yml .json
cfg, err := rockconfig.InitFromFile[Configs](path, rockconfig.AllowUnknownKeys())
errors.Is(err, rockconfig.ErrMissingFields); errors.Is(err, rockconfig.ErrUnknownKeys)
```
Tags: `config:"name"`, `config:",omitempty"`, `config:",default=8080"` (must be last option), `config:",env=VAR"`, `config:"-"`. Keys = snake_case of field names. File values: `${VAR}`, `${VAR:-default}`, `$${` = literal. Durations as strings.

## rockservices — services catalog

```go
type Service interface { Init(ctx context.Context) error; Stop() error } // no Name(): the name is the key

var Orders = rockservices.NewKey[orders.Service]("orders") // typed key, declared once in pkg/services/services.go
rockservices.Add(c, Orders, ctor func(C) S, cfg C) S     // ctor type checked against the key at compile time; returns the service
Orders.Get() S; Orders.From(c) S; Orders.Name()          // Get reads the default catalog
rockservices.Init(ctx) error   // default catalog: fails naming declared-but-not-added keys; then Init in Add order, closes opened ones on failure
rockservices.Stop() error      // reverse order, joins errors, idempotent, timeout 30s names the hanging one
rockservices.ErrAlreadyStarted // second Init
rockservices.Default() *Catalog; rockservices.New() *Catalog; c.WithStopTimeout(d)
prev := rockservices.SetDefault(c) // tests only; no t.Parallel() in such tests
```
Panics: nil key, empty or duplicate name (also two keys with one name), nil service, ctor panic, Add after Init, `Get` of a service missing from the catalog.
Shared dependency: `pg := rockservices.Add(c, Postgres, postgres.NewService, cfg.Postgres)` then `rockservices.Add(c, Orders, func(cfg orders.Configs) orders.Service { return orders.NewService(cfg, pg) }, cfg.Orders)` — never `key.Get()` inside a constructor (it reads `Default()`). Two instances = two keys. The logger is a service added first.

## rockhandlers — transport-independent request handling

```go
type Handler[P, R any] func(c *Ctx[P, R]) error         // P, R: pointer-to-struct types
// *Ctx[P, R] is a context.Context; fields Params P (filled by the transport), Result R (pre-allocated)
c.Next() error; c.Route() string; c.Base(); c.SetContext(ctx)
rockhandlers.Done // return it to end the chain early with success; never leaves Serve/Run

route := rockhandlers.Chain("tasks.create", steps...)   // Route (generics erased); P, R inferred; generic steps instantiated
route = route.With(mws...)                              // first = outermost; later With wraps earlier
rockhandlers.Prepend(steps, pre...)                     // for group functions
type Middleware func(c rockhandlers.Context) error      // Context: context.Context + Route() + Next(); no Params access
rockhandlers.Timeout(d) Middleware                      // cooperative deadline

var UserKey = rockhandlers.NewLocal[*User]("user")      // UserKey.Set(c, u); u, ok := UserKey.Get(c); UserKey.MustGet(c)

rockhandlers.Invalid/Unauthorized/Forbidden/NotFound/Conflict/Unavailable(msg, args...) *Error   // .WithFields(map) .Wrap(err)
rockhandlers.KindOf(err)   // nil → KindNone; *Error → its kind; context.Canceled → KindCanceled; DeadlineExceeded → KindTimeout; else KindInternal
rockhandlers.MessageOf(err) // "" for Internal; FieldsOf(err); *PanicError{Value, Stack}

res, err := rockhandlers.Run[*Result](ctx, route, &Params{...})   // tests: whole chain without a transport
// transports: result, err := route.Serve(ctx, func(params any) error { /* decode into params */ })
```
Order: middlewares → transport decoding (fill) → `Params.Validate()` → steps. A link that returns nil without calling `Next` continues the chain automatically; `return rockhandlers.Done` ends it early with success. A plain error from `fill` reaches the client as Invalid "invalid request" (the cause stays for logs). On error the Result is dropped. Panics anywhere → Internal. Reusable steps needing request data are generic with an interface constraint (`func Bearer[P TokenParams, R any](c *rockhandlers.Ctx[P, R]) error`); models satisfy it by embedding (`steps.BearerAuth`). Short alias: `type Ctx = rockhandlers.Ctx[*Params, *Result]`.

# Catalog (gorock-catalog)

## rockhttp — HTTP App for rockhandlers (net/http, experimental)

```go
app := rockhttp.New(cfg rockhttp.Config, endpoints ...rockhttp.Endpoint).Use(mws ...rockhandlers.Middleware)
rockhttp.Endpoint{Method: "POST", Path: "/tasks/{id}", Handler: route, Status: http.StatusCreated} // Status default 200; empty Result struct → 204
rockhttp.SetCookie(c, *http.Cookie); rockhttp.SetHeader(c, k, v); rockhttp.Request(c) *http.Request   // inside steps
// Result types with special meaning: *rockhttp.File{Name, ContentType, Content io.Reader, ModTime, Download}, *rockhttp.Redirect{URL, Code}
app.Handler() http.Handler   // after Init — for httptest
```
Params tags: `path:"id"` (pattern `{id}`), `query:"x"` (slice = repeated), `header:"X-Y"`, `form:"x"` (`*multipart.FileHeader` / `[]*multipart.FileHeader` for uploads), JSON body via `json` tags. Tags in embedded structs are bound. Order: JSON, form, query, header, path.
Error → status: Invalid 400, Unauthorized 401, Forbidden 403, NotFound 404, Conflict 409, Unavailable 503, Timeout 504, Canceled 499 (no body), Internal 500 (message hidden). Body: `{code, status, kind, message, fields}`.
`rockhttp.Config` keys: `port` (required), `prefix`, `shutdown_timeout` (30s), `read_timeout` (15s), `write_timeout` (0 = none), `max_body_bytes` (10 MiB), `internal_message`. Code only: `OnError(r, route, err)` (response write failures only — log request errors in a rockhandlers middleware), `Middleware []func(http.Handler) http.Handler` (CORS, gzip…).

> The modules below were moved from gorock-kit v0.2.0 as-is (import path `github.com/arzab/gorock-catalog/<module>`). rockbun and rockredis services are `Init` + `Stop`, so they fit rockservices directly (`rockservices.Add(c, PostgresName, rockbun.NewService, cfg.Postgres)`) or as a child inside your own service; never `RegisterResource` (removed from the kit). rocklog is configured by a logger service (see the skeleton). The Apps (rockfiber, rockbus, rockcron, rocktelebot) don't take rockhandlers routes yet.

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
