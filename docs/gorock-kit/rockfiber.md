# rockfiber

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockfiber/README_RU.md — правьте README в gorock-kit. -->

::: tip Место в GOROCK · Engine · Apps
Готовый HTTP App: регистрируется в `cmd/apps`, принимает endpoints из `internal/delivery/endpoints`. В delivery используются его хелперы: `GET/POST`, `DefaultHandler`, `GetFromContext`, `NewError`. → [Apps](/architecture/apps), [Delivery](/architecture/delivery)
:::

```sh
go get github.com/arzab/gorock-kit/rockfiber@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-kit/tree/main/rockfiber)

Обёртка над fiber для HTTP-сервера с интеграцией в [rockengine](./rockengine): структурированная маршрутизация, парсинг запросов, обработка ошибок и production-middlewares.

## Обзор

`rockfiber` оборачивает [gofiber/fiber v2](https://github.com/gofiber/fiber) и предоставляет:
- Регистрацию маршрутов с типизированным парсингом и валидацией запросов
- Совместимость с интерфейсом `App` из rockengine
- Встроенные middlewares: CORS, сжатие, security headers, rate limiting, трассировка, pprof, метрики
- Структурированные ответы на ошибки
- Поддержку TLS / HTTPS

## Быстрый старт

```go
app := rockfiber.New(
    rockfiber.Config{
        Port:       "8080",
        UseTraceId: true,
        Compress:   true,
    },
    rockfiber.GET("/hello", helloHandler),
    rockfiber.POST("/users", createUserHandler),
)

// Автономный запуск
ctx := context.Background()
if err := app.Init(ctx); err != nil {
    log.Fatal(err)
}
if err := app.Exec(ctx); err != nil {
    log.Fatal(err)
}

// Или в rockengine
engine.MustRegister("http", app, rockengine.RestartPolicy{})
engine.Run()
```

## Загрузка конфига из файла

`rockfiber.Config` совместим с `rockconfig.InitFromFile`. Поля с функциями и сложными сторонними типами помечены `config:"-"` и задаются в коде — всё остальное можно загрузить из YAML или JSON файла.

```yaml
# config.yaml
port: "8080"
endpoints_path_prefix: "/api/v1"
admin_password: "secret"
shutdown_timeout: "30s"
use_trace_id: true
compress: true
helmet: true
request_timeout: "5s"
tls_cert_file: "/etc/ssl/cert.pem"  # опционально
tls_key_file:  "/etc/ssl/key.pem"   # опционально
mask_internal_server_error_message: "internal server error"
```

```go
cfg, err := rockconfig.InitFromFile[rockfiber.Config]("config.yaml")
if err != nil {
    log.Fatal(err)
}

// Поля которые задаются в коде
cfg.OnRequest = func(path, traceId string) { ... }
cfg.OnError   = func(traceId string, err *rockfiber.ErrorResponse) { ... }
cfg.CorsConfig = &cors.Config{AllowOrigins: "https://myapp.com"}
cfg.RateLimit  = &limiter.Config{Max: 100, Expiration: time.Minute}

app := rockfiber.New(*cfg,
    rockfiber.GET("/hello", helloHandler),
    rockfiber.POST("/users", createUserHandler),
)
```

| Из файла | Только в коде (`config:"-"`) |
|----------|------------------------------|
| `port`, `admin_password` | `App fiber.Config` |
| `endpoints_path_prefix`, `admin_endpoints_path` | `RateLimit`, `CorsConfig` |
| `use_trace_id`, `compress`, `helmet`, `pprof` | `Swagger`, `MonitoringConfig` |
| `request_timeout`, `shutdown_timeout` | `NotFound`, `OnRequest`, `OnError` |
| `tls_cert_file`, `tls_key_file` | |
| `mask_internal_server_error_message` | |

## Config

```go
type Config struct {
    App  fiber.Config  // нативный конфиг fiber (body limit, prefork и др.)
    Port string        // порт, например "8080"

    EndpointsPathPrefix string        // общий префикс маршрутов, например "/api/v1"
    AdminEndpointsPath  string        // префикс admin-роутов; по умолчанию "/admin"
    AdminPassword       string        // значение заголовка X-Admin-Password; пусто = запрет всем
    ShutdownTimeout     time.Duration // таймаут graceful shutdown; по умолчанию 30s

    // TLS — оба поля должны быть заданы вместе.
    TLSCertFile string
    TLSKeyFile  string

    // Глобальные middlewares (все маршруты включая /status и admin).
    Compress bool // сжатие ответов: gzip/deflate/brotli
    Helmet   bool // security headers: X-Frame-Options, CSP, HSTS и др.

    Pprof bool // pprof по адресу <AdminEndpointsPath>/debug/pprof/, за admin-паролем; по умолчанию выключен

    // Endpoint-middlewares (только маршруты под EndpointsPathPrefix).
    UseTraceId     bool
    RequestTimeout time.Duration   // deadline контекста; 0 = выключено
    RateLimit      *limiter.Config // nil = выключено

    Swagger          *SwaggerConfig
    MonitoringConfig *monitor.Config
    CorsConfig       *cors.Config   // nil = CORS выключен (см. раздел CORS)

    MaskInternalServerErrorMessage string        // маскировка 500-ошибок; см. Обработка ошибок
    NotFound                       fiber.Handler // catch-all для несуществующих маршрутов

    OnRequest func(path, traceId string)
    OnError   func(traceId string, err *ErrorResponse)
}
```

## Маршруты

### Методы-хелперы

```go
rockfiber.GET(path, handlers...)
rockfiber.POST(path, handlers...)
rockfiber.PUT(path, handlers...)
rockfiber.PATCH(path, handlers...)
rockfiber.DELETE(path, handlers...)
rockfiber.HEAD(path, handlers...)
rockfiber.OPTIONS(path, handlers...)
```

Middleware-хендлеры передаются перед финальным:

```go
rockfiber.GET("/profile", authMiddleware, getProfileHandler)
```

### Произвольный метод

```go
rockfiber.NewEndpoint("PURGE", "/cache", purgeHandler)
```

### Кастомный тип маршрута

Реализуйте интерфейс `FiberEndpoint` для своих дескрипторов маршрутов:

```go
type FiberEndpoint interface {
    GetPath() string
    GetMethod() string
    GetHandlers() []fiber.Handler
}
```

## Парсинг запросов

### DefaultHandler

Универсальный middleware: парсит запрос, валидирует и кладёт результат в `ctx.Locals`:

```go
type CreateUserParams struct {
    Name  string `json:"name"`
    Email string `json:"email" query:"email"`
}

