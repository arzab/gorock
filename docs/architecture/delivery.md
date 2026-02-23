# Delivery

**Realm** · `internal/delivery/`

Здесь вы готовите бизнес-логику. В delivery вы создаёте сущности которые описывают правила входящего запроса или события: что делать, в каком порядке, что вернуть.

```
internal/delivery/
├── endpoints/
│   ├── endpoints.go         — список всех HTTP endpoints
│   └── create_user/         — один endpoint = одна папка
│       ├── endpoint.go      — метод, путь, цепочка handlers
│       ├── handlers.go      — цепочка функций, каждая делает один шаг
│       ├── params.go        — Params: структура + InitParams() + GetParams()
│       └── response.go      — Response: структура + InitResponse() + GetResponse()
├── consumers/
│   └── new_task.go          — handler-функция для consumer
└── events/
    └── user_created.go      — handler для внутреннего eventbus
```

---

## Что такое delivery

Delivery — это описание того что происходит при входящем запросе или событии. Каждая delivery-сущность описывает две вещи:

**Транспортные правила** — для HTTP: метод (`POST`), путь (`/users`), список обработчиков. Для consumer: функция-handler. Delivery само-описывает как его зарегистрировать — App просто получает список и регистрирует.

**Бизнес-поведение** — handlers это список функций, каждая делает один шаг. Вызвали сервис, обработали ошибку, сформировали ответ.

Формат delivery-сущностей определяется конкретным App — это решение разработчика.

---

## Что использует delivery

- `pkg/services` — для вызова бизнес-операций
- `internal/models` — для работы со структурами данных
- `pkg/infra/logs` — для логирования

Delivery не знает о `cmd/apps` — зависимость односторонняя: App подключает delivery, а не наоборот.

---

## Форматы delivery-сущностей

### Функция — для consumer

Простейший случай — delivery возвращает функцию нужного типа. App вызывает её и сам решает логику ack/nack по результату.

```go
// internal/models/entities/handler.go
type MessageHandler func(msg *pubsub.Message) (err error, shouldNack bool)

// internal/delivery/consumers/new_task.go
func NewTask() entities.MessageHandler {
    return func(msg *pubsub.Message) (err error, shouldNack bool) {
        task := &models.Task{}
        if err = json.Unmarshal(msg.Data, task); err != nil {
            return fmt.Errorf("unmarshal task: %w", err), false
        }

        if err = services.FileStorage().DownloadFile(task.BucketName, task.ObjectName, archivePath); err != nil {
            return fmt.Errorf("download file: %w", err), false
        }

        if err = services.RunManager().ExecChunk(tempDir); err != nil {
            return fmt.Errorf("exec script: %w", err), false
        }

        return nil, false
    }
}
```

App регистрирует handler при `Init`:

```go
// cmd/apps/rabbit_consumer/app.go
func (a *app) Init() error {
    a.handler = consumers.NewTask()
    return nil
}
```

### Самоописывающаяся структура — для HTTP

Endpoint сам описывает себя: метод, путь, цепочка handlers. App регистрирует маршруты не зная деталей каждого endpoint.

```go
// internal/delivery/endpoints/endpoints.go
func HttpEndpoints() []endpoints.FiberEndpoint {
    return []endpoints.FiberEndpoint{
        exec.Endpoint(),
    }
}

// internal/delivery/endpoints/exec/endpoint.go
func Endpoint() endpoints.FiberEndpoint {
    return endpoints.BuildFiberEndpoint("post", "/exec", handlers())
}
```

### Цепочка handlers

Бизнес-логика разбивается на несколько функций — каждая делает один шаг. Данные передаются через shared-контекст (например `fiber.Ctx.Locals`).

```go
// internal/delivery/endpoints/exec/handlers.go
func handlers() []fiber.Handler {
    return []fiber.Handler{
        InitParams(),     // парсинг и валидация входных данных
        InitResponse(),   // инициализация объекта ответа
        execScript(),     // бизнес-логика
        returnResponse(), // сериализация и отправка ответа
    }
}

func execScript() fiber.Handler {
    return func(ctx *fiber.Ctx) error {
        params, err := GetParams(ctx)
        if err != nil {
            return err
        }
        // вызов services...
        return ctx.Next()
    }
}
```

`params.go` и `response.go` хранят типизированные объекты в контексте:

```go
// params.go
type Params struct {
    RunId string `json:"run_id"`
}

func InitParams() fiber.Handler { return params.DefaultHandler[Params]() }
func GetParams(ctx *fiber.Ctx) (*Params, error) { ... }

// response.go
type Response struct {
    Status string `json:"status"`
}

func InitResponse() fiber.Handler { return http.HandlerInitInCtx[Response]("response") }
func GetResponse(ctx *fiber.Ctx) (*Response, error) { ... }
```

---

## Версионирование

При необходимости поддерживать несколько версий API — через папки:

```
internal/delivery/endpoints/
├── endpoints.go
├── v1/
│   └── exec/
│       └── ...
└── v2/
    └── exec/
        └── ...
```

---

## Правила

- **Delivery описывает себя.** Endpoint знает path, method, handlers. App просто регистрирует — не вникает в логику.
- **Никаких гигантских handlers.** Каждая функция делает один шаг — парсинг, валидация, вызов сервиса, формирование ответа.
- **Ошибки возвращаются наверх.** Delivery возвращает ошибку с контекстом, App маппит её в транспортный ответ (HTTP-код, ack/nack).
- **Форма delivery — решение разработчика.** Архитектура описывает слой, но не диктует технологию.
- **Зависимость односторонняя.** Delivery знает о services и models, но не о apps.
