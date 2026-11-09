# rockhandlers

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockhandlers/README_RU.md — правьте README в репозитории модуля. -->

::: tip Место в GOROCK · Realm · Delivery
Форма обработки запроса в `internal/delivery`: цепочки шагов с `Next`, Locals, middleware, валидацией и видами ошибок — без привязки к HTTP или очереди. Транспорт (App) только заполняет Params и переводит результат в свой протокол. → [Delivery](/architecture/delivery)
:::

```sh
go get github.com/arzab/gorock-kit/rockhandlers@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-kit/tree/main/rockhandlers)

Форма обработки запроса, не зависящая от технологии. Это то, что во Fiber дают `Ctx`, `Next`, `Locals`, обработчик ошибок и recover, только одинаково для HTTP, очередей, CLI и чего угодно. Без зависимостей.

```
транспорт (HTTP, очередь…)        rockhandlers                         Realm
заполняет Params        ──▶  middleware → шаг 1 → шаг 2 … ──▶  сервисы / сценарии
переводит Result/ошибку ◀──  Result или ошибка с видом (Kind)
```

Слой отвечает за цепочку, `Next`, Locals, перехват паники, виды ошибок и отмену. Транспорт — за протокол:
- заполнить `Params`: тело, заголовки, path, AMQP-заголовки, флаги…;
- превратить `Result` в ответ: JSON, файл, редирект, ack, stdout…;
- перевести вид ошибки в протокол: статус, nack, код выхода.

## Шаги и цепочки

```go
type CreateParams struct {
    auth.BearerAuth          // встроенная модель → шаг auth.Bearer принимает эти параметры
    Item string `json:"item"`
    Qty  int    `json:"qty"`
}
type CreateResult struct{ ID string `json:"id"` }

func Create(c *rockhandlers.Ctx[*CreateParams, *CreateResult]) error {
    id, err := orders.Create(c, c.Params.Item, c.Params.Qty) // c — это context.Context
    if err != nil {
        return err
    }
    c.Result.ID = id
    return nil
}

var CreateOrder = rockhandlers.Chain("orders.create", auth.Bearer, Create)
```

Длинную сигнатуру шага можно сократить псевдонимом типа рядом с моделью — это тот же тип, в рантайме он ничего не стоит:

```go
type CreateCtx = rockhandlers.Ctx[*CreateParams, *CreateResult]

func Create(c *CreateCtx) error { … }
```

- `Ctx[P, R]` хранит типизированные `Params` и `Result`. `Params` заполняет транспорт, `Result` создаётся до цепочки, и его заполняют шаги. P и R — указатели на структуры.
- Шаг, который не вызвал `c.Next()`, завершает цепочку. Ранний успешный выход — например, попадание в кэш — это нормально.
- Повторный `Next()` возвращает результат первого вызова. `Next()` после последнего шага возвращает nil.
- Если `Params` реализует `Validate() error`, валидация выполняется после того, как транспорт заполнил `Params`, и перед первым шагом: сначала проверка, потом использование. Простая ошибка из неё становится `Invalid`.
- `Chain` стирает дженерики. `Route` — обычное значение, поэтому маршруты разных типов лежат в одном слайсе.

## Переиспользуемые шаги: ограничения вместо приведений

Шаг, которому нужны данные запроса, объявляет требование интерфейсом, а компилятор проверяет, что модель его выполняет:

```go
type Params interface{ BearerToken() string }

func Bearer[P Params, R any](c *rockhandlers.Ctx[P, R]) error {
    user, err := verify(c, c.Params.BearerToken())
    if err != nil {
        return rockhandlers.Unauthorized("invalid token")
    }
    UserKey.Set(c, user)
    return c.Next()
}
```

Модели выполняют требование встраиванием — как `auth.BearerAuth` выше. Забыли встроить — код не соберётся. `Chain` выводит P и R из конкретного шага и подставляет эти типы в дженерик-шаги.

## Middleware

Для сквозной логики, которая не смотрит в данные запроса: логирование, метрики, трейсинг, таймауты, обработка ошибок.

```go
func Log(c rockhandlers.Context) error {
    err := c.Next()
    if err != nil { /* лог, метрика, переклассификация */ }
    return err
}

