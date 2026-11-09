# Delivery

**Realm** · `internal/delivery/`

Здесь вы пишете бизнес-логику входящих запросов и событий: что проверить, что сделать, что вернуть.

```
internal/delivery/
├── steps/                   — переиспользуемые шаги: авторизация, тенант…
│   └── auth.go
├── endpoints/
│   ├── endpoints.go         — список всех HTTP endpoints
│   ├── create_task/         — один маршрут = одна папка
│   │   ├── route.go         — Route: цепочка шагов
│   │   ├── params.go        — Params + Validate
│   │   ├── result.go        — Result
│   │   └── steps.go         — шаги маршрута
│   └── get_task/
│       └── route.go         — маленький маршрут может жить в одном файле
├── events/                  — обработчики событий шины
└── jobs/                    — задачи по расписанию
```

---

## Маршрут — это цепочка шагов

Запрос обрабатывается цепочкой [rockhandlers](/gorock-kit/rockhandlers): каждый шаг делает одно дело и вызывает `c.Next()`. Как во Fiber, только без привязки к технологии: одна и та же цепочка работает и в HTTP, и в очереди, и в CLI.

```
транспорт (App)                 rockhandlers                        Realm
заполняет Params      ──▶  middleware → шаг 1 → шаг 2 … ──▶  сервисы
переводит Result      ◀──  Result или ошибка с видом
```

Маршрут ничего не знает про HTTP: ни путь, ни метод, ни статус. Где он «висит» — решает `internal/delivery/endpoints/endpoints.go` и App.

---

## Params, Result и шаги

```go
// internal/delivery/endpoints/create_task/params.go
type Params struct {
    steps.BearerAuth          // встроенная модель шага авторизации (см. ниже)
    Title string `json:"title"`
}

// Validate выполняется сразу после разбора запроса: сначала проверка, потом использование.
func (p *Params) Validate() error {
    if strings.TrimSpace(p.Title) == "" {
        return rockhandlers.Invalid("invalid task").WithFields(map[string]string{"title": "required"})
    }
    return nil
}
```

```go
// internal/delivery/endpoints/create_task/result.go
type Result struct {
    Task *models.Task `json:"task"`
}

type Ctx = rockhandlers.Ctx[*Params, *Result] // короткое имя для сигнатур шагов
```

```go
// internal/delivery/endpoints/create_task/steps.go
func create(c *Ctx) error {
    task := &models.Task{ID: uuid.NewString(), Title: c.Params.Title, Owner: steps.UserKey.MustGet(c)}
    if err := repository.Get().Insert(c, task); err != nil { // c — это context.Context
        return fmt.Errorf("insert task: %w", err) // обычная ошибка → Internal, текст скрыт
    }
    c.Result.Task = task
    return nil
}
```

```go
// internal/delivery/endpoints/create_task/route.go
var Route = rockhandlers.Chain("tasks.create", steps.Bearer, create)
```

- `Params` заполняет транспорт (у HTTP — по тегам `path`, `query`, `header`, `form` и JSON-телу), `Result` создан заранее — шаги его заполняют.
- Шаг, который не вызвал `c.Next()`, завершает цепочку — так работает, например, ответ из кэша.
- Данные между шагами — через типизированные Locals: `steps.UserKey.Set(c, user)` / `steps.UserKey.MustGet(c)`.
- `c` — это `context.Context`: передавайте его в сервисы, тогда отмена запроса и таймауты доходят до БД.

---

## Переиспользуемые шаги

Шаг, которому нужны данные запроса, объявляет требование интерфейсом — компилятор проверяет, что модель его выполняет. Модели выполняют требование **встраиванием**:

```go
// internal/delivery/steps/auth.go
type BearerAuth struct {
    Authorization string `header:"Authorization" json:"-"`
}

func (a *BearerAuth) BearerToken() string { return strings.TrimPrefix(a.Authorization, "Bearer ") }

type TokenParams interface{ BearerToken() string }

var UserKey = rockhandlers.NewLocal[string]("user")

func Bearer[P TokenParams, R any](c *rockhandlers.Ctx[P, R]) error {
    user, err := auth.Get().Verify(c, c.Params.BearerToken())
    if err != nil {
        return rockhandlers.Unauthorized("invalid token")
    }
    UserKey.Set(c, user)
    return c.Next()
}
```

