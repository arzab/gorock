# Delivery

**Realm** · `internal/delivery/`

Здесь вы пишете бизнес-логику. Delivery-сущности описывают правила входящего запроса или события: что делать, в каком порядке, что вернуть.

```
internal/delivery/
├── endpoints/
│   ├── endpoints.go         — список всех HTTP endpoints
│   ├── create_task/         — один endpoint = одна папка
│   │   ├── endpoint.go      — метод, путь, цепочка handlers
│   │   ├── handlers.go      — цепочка функций, каждая делает один шаг
│   │   ├── params.go        — Params + Validate + InitParams/GetParams
│   │   └── response.go      — Response + InitResponse/GetResponse
│   └── get_task/
│       └── endpoint.go      — маленький endpoint может жить в одном файле
├── events/
│   └── events.go            — топики и обработчики событий шины
└── jobs/
    └── jobs.go              — задачи по расписанию
```

---

## Что такое delivery

Каждая delivery-сущность описывает две вещи:

**Транспортные правила** — для HTTP: метод (`POST`), путь (`/tasks`), список обработчиков. Для событий — топик и обработчик. Для расписания — интервал и функция. Delivery сам описывает, как его зарегистрировать, — App просто получает список и регистрирует.

**Бизнес-поведение** — handlers это список функций, каждая делает один шаг: распарсили, вызвали сервис, обработали ошибку, сформировали ответ.

Формат delivery-сущностей определяется App, который их принимает. Для готовых App из gorock-kit это `rockfiber.FiberEndpoint`, `rockbus.Subscription`, `rockcron.Job`.

---

## Что использует delivery

- `pkg/services` — для вызова операций над технологиями
- `internal/models` — для работы со структурами данных
- [rocklog](/gorock-kit/rocklog) — для логирования
- хелперы App: `rockfiber.NewError`, `rockfiber.DefaultHandler`, `rockbus.PublishAsync`

Delivery не знает о `cmd/apps` — зависимость односторонняя: App подключает delivery, а не наоборот.

---

## HTTP endpoints

### Список

```go
// internal/delivery/endpoints/endpoints.go
package endpoints

// All — все HTTP endpoints. HTTP App регистрирует их как есть.
func All() []rockfiber.FiberEndpoint {
    return []rockfiber.FiberEndpoint{
        create_task.Endpoint(),
        get_task.Endpoint(),
    }
}
```

### Endpoint сам себя описывает

```go
// internal/delivery/endpoints/create_task/endpoint.go
package create_task

func Endpoint() rockfiber.FiberEndpoint {
    return rockfiber.POST("/tasks", handlers()...)
}
```

`rockfiber.GET`, `POST`, `PUT`, `PATCH`, `DELETE` — метод, путь и цепочка handlers. Общий префикс (`/api/v1`) задаётся в конфиге App, а не в каждом endpoint.

### Цепочка handlers

Бизнес-логика разбивается на несколько функций — каждая делает один шаг и вызывает `c.Next()`. Данные передаются через `fiber.Ctx.Locals`.

```go
// internal/delivery/endpoints/create_task/handlers.go
func handlers() []fiber.Handler {
    return []fiber.Handler{
        InitParams(),   // парсинг и валидация входных данных
        InitResponse(), // объект ответа
        createTask(),   // бизнес-логика
        sendResponse(), // сериализация
    }
}

func createTask() fiber.Handler {
    return func(c *fiber.Ctx) error {
        params, err := GetParams(c)
        if err != nil {
            return err
        }
        resp, err := GetResponse(c)
        if err != nil {
            return err
        }

        // Логгер запроса с trace_id — уедет вместе с ctx в сервисы и в событие.
        ctx := rocklog.WithContext(c.UserContext(),
            rocklog.Str("trace_id", rockfiber.TraceIdFromContext(c.UserContext())))

        task := &models.Task{ID: uuid.NewString(), Title: params.Title}
        if err := services.Repository().Insert(ctx, task); err != nil {
            return fmt.Errorf("insert task: %w", err) // обычная ошибка → 500
        }

        rockbus.PublishAsync(ctx, rockbus.Event{Topic: events.TaskCreated, Payload: *task})
        resp.Task = task
        return c.Next()
    }
}

func sendResponse() fiber.Handler {
    return func(c *fiber.Ctx) error {
        resp, err := GetResponse(c)
        if err != nil {
            return err
        }
        return c.Status(http.StatusCreated).JSON(resp)
    }
}
```

### Params

`rockfiber.DefaultHandler` разбирает query, тело, заголовки и параметры пути в одну структуру по тегам (`query`, `json`, `reqHeader`, `params`), вызывает `Validate` и кладёт результат в `ctx.Locals("params")`.

