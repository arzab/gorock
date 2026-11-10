# Как писать Handler

Обработчик — бизнес-логика запроса: что проверить, какие сервисы вызвать, что вернуть. Он получает заполненные параметры и возвращает результат или типизированную ошибку, а как к нему пришли и как будет выглядеть ответ, не знает. Зачем так — на странице [Архитектура](/architecture/#_2-handlers). Механизм — модуль [rockhandlers](/gorock-kit/rockhandlers).

```
internal/handlers/
├── orders/
│   ├── create.go          параметры, результат, шаги, var Create
│   ├── create_test.go
│   └── get.go
├── steps/                 общие шаги: авторизация, тенант…
│   └── auth.go
└── middleware/            общая обвязка всех обработчиков
    └── log.go
```

Одно действие — один файл. Большое действие можно разложить в папку.

---

## 1. Параметры — всё, что пришло

```go
// internal/handlers/orders/create.go
type CreateParams struct {
    steps.BearerAuth        // токен — для шага авторизации
    Item             string `json:"item"`
    Qty              int    `json:"qty"`
}
```

В параметрах всё, что нужно обработчику:
- данные от клиента — тело запроса;
- метаданные запроса — заголовки, идентификатор из пути, параметры строки запроса.

Теги говорят App, откуда взять поле: `json` — тело, `header` — заголовок, `path` — часть пути, `query` — строка запроса, `form` — форма и файлы. Обработчику это безразлично: к нему структура приходит уже заполненной.

## 2. Проверка — до всех шагов

```go
func (p *CreateParams) Validate() error {
    fields := map[string]string{}
    if strings.TrimSpace(p.Item) == "" {
        fields["item"] = "required"
    }
    if p.Qty <= 0 {
        fields["qty"] = "must be > 0"
    }
    if len(fields) > 0 {
        return rockhandlers.Invalid("invalid order").WithFields(fields)
    }
    return nil
}
```

Если у параметров есть метод `Validate`, он выполняется сразу после заполнения, до первого шага. Сначала проверяем, потом используем. Ошибки по полям (`WithFields`) App отдаст клиенту — у HTTP это поле `fields` в JSON.

## 3. Результат — что вернуть

```go
type CreateResult struct {
    Order *models.Order `json:"order"`
}
```

Результат создан заранее, шаги его заполняют. App отправит его клиенту в формате своей технологии — у HTTP это JSON. Если отправлять нечего, результат — пустая структура: HTTP ответит `204`.

## 4. Шаги — что сделать

```go
type CreateCtx = rockhandlers.Ctx[*CreateParams, *CreateResult] // короткое имя для сигнатур

// Create: проверить токен → создать заказ.
var Create = rockhandlers.Chain("orders.create", steps.Bearer, create)

func create(c *CreateCtx) error {
    order := &models.Order{
        ID:    uuid.NewString(),
        Owner: steps.UserKey.MustGet(c), // положил шаг Bearer
        Item:  c.Params.Item,
        Qty:   c.Params.Qty,
    }
    if err := services.Orders().Insert(c, order); err != nil {
        return fmt.Errorf("insert order: %w", err)
    }
    c.Result.Order = order
    return nil
}
```

- **`rockhandlers.Chain(имя, шаги...)`** собирает обработчик из шагов. Имя попадает в логи и метрики.
- **Шаг вызывает `c.Next()`,** чтобы передать управление следующему. Последний шаг `Next` не вызывает. Шаг, который не вызвал `Next`, завершает цепочку — например, при ответе из кэша.
- **`c` — это `context.Context` запроса.** Передавайте его в сервисы, и отмена запроса дойдёт до базы.
- **Шаги передают друг другу данные через типизированные значения:** `UserKey.Set(c, user)` в одном шаге, `UserKey.MustGet(c)` в следующем.

## 5. Ошибки — что случилось

Обработчик говорит, **что** случилось, а App решает, как это выглядит для клиента:

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
// internal/handlers/orders/get.go
var Get = rockhandlers.Chain("orders.get", get)

func get(c *rockhandlers.Ctx[*GetParams, *models.Order]) error {
    err := services.Orders().GetByID(c, c.Result, c.Params.ID)
    if errors.Is(err, ordersvc.ErrNotFound) {
        return rockhandlers.NotFound("order not found")
    }
    return err // непредвиденное — текст увидит лог, а не клиент
}
```

---

## Общие шаги

Проверка, нужная многим обработчикам, пишется один раз. Если шагу нужны данные запроса, он требует их через интерфейс, а параметры обработчика выполняют требование встраиванием:

```go
// internal/handlers/steps/auth.go
type BearerAuth struct {
    Authorization string `header:"Authorization" json:"-"`
}

func (a *BearerAuth) BearerToken() string { return strings.TrimPrefix(a.Authorization, "Bearer ") }

type TokenParams interface{ BearerToken() string }

var UserKey = rockhandlers.NewLocal[string]("user")

func Bearer[P TokenParams, R any](c *rockhandlers.Ctx[P, R]) error {
    user, ok := verify(c.Params.BearerToken())
    if !ok {
        return rockhandlers.Unauthorized("invalid token")
    }
    UserKey.Set(c, user)
    return c.Next()
}
```

Шаг `Bearer` ставится в любую цепочку, параметры которой встраивают `BearerAuth`. Если забыть встроить, код не соберётся. Приведений типов нет.

**Группы** — обычные функции, которые добавляют общие шаги:

```go
func Secured[P steps.TokenParams, R any](name string, s ...rockhandlers.Handler[P, R]) rockhandlers.Route {
    return rockhandlers.Chain(name, rockhandlers.Prepend(s, steps.Bearer[P, R])...)
}
```

## Общая обвязка (middleware)

То, что нужно всем обработчикам и не смотрит в параметры — логирование, метрики, ограничение времени, — пишется как middleware и вешается на App один раз:

```go
// internal/handlers/middleware/log.go
func Log(c rockhandlers.Context) error {
    err := c.Next()
    if err == nil {
        return nil
    }
    switch kind := rockhandlers.KindOf(err); kind {
    case rockhandlers.KindInternal, rockhandlers.KindUnavailable, rockhandlers.KindTimeout:
        rocklog.Error("request failed", rocklog.Str("handler", c.Route()), rocklog.Str("kind", kind.String()), rocklog.Err(err))
    default:
        rocklog.Warn("request rejected", rocklog.Str("handler", c.Route()), rocklog.Str("kind", kind.String()), rocklog.Err(err))
    }
    return err
}
```

Ошибки разбора запроса тоже проходят через middleware. Ограничение времени — `rockhandlers.Timeout(5*time.Second)`. Для одного обработчика: `Create.With(rockhandlers.Timeout(2*time.Second))`.

---

## Тестирование

Обработчик вызывается напрямую — без сервера, с подделкой сервиса в каталоге ([как подменить](/guide/services#тестирование)):

```go
res, err := rockhandlers.Run[*orders.CreateResult](context.Background(), orders.Create, &orders.CreateParams{
    BearerAuth: steps.BearerAuth{Authorization: "Bearer ann"},
    Item:       "tea",
    Qty:        2,
})
```

`Run` проходит всю цепочку: проверку параметров, шаги, middleware.

---

## Чего не делать

- **Не разбирать запрос в обработчике.** Всё входящее — поля параметров с тегами.
- **Не возвращать статус-коды и не писать ответ.** Только результат или ошибку с видом.
- **Не приводить параметры к интерфейсам в шагах.** Требования — через ограничения типа и встраивание.
- **Не делать гигантский шаг.** Каждый шаг — одно дело.

→ Дальше: [Как писать App](/guide/apps) · API: [rockhandlers](/gorock-kit/rockhandlers)
