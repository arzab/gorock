# rockhandlers

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockhandlers/README_RU.md — правьте README в репозитории модуля. -->

::: tip Место в GOROCK · Handlers
Обработчики в `internal/handlers`: цепочки в стиле Fiber — `func(c, params, response) error`, `c.Next()`, `Validate`/`Normalize`, ошибки с видом, middleware. App заполняет параметры и переводит итог на свой язык. → [Как писать Handler](/guide/handlers)
:::

```sh
go get github.com/arzab/gorock-kit/rockhandlers@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-kit/tree/main/rockhandlers)

Слой обработчиков GOROCK-проекта: бизнес-логика как цепочка обработчиков — как во Fiber, только без привязки к HTTP. Одну и ту же цепочку запускает HTTP-сервер, consumer очереди или cron. Только стандартная библиотека.

## Обработчик

Всё нужное приходит аргументами, как в `net/http`:

```go
type Handler[P Params, R Response] func(c *Ctx, params P, response R) error
```

- `Params` — то, что пришло: тело, заголовки, путь. Заполняет App. Обязан уметь `Validate() error`.
- `Response` — то, что вернём: заполняют обработчики. Обязан уметь `Normalize() error`.
- `Ctx` — контекст запроса (`context.Context`) с `Next()` и значениями `Set`/`Get`.

```go
// internal/handlers/tasks/create.go
type CreateParams struct {
    Token string `header:"Authorization"`
    Title string `json:"title"`
}

func (p *CreateParams) Validate() error {
    if strings.TrimSpace(p.Title) == "" {
        return rockhandlers.Invalid("invalid task").WithFields(map[string]string{"title": "required"})
    }
    return nil
}

type CreateResponse struct {
    Task *models.Task `json:"task"`
}

func (r *CreateResponse) Normalize() error { return nil }

func checkAuth(c *rockhandlers.Ctx, params *CreateParams, _ *CreateResponse) error {
    if params.Token == "" {
        return rockhandlers.Unauthorized("token required")
    }
    return c.Next() // дальше
}

func create(c *rockhandlers.Ctx, params *CreateParams, response *CreateResponse) error {
    task := &models.Task{ID: uuid.NewString(), Title: params.Title}
    if err := services.Repository().Insert(c, task); err != nil {
        return err
    }
    response.Task = task
    return nil // готово
}

var Create = rockhandlers.Chain(checkAuth, create)
```

`Chain` выводит типы параметров и ответа из обработчиков. Обработчик с другими типами — ошибка компиляции.

## Цепочка — как во Fiber

- `return nil` — цепочка закончилась без ошибок.
- `return c.Next()` — запустить следующий обработчик и вернуть его результат.
- `return err` — цепочка остановилась, ошибку получает App.
- `c.Next()`, когда обработчиков больше нет, возвращает `ErrHandlerNotFound`. Поэтому последний обработчик возвращает `nil`.

Цепочку можно создать пустой — заготовкой. Тогда запуск вернёт `ErrHandlerNotFound`, а HTTP-сервер ответит `501 Not Implemented`.

## Порядок запуска

App вызывает `pipeline.Run(ctx, fill)`. Порядок:

1. **middleware;**
2. **`fill`** — App раскладывает запрос в параметры;
3. **`params.Validate()`;**
4. **обработчики;**
5. **`response.Normalize()`** — только если ошибок не было.

| Что пошло не так | Что получает App |
|---|---|
| простая ошибка из `fill` | «неверные данные» с сообщением `invalid request`; подробность остаётся в причине — для логов |
| простая ошибка из `Validate` | «неверные данные» с её текстом |
| `rockhandlers.Invalid(...)` и другие виды | как есть |
| паника где угодно | «непредвиденное» с `*PanicError` |
| ошибка из `Normalize` | `*NormalizeError` (распознаётся через `errors.As`) |

## Middleware

Middleware — тот же обработчик, только с общими интерфейсами вместо конкретных структур, потому что он висит на любых цепочках:

```go
type Middleware = Handler[Params, Response]

func Log(c *rockhandlers.Ctx, _ rockhandlers.Params, _ rockhandlers.Response) error {
    err := c.Next()
    if err != nil {
        rocklog.Error("request failed", rocklog.Err(err))
    }
    return err
}

Create.Use(Log, rockhandlers.Timeout(5*time.Second))
```

`Use` ставит middleware перед цепочкой. Каждый следующий `Use` встаёт перед предыдущими, поэтому App добавляет свои middleware при регистрации, и они выполняются первыми.

## Значения запроса

```go
c.Set(userKey{}, user)  // в одном обработчике
v, ok := c.Get(userKey{}) // в следующем
```

## Ошибки

- **Конструкторы:** `Invalid`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`, `Unavailable`.
- **Детали:** `.WithFields(map)` — ошибки по полям; `.Wrap(err)` — причина, только для логов.
- **`KindOf(err)`:**
  - nil → `KindNone`;
  - `*Error` → его вид;
  - `context.Canceled` → `KindCanceled`;
  - `context.DeadlineExceeded` → `KindTimeout`;
  - всё остальное → `KindInternal`.
- **`MessageOf`** скрывает текст `Internal`; **`FieldsOf`** отдаёт ошибки по полям.

## Отмена

`Ctx` — это `context.Context`: передавайте его в базу, клиенты и каналы. `Next` не запускает следующий обработчик, если запрос уже отменён. `Timeout(d)` — middleware с дедлайном.

## Тестирование

```go
res, err := tasks.Create.Call(ctx, &tasks.CreateParams{Token: "t", Title: "write docs"})
```

`Call` проходит всё то же, что `Run`, только с готовыми параметрами: `Validate`, обработчики, `Normalize`.

## Для App

```go
response, err := pipeline.Run(ctx, func(params rockhandlers.Params) error {
    return decode(req, params) // заполнить пустые параметры из запроса
})
// err → ответ по KindOf(err); response → ответ протокола
```

Подробно — [Как написать свой App](./rockhandlers-transport).