Забыли встроить `BearerAuth` в `Params` маршрута с `steps.Bearer` — код не соберётся. Приведений типов нет.

**Группы** — обычные функции, которые добавляют общие шаги и middleware:

```go
func Secured[P steps.TokenParams, R any](name string, s ...rockhandlers.Handler[P, R]) rockhandlers.Route {
    return rockhandlers.Chain(name, rockhandlers.Prepend(s, steps.Bearer[P, R])...)
}

var Route = Secured("tasks.create", create)
```

---

## Ошибки — это бизнес-решения

Что увидит клиент — решает delivery, но в терминах бизнеса, а не протокола. Транспорт сам переводит вид ошибки: в HTTP — в статус, в очереди — в ack/nack.

| Ошибка | Когда | HTTP ([rockhttp](/catalog/rockhttp)) |
|---|---|---|
| `rockhandlers.Invalid(...)` (+ `.WithFields`) | плохой ввод | 400 |
| `Unauthorized` / `Forbidden` | нет входа / нет прав | 401 / 403 |
| `NotFound` | нет сущности | 404 |
| `Conflict` | конфликт состояния | 409 |
| `Unavailable` | временно недоступно | 503 |
| обычная `error`, паника | непредвиденное | 500, текст скрыт |
| отмена / дедлайн ctx | клиент ушёл / не успели | 499 / 504 |

```go
// internal/delivery/endpoints/get_task/route.go
type Params struct {
    ID string `path:"id"`
}

var Route = rockhandlers.Chain("tasks.get", func(c *rockhandlers.Ctx[*Params, *models.Task]) error {
    err := repository.Get().GetByID(c, c.Result, c.Params.ID)
    if errors.Is(err, repository.ErrNotFound) {
        return rockhandlers.NotFound("task not found")
    }
    return err
})
```

---

## Middleware

Сквозная логика, которой не нужны данные запроса — логирование, метрики, таймауты, — пишется как middleware и вешается на маршрут (`route.With(...)`) или на весь App (`App.Use(...)`):

```go
func Log(c rockhandlers.Context) error {
    err := c.Next()
    if err != nil {
        rocklog.FromContext(c).Error("request failed", rocklog.Str("route", c.Route()), rocklog.Err(err))
    }
    return err
}
```

Ошибки разбора запроса тоже проходят через middleware: разбор — часть цепочки.

---

## Список endpoints

```go
// internal/delivery/endpoints/endpoints.go
package endpoints

func All() []rockhttp.Endpoint {
    return []rockhttp.Endpoint{
        {Method: "POST", Path: "/tasks", Handler: create_task.Route, Status: http.StatusCreated},
        {Method: "GET", Path: "/tasks/{id}", Handler: get_task.Route},
    }
}
```

Метод, путь и статус успеха — свойства HTTP, поэтому они здесь, а не в маршруте. Общий префикс (`/api/v1`) задаётся в конфиге App.

---

## Тестирование

Маршрут тестируется без HTTP:

```go
res, err := rockhandlers.Run[*create_task.Result](ctx, create_task.Route,
    &create_task.Params{BearerAuth: steps.BearerAuth{Authorization: "Bearer t"}, Title: "write docs"})
```

Сервисы подменяются через `rockservices.SetDefault` → [Services](/architecture/services).

---

## События и задачи по расписанию

Шина событий [rockbus](/catalog/rockbus) и планировщик [rockcron](/catalog/rockcron) из каталога пока используют свои форматы обработчиков (`rockbus.Subscription`, `rockcron.Job`), а не маршруты rockhandlers — их перевод запланирован. Описываются так же в `internal/delivery/events` и `internal/delivery/jobs`, подробности — на страницах модулей.

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

- **Маршрут не знает транспорт.** Путь, метод, статус — в списке endpoints; заголовки — теги в Params.
- **Никаких гигантских шагов.** Каждый шаг делает одно: проверить доступ, загрузить, изменить, заполнить результат.
- **Валидация — в `Params.Validate`**, она выполняется раньше всех шагов.
- **Ошибки — виды rockhandlers**, а не коды протокола. Обычная ошибка с контекстом (`fmt.Errorf("...: %w", err)`) — для непредвиденного.
- **`c` — в сервисы.** В нём отмена и дедлайн запроса.
- **Зависимость односторонняя.** Delivery знает о services и models, но не о apps.