func (p *CreateUserParams) Validate(ctx *fiber.Ctx) error {
    if p.Name == "" {
        return rockfiber.NewError(http.StatusBadRequest, "name is required")
    }
    return nil
}

rockfiber.POST("/users",
    rockfiber.DefaultHandler[CreateUserParams](),
    createUserHandler,
)

func createUserHandler(ctx *fiber.Ctx) error {
    params, err := rockfiber.GetFromContext[CreateUserParams](ctx, "params")
    // ...
}
```

Кастомный ключ если в цепочке несколько `DefaultHandler`:

```go
rockfiber.DefaultHandler[CreateUserParams]("user")
rockfiber.GetFromContext[CreateUserParams](ctx, "user")
```

### ParseRequest

`ParseRequest` заполняет любую структуру из входящего запроса. Поддерживаемые теги:

| Тег              | Источник                        |
|------------------|---------------------------------|
| `query:"name"`   | Query-параметр                  |
| `json:"name"`    | JSON / XML / form тело запроса  |
| `reqHeader:"name"` | Заголовок запроса             |
| `params:"name"`  | URL path-параметр               |
| `form:"name"`    | Multipart файл                  |

Порядок парсинга: **query → body → headers → path params**. Path params применяются последними и перезаписывают совпадающие поля.

### Загрузка файлов

```go
type UploadParams struct {
    Title    string                    `json:"title"`
    File     *multipart.FileHeader     `form:"file"`    // один файл
    Gallery  []*multipart.FileHeader   `form:"gallery"` // несколько файлов
}
```

### Интерфейс Params

`Params[T]` — ограничение для структуры параметров при использовании с `DefaultHandler`:

```go
type Params[T any] interface {
    *T
    Validate(ctx *fiber.Ctx) error
}
```

## Обработка ошибок

### ErrorResponse

Все ошибки возвращаются как JSON:

```json
{
  "code": 404,
  "status": "Not Found",
  "message": "user not found",
  "source": "user-service",
  "action": "get"
}
```

`source` и `action` не включаются в JSON если пустые.

```go
// Простая ошибка
return rockfiber.NewError(http.StatusNotFound, "user not found")

// С контекстом
return rockfiber.NewError(http.StatusInternalServerError, "query failed").
    WithSource("database").
    WithAction("get-user")
