# Libs

**Toolkit** · `pkg/libs/`

Переиспользуемые строительные блоки без глобального состояния. Не singletons, не требуют Init — импортируешь и используешь.

```
pkg/libs/
└── go-kit/
    ├── configs/          — загрузка конфигурационных файлов
    ├── http/
    │   ├── server/       — Fiber HTTP-сервер с батареей из коробки
    │   ├── endpoints/    — интерфейс и конструктор для самоописывающихся endpoints
    │   ├── params/       — generic парсинг и валидация параметров запроса
    │   ├── responses/    — стандартная структура ошибок
    │   └── utils.go      — типизированная работа с fiber.Ctx.Locals
    └── logs/             — базовая обёртка над logrus
```

---

## Что такое libs

Внутренние библиотеки — переиспользуемые строительные блоки на которых строятся слои проекта. В отличие от `pkg/services`, libs не являются singleton'ами и не хранят глобального состояния. Это просто библиотечный код: берёшь, используешь, передаёшь как аргумент.

---

## Отличие от services и infra

| | `pkg/libs` | `pkg/services` | `pkg/infra` |
|---|---|---|---|
| Состояние | Нет | Singleton | Singleton |
| Init/Stop | Нет | Есть | Есть |
| Использование | Импортируешь и вызываешь | Через геттер | Через функции пакета |
| Назначение | Строительные блоки | Обёртки над внешними системами | Логирование, трейсинг, eventbus |

---

## go-kit

Основная библиотека — `pkg/libs/go-kit`. Предоставляет базовые реализации для HTTP-сервера, работы с параметрами запросов, загрузки конфигов и логирования. В перспективе может быть вынесена как отдельный внешний пакет.

### configs

`InitFromFile[T](path)` — загружает JSON-файл в структуру `T`:
1. Читает файл
2. Прогоняет через `os.ExpandEnv()` — подставляет env-переменные
3. Анмаршаллит в структуру
4. Рекурсивно проверяет поля на пустоту с учётом тегов `config:"ignore"` / `config:"omitempty"`

```go
mainConfigs, err := configs.InitFromFile[Configs](configsPath)
```

### http/server

Fiber-сервер с батареей из коробки. Принимает список `FiberEndpoint` и регистрирует маршруты сам.

```go
type Service interface {
    Init() error
    Exec(httpEndpoints []endpoints.FiberEndpoint) error
    Shutdown(shutdownFunc func() []error) []error
}
```

Из коробки: recover, CORS, pprof, swagger, trace-id middleware, логирование входящих запросов, admin-эндпоинты `/metrics`, `/status`.

### http/endpoints

Интерфейс самоописывающегося endpoint и его конструктор — используется в `internal/delivery/endpoints`.

```go
type FiberEndpoint interface {
    GetPath() string
    GetMethod() string
    GetHandlers() []fiber.Handler
}

func BuildFiberEndpoint(method, path string, handlers []fiber.Handler) FiberEndpoint
```

### http/params

Generic обработчик для парсинга и валидации входных параметров. Парсит query, body, headers, url-параметры в одну структуру по тегам.

```go
// Params должна реализовывать Validate(ctx) error
func DefaultHandler[T any, pointer Service[T]](key ...string) fiber.Handler
```

### http/responses

Стандартная структура ошибки, которая реализует `error` и возвращается напрямую из fiber.Handler:

```go
type ErrorResponse struct {
    Code    int    `json:"code"`
    Status  string `json:"status"`
    Message string `json:"message"`
    Source  string `json:"source"`
    Action  string `json:"action"`
}

func NewError(statusCode int, message string, sourceAction ...string) *ErrorResponse
```

### http/utils

Типизированные хелперы для работы с `fiber.Ctx.Locals` — кладут и достают объекты с проверкой типа:

```go
func GetFromContext[T any](ctx *fiber.Ctx, key string) (*T, error)
func HandlerInitInCtx[T any](key string) fiber.Handler
```

### logs

Базовая обёртка над logrus. Служит основой для `pkg/infra/logs` — infra импортирует go-kit/logs и расширяет его под нужды проекта.

```go
func Init(configs Configs)
func Infof(format string, args ...interface{})
func Errorf(format string, args ...interface{})
func Warnf(format string, args ...interface{})
func Fatalf(format string, args ...interface{})
func InfoWithFields(fields map[string]interface{}, message string, args ...interface{})
func ErrorWithFields(fields map[string]interface{}, message string, args ...interface{})
```

---

## Правила

- **Libs — не singletons.** Никакого глобального состояния, никакого Init на уровне слоя.
- **`pkg/libs`** — доступен всем слоям, включая services и infra.
- **Независимость от проекта.** Libs не знают о `internal/models` и бизнес-логике — это переносимый инструментарий.