route = route.With(Log, rockhandlers.Timeout(5*time.Second))
```

`With` оборачивает маршрут:
- первый middleware — самый внешний, а каждый следующий вызов `With` оборачивает всё, что добавили раньше. Так же транспорт добавляет свои middleware уровня приложения — через `route.With(...)` при регистрации;
- порядок выполнения: middleware транспорта → `With` → разбор запроса (заполнение от транспорта) → валидация → шаги;
- `c.SetContext(ctx)` заменяет контекст на остаток запроса.

## Группы

Группа — обычная дженерик-функция в вашем коде: у методов в Go не бывает своих параметров типа.

```go
func Admin[P auth.Params, R any](name string, steps ...rockhandlers.Handler[P, R]) rockhandlers.Route {
    return rockhandlers.Chain(name, rockhandlers.Prepend(steps, auth.Bearer[P, R], admin.Require[P, R])...).
        With(Log)
}

var BanUser = Admin("users.ban", users.Ban)
```

## Locals

```go
var UserKey = rockhandlers.NewLocal[*User]("user")

UserKey.Set(c, u)
u, ok := UserKey.Get(c)  // *User, без приведения
u := UserKey.MustGet(c)  // паника (→ Internal), если нет шага, который кладёт значение
```

## Ошибки

- **Конструкторы:** `Invalid`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`, `Unavailable`.
- **Детали:** `.WithFields(map)` добавляет ошибки по полям; `.Wrap(err)` — причину, только для логов.
- **`KindOf(err)`:**
  - nil → `KindNone`;
  - `*Error` → его вид;
  - `context.Canceled` → `KindCanceled`;
  - `context.DeadlineExceeded` → `KindTimeout`;
  - всё остальное → `KindInternal`.
- **`MessageOf`** отдаёт сообщение для клиента и скрывает сообщения Internal. **`FieldsOf`** отдаёт ошибки по полям.
- **Паники:** паника в любом месте цепочки становится ошибкой Internal с `*PanicError{Value, Stack}`. Middleware видят её как обычную ошибку.

## Отмена

Модель нативная для Go и кооперативная. `Ctx` — это `context.Context`: передавайте его в БД, клиенты и каналы, и они вернутся сразу после отмены. `Next` не запускает следующие шаги, если контекст уже завершён.

Логика «при отмене» — из стандартной библиотеки:
- `defer` — уборка;
- `context.AfterFunc(c, fn)` — немедленная реакция на сигнал;
- `context.WithoutCancel(c)` — работа, которая должна завершиться вопреки отмене.

## Для транспортов

```go
result, err := route.Serve(ctx, func(params any) error {
    return decode(req, params) // заполнить новый P из запроса
})                             // не паникует; при ошибке result == nil
// err → протокол по KindOf(err); result (R) → протокол
```

Функция заполнения выполняется внутри цепочки — после middleware и перед валидацией. Поэтому middleware видят ошибки разбора, а таймаут покрывает и разбор. Простая ошибка из неё становится `Invalid` с её текстом; чтобы задать вид и сообщение, верните `*Error`.

`ParamsType()` и `ResultType()` позволяют транспорту проверить маршруты на старте — например, отказать типу результата, который он не умеет отдавать.

## Тестирование

```go
res, err := rockhandlers.Run[*CreateResult](ctx, CreateOrder, &CreateParams{Qty: 1}) // P выводится
```

`Run` прогоняет всю цепочку — middleware, валидацию, шаги — без транспорта. Шаги — обычные функции, их можно тестировать и по отдельности.

## Вне слоя

- **Это задача транспортов:** стриминг (SSE, WebSocket), повторы, DLQ, prefetch, лимиты параллельности, протокольные middleware (CORS, gzip).
- **Это задача вашего кода или каталога:** готовые шаги — auth, tenant, paging, кэш, логирование.
