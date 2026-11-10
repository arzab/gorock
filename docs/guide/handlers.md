# Как писать Handler

Обработчик — бизнес-логика запроса: что проверить, какие сервисы вызвать, что вернуть. Устроен как во Fiber, только без привязки к HTTP: одна и та же цепочка работает в HTTP, очереди и по расписанию. Зачем так — на странице [Архитектура](/architecture/#_2-handlers). Модуль — [rockhandlers](/gorock-kit/rockhandlers).

Код ниже — из примера [taskapi](https://github.com/arzab/gorock/tree/main/examples/taskapi).

```
internal/handlers/
├── tasks/
│   ├── create.go          параметры, ответ, обработчики, var Create
│   ├── create_test.go
│   └── get.go
└── middleware/
    └── log.go             общая обвязка всех цепочек
```

---

## 1. Параметры — всё, что пришло

```go
// internal/handlers/tasks/create.go
type CreateParams struct {
    Title string `json:"title"`
}

// Validate выполняется до всех обработчиков: сначала проверяем, потом используем.
func (p *CreateParams) Validate() error {
    if strings.TrimSpace(p.Title) == "" {
        return rockhandlers.Invalid("invalid task").WithFields(map[string]string{"title": "required"})
    }
    return nil
}
```

Параметры заполняет App: из тела запроса, заголовков, пути, сообщения. Теги говорят, откуда взять поле: `json` — тело, `header` — заголовок, `path` — часть пути, `query` — строка запроса, `form` — форма и файлы.

`Validate` обязателен. Обычная ошибка из него становится «неверными данными» с её текстом, а `rockhandlers.Invalid(...)` с полями проходит как есть.

## 2. Ответ — что вернуть

```go
type CreateResponse struct {
    Task *models.Task `json:"task"`
}

// Normalize выполняется после цепочки, если ошибок не было.
func (r *CreateResponse) Normalize() error { return nil }
```

Ответ создан заранее, обработчики его заполняют. App отправит его в формате своей технологии — у HTTP это JSON. `Normalize` — последний шанс привести ответ в порядок или отказаться отдавать его: ошибка из него приходит в App как `*rockhandlers.NormalizeError`.

## 3. Обработчики и цепочка

Обработчик получает всё аргументами, как в `net/http`:

```go
func create(c *rockhandlers.Ctx, params *CreateParams, response *CreateResponse) error {
    task := &models.Task{ID: uuid.NewString(), Title: strings.TrimSpace(params.Title), CreatedAt: time.Now().UTC()}
    if err := services.Repository().Insert(c, task); err != nil {
        return fmt.Errorf("insert task: %w", err) // непредвиденное: 500, текст скрыт
    }
    response.Task = task
    return nil
}

var Create = rockhandlers.Chain(create)
```

Цепочка — как во Fiber:
- **`return nil`** — цепочка закончилась без ошибок;
- **`return c.Next()`** — запустить следующий обработчик и вернуть его результат;
- **`return err`** — цепочка остановилась, ошибку получает App;
- **`c.Next()` после последнего обработчика** вернёт `rockhandlers.ErrHandlerNotFound`, поэтому последний возвращает `nil`.

Несколько обработчиков:

```go
func checkAuth(c *rockhandlers.Ctx, params *CreateParams, _ *CreateResponse) error {
    if params.Token == "" {
        return rockhandlers.Unauthorized("token required")
    }
    c.Set(userKey{}, params.Token) // значение для следующих обработчиков
    return c.Next()
}

var Create = rockhandlers.Chain(checkAuth, create)
```

- **`Chain` выводит типы параметров и ответа из обработчиков.** Обработчик с другими типами — ошибка компиляции.
- **Цепочку можно создать пустой** (`rockhandlers.Chain[*CreateParams, *CreateResponse]()`) — как заготовку: HTTP ответит `501 Not Implemented`.
- **`c` — это `context.Context` запроса.** Передавайте его в сервисы, и отмена запроса дойдёт до базы.
- **Значения между обработчиками** — `c.Set(key, v)` и `c.Get(key)`.

## 4. Ошибки — что случилось

Обработчик говорит, **что** случилось, а App решает, как это выглядит:

| Ошибка | Когда | HTTP | Очередь |
|---|---|---|---|
| `rockhandlers.Invalid(...)` | неверные данные | 400 | в недоставленные |
| `rockhandlers.Unauthorized(...)` | не авторизован | 401 | в недоставленные |
| `rockhandlers.Forbidden(...)` | нет прав | 403 | в недоставленные |
| `rockhandlers.NotFound(...)` | нет сущности | 404 | в недоставленные |
| `rockhandlers.Conflict(...)` | конфликт состояния | 409 | в недоставленные |
| `rockhandlers.Unavailable(...)` | временно недоступно | 503 | повторить позже |
| обычная ошибка, паника | непредвиденное | 500, текст скрыт | повторить |

Ошибки сервисов переводите в бизнес-ошибки там, где это решение:

```go
// internal/handlers/tasks/get.go
type GetParams struct {
    ID string `path:"id"`
}

func (p *GetParams) Validate() error { return nil }

type GetResponse struct {
    models.Task // поля задачи — тело ответа
}

func (r *GetResponse) Normalize() error { return nil }

var Get = rockhandlers.Chain(get)

func get(c *rockhandlers.Ctx, params *GetParams, response *GetResponse) error {
    err := services.Repository().GetByID(c, &response.Task, params.ID)
    if errors.Is(err, repository.ErrNotFound) {
        return rockhandlers.NotFound("task not found")
    }
    return err
}
```

---

## Middleware

Middleware — тот же обработчик, только с общими интерфейсами вместо конкретных структур, потому что он стоит перед любыми цепочками:

```go
// internal/handlers/middleware/log.go
func Log(c *rockhandlers.Ctx, _ rockhandlers.Params, _ rockhandlers.Response) error {
    err := c.Next()
    if err == nil {
        return nil
    }
    switch kind := rockhandlers.KindOf(err); kind {
    case rockhandlers.KindInternal, rockhandlers.KindUnavailable, rockhandlers.KindTimeout:
        rocklog.FromContext(c).Error("request failed", rocklog.Str("kind", kind.String()), rocklog.Err(err))
    default:
        rocklog.FromContext(c).Warn("request rejected", rocklog.Str("kind", kind.String()), rocklog.Err(err))
    }
    return err
}
```

Его вешают на App: `rockhttp.New(cfg, endpoints...).Use(middleware.Log)`. На одну цепочку — `tasks.Create.Use(rockhandlers.Timeout(2*time.Second))`. Middleware видит и ошибки разбора и проверки параметров. Клиенту они приходят без имён Go-типов: `invalid JSON body`, `invalid path` с полем в `fields`.

---

## Тестирование

Цепочка вызывается напрямую — без сервера, с подделкой сервиса ([как подменить](/guide/services#тестирование)):

```go
func TestCreate(t *testing.T) {
    fake := useFake(t)

    res, err := tasks.Create.Call(context.Background(), &tasks.CreateParams{Title: " write docs "})
    if err != nil {
        t.Fatal(err)
    }
    task := res.(*tasks.CreateResponse).Task
    if task.Title != "write docs" || fake.tasks[task.ID] == nil {
        t.Fatalf("result %+v", task)
    }
}
```

`Call` проходит всё то же, что запрос из App: `Validate`, обработчики, `Normalize`.

---

## Чего не делать

- **Не разбирать запрос в обработчике.** Всё входящее — поля параметров с тегами.
- **Не возвращать статус-коды и не писать ответ.** Только заполненный ответ или ошибку с видом.
- **Не вызывать `c.Next()` в последнем обработчике.** Цепочку завершает `return nil`.
- **Не делать гигантский обработчик.** Каждый — одно дело.

→ Дальше: [Как писать App](/guide/apps) · API: [rockhandlers](/gorock-kit/rockhandlers)