```go
// internal/delivery/endpoints/create_task/params.go
type Params struct {
    Title string `json:"title"`
}

func (p *Params) Validate(*fiber.Ctx) error {
    if strings.TrimSpace(p.Title) == "" {
        return rockfiber.NewError(http.StatusBadRequest, "title is required")
    }
    return nil
}

func InitParams() fiber.Handler { return rockfiber.DefaultHandler[Params]() }

func GetParams(c *fiber.Ctx) (*Params, error) {
    return rockfiber.GetFromContext[Params](c, "params")
}
```

### Response

```go
// internal/delivery/endpoints/create_task/response.go
type Response struct {
    Task *models.Task `json:"task"`
}

const responseKey = "response"

func InitResponse() fiber.Handler { return rockfiber.HandlerInitInCtx[Response](responseKey) }

func GetResponse(c *fiber.Ctx) (*Response, error) {
    return rockfiber.GetFromContext[Response](c, responseKey)
}
```

### Ошибки — это бизнес-решения

Что увидит клиент — решает delivery. `rockfiber.NewError` задаёт код и сообщение; любая другая ошибка превращается в 500, а её текст скрывается от клиента, если в конфиге задан `mask_internal_server_error_message`.

```go
// internal/delivery/endpoints/get_task/endpoint.go
type Params struct {
    ID string `params:"id"`
}

func (p *Params) Validate(*fiber.Ctx) error {
    if p.ID == "" {
        return rockfiber.NewError(http.StatusBadRequest, "id is required")
    }
    return nil
}

func Endpoint() rockfiber.FiberEndpoint {
    return rockfiber.GET("/tasks/:id", rockfiber.DefaultHandler[Params](), getTask())
}

func getTask() fiber.Handler {
    return func(c *fiber.Ctx) error {
        params, err := rockfiber.GetFromContext[Params](c, "params")
        if err != nil {
            return err
        }

        task := &models.Task{}
        err = services.Repository().GetByID(c.UserContext(), task, params.ID)
        if errors.Is(err, repository.ErrNotFound) {
            return rockfiber.NewError(http.StatusNotFound, "task not found") // 404
        }
        if err != nil {
            return err // 500
        }
        return c.JSON(task)
    }
}
```

---

## События

Обработчики событий шины [rockbus](/gorock-kit/rockbus) — такие же delivery-сущности: топик и функция.

```go
// internal/delivery/events/events.go
package events

const TaskCreated rockbus.Topic = "task.created"

// Subscriptions — все обработчики событий. Bus App регистрирует их.
func Subscriptions() []rockbus.Subscription {
    return []rockbus.Subscription{
        rockbus.On(TaskCreated, onTaskCreated),
    }
}

func onTaskCreated(ctx context.Context) error {
    task, err := rockbus.Payload[models.Task](ctx)
    if err != nil {
        return err
    }
    rocklog.FromContext(ctx).Info("task created", rocklog.Str("id", task.ID))
    return nil
}
```

Логгер с `trace_id`, положенный в `ctx` в HTTP-обработчике, доходит сюда: в логе события будет тот же trace_id, что и у запроса.

---

## Задачи по расписанию

```go
// internal/delivery/jobs/jobs.go
package jobs

// All — все задачи. Cron App регистрирует их.
func All() []rockcron.Job {
    return []rockcron.Job{
        rockcron.Every("ping-db", time.Minute, pingDB),
        rockcron.Cron("daily-report", "0 3 * * *", buildReport, sendReport),
    }
}

func pingDB(ctx context.Context) error {
    return services.Repository().Ping(ctx)
}
```

Несколько функций в задаче выполняются по очереди, как цепочка handlers: первая ошибка останавливает цепочку и уходит в обработчик ошибок App.

---

## Версионирование

Несколько версий API — через папки:

```
internal/delivery/endpoints/
├── endpoints.go
├── v1/
│   └── create_task/
└── v2/
    └── create_task/
```

---

## Правила

- **Delivery описывает себя.** Endpoint знает path, method, handlers. App просто регистрирует — не вникает в логику.
- **Никаких гигантских handlers.** Каждая функция делает один шаг — парсинг, валидация, вызов сервиса, ответ.
- **Ошибки — бизнес-решения.** `rockfiber.NewError` — когда клиент должен получить конкретный код; обычная ошибка с контекстом (`fmt.Errorf("...: %w", err)`) — для непредвиденного.
- **`c.UserContext()` — в сервисы.** В нём дедлайн запроса и trace_id.
- **Зависимость односторонняя.** Delivery знает о services и models, но не о apps.