```

### Маскировка внутренних ошибок

Установите `MaskInternalServerErrorMessage` чтобы скрыть сырые 500-ошибки от клиентов:

```go
Config{
    MaskInternalServerErrorMessage: "internal server error",
}
```

Маскируются только неожиданные ошибки (500 с source `internal`); возвращённые вами `ErrorResponse` и `fiber.Error` сохраняют своё сообщение. Клиент не может обойти маскировку. `OnError` всегда получает реальное сообщение — логируйте там.

### Кастомный error handler

Если задан `cfg.App.ErrorHandler`, он имеет полный приоритет, наш не регистрируется:

```go
Config{
    App: fiber.Config{
        ErrorHandler: myCustomHandler,
    },
}
```

### Хук логирования ошибок

```go
Config{
    OnError: func(traceId string, err *rockfiber.ErrorResponse) {
        log.Error("request failed",
            rocklog.Str("trace_id", traceId),
            rocklog.Int("code", err.Code),
            rocklog.Str("message", err.Message),
        )
    },
}
```

## Трассировка

При `UseTraceId: true` каждый запрос под `EndpointsPathPrefix` получает заголовок `X-Trace-Id`:
- Если клиент присылает `X-Trace-Id` — значение пропагируется (санитизация: max 64 символа, без управляющих символов).
- Иначе генерируется новый UUID v4.

Доступен в хендлерах:

```go
traceId, _ := ctx.Locals(rockfiber.TraceIdKey).(string)
```

И автоматически добавляется в заголовки ответа.

## Хук логирования запросов

```go
Config{
    UseTraceId: true,
    OnRequest: func(path, traceId string) {
        log.Info("request received",
            rocklog.Str("path", path),
            rocklog.Str("trace_id", traceId),
        )
    },
}
```

`OnRequest` выполняется после `TraceIdMiddleware` — trace ID всегда доступен.

## Rate Limiting

```go
Config{
    RateLimit: &limiter.Config{
        Max:        100,
        Expiration: 1 * time.Minute,
    },
}
```

Применяется только к маршрутам под `EndpointsPathPrefix`. Admin-маршруты и `/status` не затрагиваются.

## Таймаут запросов

```go
Config{
    RequestTimeout: 5 * time.Second,
}
```

Устанавливает `context.WithTimeout` deadline на каждый запрос. Любая context-aware операция (запросы к БД, HTTP-клиенты) автоматически получит этот deadline. Хендлер не прерывается принудительно — код, игнорирующий `ctx.Done()`, выполнится до конца.

## TLS / HTTPS

```go
Config{
    Port:        "443",
    TLSCertFile: "/etc/ssl/cert.pem",
    TLSKeyFile:  "/etc/ssl/key.pem",
}
```

Оба поля должны быть заданы вместе. Если задано только одно — `Init` вернёт ошибку.

## Сжатие

```go
Config{Compress: true}
```

Включает gzip / deflate / brotli на основе заголовка `Accept-Encoding`. Применяется глобально. Middleware автоматически пропускает уже сжатые форматы.

## Security Headers (Helmet)

```go
Config{Helmet: true}
```

Добавляет: `X-XSS-Protection`, `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Content-Security-Policy`, `Permissions-Policy`, `Strict-Transport-Security`.

> **Важно:** Дефолтный CSP может конфликтовать со Swagger UI. При одновременном использовании `Helmet: true` и `Swagger` настройте кастомный Helmet config через `cfg.App`.

## Admin-маршруты

Зарегистрированы под `AdminEndpointsPath` (по умолчанию `/admin`), защищены заголовком `X-Admin-Password`:

| Маршрут                    | Описание                      |
|----------------------------|-------------------------------|
| `GET /admin/metrics`       | Fiber monitor dashboard       |
| `GET /admin/debug/pprof/*` | Go pprof endpoints (только при `Pprof: true`) |

Если `AdminPassword` пустой — все запросы к admin возвращают 401.

## Встроенные маршруты

| Маршрут      | Описание                    |
|--------------|-----------------------------|
| `GET /status` | Возвращает `{"status":"ok"}` |

## CORS

```go
// Без CORS middleware (по умолчанию) — браузер блокирует кросс-доменные запросы
Config{CorsConfig: nil}

// Явно разрешить все origins (публичный API без credentials)
Config{CorsConfig: &cors.Config{}}

// Ограничить origins
Config{
    CorsConfig: &cors.Config{
        AllowOrigins: "https://app.example.com",
        AllowHeaders: "Content-Type, Authorization",
    },
}
```

> **Изменено:** раньше `nil` разрешал все origins (`*`). Теперь CORS-заголовки не отправляются, пока не задан `CorsConfig`. Запросы с того же origin и не-браузерные клиенты не затронуты.

## Хелперы контекста

```go
// Инициализировать значение в request locals
rockfiber.HandlerInitInCtx[MyStruct]("my-key")

// Получить его в следующем хендлере (типобезопасно)
val, err := rockfiber.GetFromContext[MyStruct](ctx, "my-key")

// Trace ID (при UseTraceId) из любого контекста, производного от c.UserContext()
id := rockfiber.TraceIdFromContext(c.UserContext())
```

Передавайте `c.UserContext()` в сервисы, чтобы они могли прочитать trace ID — например, добавить его в логгер через `rocklog.WithContext`.

## Восстановление после паники

`RecoverHandler` зарегистрирован глобально. При панике:
1. Стек трейс записывается в `stderr`
2. Клиент получает 500 `ErrorResponse` через `ErrorHandler`

## Ограничения

- `RequestTimeout` устанавливает deadline контекста, но не прерывает горутину принудительно. Хендлеры, игнорирующие `ctx.Done()`, выполняются до конца.
- `Helmet: true` + Swagger UI могут конфликтовать по `Content-Security-Policy`.
- `EndpointsPathPrefix` не должен совпадать с `/status` или `AdminEndpointsPath`.
