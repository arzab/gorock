# gorock-kit v0.2.x API (compact)

Import path: `github.com/arzab/gorock-kit/<module>`. Go 1.26+. Modules don't import each other.
Full READMEs: https://github.com/arzab/gorock-kit/tree/main/<module>

## rockengine — lifecycle

```go
type App interface { Init(ctx context.Context) error; Exec(ctx context.Context) error; Stop() []error }
type Resource interface { Init(ctx context.Context) error; Stop() error }

e := rockengine.NewEngine()                 // .WithShutdownTimeout(d) .WithStopOnFatal(bool) .WithFatalHandler(func(name string, err error))
e.MustRegisterResource(name, res)           // RegisterResource(name, res) (*Engine, error)
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
Order: resources init (registration order) → apps init → apps exec. Shutdown: apps one by one in reverse (cancel ctx → Stop → wait Exec), then resources in reverse. Default `StopOnFatal(true)`: a permanently failed app stops the engine and `Run` returns its error. Shutdown timeout 30s.
Package-level default engine: `rockengine.Register`, `RegisterResource`, `Run`, `SetStopOnFatal`, ...

## rockconfig — config file → struct

```go
cfg, err := rockconfig.InitFromFile[Configs]("./configs/configs.yaml") // .yaml .yml .json
cfg, err := rockconfig.InitFromFile[Configs](path, rockconfig.AllowUnknownKeys())
errors.Is(err, rockconfig.ErrMissingFields); errors.Is(err, rockconfig.ErrUnknownKeys)
```
Tags: `config:"name"`, `config:",omitempty"`, `config:",default=8080"` (must be last option), `config:",env=VAR"`, `config:"-"`. Keys = snake_case of field names. File values: `${VAR}`, `${VAR:-default}`, `$${` = literal. Durations as strings.

## rocklog — logger (log/slog)

```go
rocklog.Init(rocklog.Config{Level: rocklog.LevelInfo, Format: rocklog.FormatJSON, TimeFormat: time.RFC3339, Caller: true})
rocklog.Debug/Info/Warn/Error(msg, fields...)   // Fatal exits the process — only in main()
rocklog.Str(k, v) Int Int64 Float64 Bool Dur Time Err(err) Any(k, v) F(k, v) Stringer
l := rocklog.With(fields...); l = rocklog.Named("db"); l.Info(...)
ctx = rocklog.WithContext(ctx, fields...); rocklog.FromContext(ctx).Info(...)
rocklog.ContextWithLogger(ctx, l); rocklog.New(cfg) Logger; rocklog.SetDefault(l)
```
Levels: `LevelDebug LevelInfo LevelWarn LevelError LevelFatal`. Formats: `FormatText FormatJSON`. `rocklog.Config` is not file-loadable (io.Writer, ints) — map from your own Configs in `pkg/infra/logs`.

## rockfiber — HTTP App (Fiber v2)

```go
app := rockfiber.New(cfg rockfiber.Config, endpoints ...rockfiber.FiberEndpoint) // App; also SetEndpoints
rockfiber.GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS(path, handlers ...fiber.Handler) FiberEndpoint
rockfiber.NewEndpoint(method, path, handlers...) FiberEndpoint

rockfiber.DefaultHandler[Params]()          // parse query/body/headers/path into Params, call Validate, store in Locals("params")
rockfiber.DefaultHandler[Params]("key")     // custom Locals key
rockfiber.GetFromContext[T](c, key) (*T, error)
rockfiber.HandlerInitInCtx[T](key) fiber.Handler   // put new(T) into Locals
rockfiber.ParseRequest(c, &p) error
rockfiber.NewError(code, msg) *ErrorResponse      // .WithSource(s) .WithAction(a); JSON {code,status,message,source,action}
rockfiber.TraceIdFromContext(c.UserContext()) string
```
Params tags: `query:"x"`, `json:"x"`, `form:"x"` (also `*multipart.FileHeader`), `reqHeader:"X-Y"`, `params:"id"` (path `/:id`). Params type needs `func (p *Params) Validate(*fiber.Ctx) error`.

`rockfiber.Config` (file keys): `port` (required), `endpoints_path_prefix`, `admin_endpoints_path` (/admin), `admin_password`, `shutdown_timeout`, `tls_cert_file`, `tls_key_file`, `compress`, `helmet`, `pprof`, `use_trace_id`, `request_timeout`, `mask_internal_server_error_message`.
Code only (`config:"-"`): `App fiber.Config`, `RateLimit *limiter.Config`, `Swagger *rockfiber.SwaggerConfig`, `MonitoringConfig *monitor.Config`, `CorsConfig *cors.Config` (nil = CORS off), `NotFound fiber.Handler`, `OnRequest func(path, traceId string)`, `OnError func(traceId string, err *rockfiber.ErrorResponse)`.
Built-in: `GET /status`, `/admin/metrics`, `/admin/debug/pprof/*` (Pprof: true), behind `X-Admin-Password`.
Error mapping: `*ErrorResponse` → its code; `*fiber.Error` → its code; other errors → 500 (message masked if configured; `OnError` gets the real one); panics → 500.

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

## rockbun — PostgreSQL (Resource)

```go
db := rockbun.NewService(rockbun.Configs{...})   // Service: Init(ctx) Stop() DB() *bun.DB Ping(ctx) RunInTx(ctx, *sql.TxOptions, func(ctx, bun.Tx) error)
db.DB().NewSelect().Model(&m).Where("id = ?", id).Scan(ctx)   // full bun query builder
```
Config keys (all optional): `dsn` (wins over fields), `host` (localhost), `port` (5432), `user`, `password`, `database`, `ssl_mode` (disable; use verify-full in prod), `max_open_conns` (25), `max_idle_conns` (5), `conn_max_lifetime` (1h), `conn_max_idle_time` (30m). Code only: `QueryHooks []bun.QueryHook`. Credentials are URL-escaped. After Stop, queries return `sql: database is closed`.

## rockredis — Redis (Resource)

```go
r := rockredis.NewService(rockredis.Configs{Addr: "localhost:6379"})
// Service: Init Stop Ping Get Set SetNX Del Exists Expire TTL Incr IncrBy
//          HSet HGet HGetAll HDel HExists LPush RPush LPop RPop LLen LRange
//          SAdd SRem SMembers SIsMember SCard ZAdd ZRem ZScore ZRank ZRange ZCard Publish Subscribe
errors.Is(err, rockredis.ErrNil)        // key missing
rockredis.Client(r).ZRevRank(ctx, k, m) // underlying *redis.Client for anything else (nil before Init)
```
Config keys (all optional): `addr`, `password`, `db`, `dial_timeout` (5s), `read_timeout` (3s), `write_timeout` (3s), `pool_size` (10), `min_idle_conns`, `tls_enabled`.
