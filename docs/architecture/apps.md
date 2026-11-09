# Apps

**Engine** · `cmd/apps/`

App оборачивает технологию взаимодействия — HTTP-сервер, consumer, планировщик, бот: принимает запросы, отдаёт их маршрутам из Realm и переводит результаты на язык протокола.

```
cmd/apps/
├── apps.go            — Configs всех Apps + Register(engine, configs)
└── queue_consumer/    — свой App, только если готового в каталоге нет
    ├── configs.go
    └── app.go
```

Для HTTP папка не нужна: готовый [rockhttp](/catalog/rockhttp) создаётся прямо в `apps.go`.

---

## Что такое App

HTTP, очередь, CLI, таймер — это технологии, через которые кто-то хочет что-то от системы. App оборачивает технологию и берёт на себя три ответственности:

**1. Гарантии транспорта.**
Graceful shutdown, лимиты, протокольные вещи (CORS, gzip, ack/nack, prefetch) — всё, что зависит от технологии.

**2. Перевод запроса и результата.**
App заполняет `Params` маршрута из своего протокола (тело, заголовки, path, заголовки сообщения) и переводит результат или **вид** ошибки обратно: `NotFound` → 404 в HTTP, nack в очереди, код выхода в CLI.

**3. Регистрация маршрутов.**
Маршруты — цепочки [rockhandlers](/gorock-kit/rockhandlers) из `internal/delivery`. App регистрирует их, не зная, что они делают.

Всё, что одинаково для любого транспорта — цепочка шагов, валидация, перехват паник, Locals, middleware, — уже в rockhandlers. App его не повторяет.

---

## Интерфейс

App — это `rockengine.App`:

```go
type App interface {
    Init(ctx context.Context) error
    Exec(ctx context.Context) error
    Stop() []error
}
```

| Метод | Когда | Что делает |
|---|---|---|
| `Init(ctx)` | после того, как `cmd/main` поднял сервисы | подготовка: создать сервер или клиента, зарегистрировать маршруты |
| `Exec(ctx)` | после `Init` всех apps, в своей горутине | работа. Для long-running — до отмены `ctx`; при отмене перестать принимать новое, завершить текущее и вернуться |
| `Stop()` | при остановке, сразу после отмены `ctx` | разблокировать `Exec`, если он блокирующий (например, `server.Shutdown`), и дождаться текущей работы |

Отдельного `Shutdown` нет: «заканчивай текущее, новое не принимай» — это отмена `ctx`, переданного в `Exec`. Движок вызывает `Stop` не больше одного раза на каждый успешный `Init` и сам следит за таймаутом. Что бы `Exec` ни вернул после отмены `ctx`, это чистая остановка, а не ошибка.

---

## `cmd/apps/apps.go`

Единственная точка входа в слой: собирает конфиги и регистрирует все App.

```go
package apps

import (
    "time"

    "github.com/arzab/gorock-catalog/rockhttp"
    "github.com/arzab/gorock-kit/rockengine"

    "github.com/yourname/taskapi/internal/delivery/endpoints"
    "github.com/yourname/taskapi/internal/delivery/middleware"
)

type Configs struct {
    HTTP rockhttp.Config
}

func Register(e *rockengine.Engine, cfg Configs) {
    http := rockhttp.New(cfg.HTTP, endpoints.All()...).
        Use(middleware.Log) // middleware rockhandlers вокруг всех маршрутов
    e.MustRegister("http", http, rockengine.RestartPolicy{MaxRetries: 3, Delay: time.Second, ResetAfter: time.Minute})
}
```

Порядок регистрации важен: apps останавливаются в обратном порядке. Регистрируйте HTTP последним — он остановится первым, пока то, во что он пишет (шина событий, consumer), ещё работает.

---

## Перевод результатов

App знает язык своего протокола, Realm знает язык бизнеса. Маршрут возвращает результат или ошибку с видом, App переводит:

| Realm вернул | [rockhttp](/catalog/rockhttp) | consumer очереди (пример ниже) |
|---|---|---|
| заполненный `Result` | `200` / `Endpoint.Status`, JSON | ack (+ ответ) |
| `rockhandlers.Invalid("...")` | `400 {"kind":"invalid","message":"...","fields":{...}}` | nack → DLQ |
| `rockhandlers.NotFound("task not found")` | `404` | nack → DLQ |
| `rockhandlers.Unavailable(...)` | `503` | requeue |
| обычная `error`, паника | `500 "internal error"`, текст скрыт | requeue |

---

## Готовые App

| App | Модуль | Что принимает |
|-----|-------|------------|
| HTTP-сервер | [rockhttp](/catalog/rockhttp) | `[]rockhttp.Endpoint{Method, Path, Handler: rockhandlers.Route}` |
| HTTP-сервер на Fiber | [rockfiber](/catalog/rockfiber) | свои `[]rockfiber.FiberEndpoint` (старая модель) |
| Шина событий | [rockbus](/catalog/rockbus) | `[]rockbus.Subscription` |
| Планировщик | [rockcron](/catalog/rockcron) | `[]rockcron.Job` |
| Telegram-бот | [rocktelebot](/catalog/rocktelebot) | свои обработчики |

Шина, планировщик и бот пока не переведены на маршруты rockhandlers — см. [каталог](/catalog/).

---

## Как написать свой App

Свой App нужен, когда готового нет — например, consumer очереди или gRPC-сервер. С rockhandlers App пишет только протокольную часть: как заполнить `Params` и что сделать с результатом. Пошагово, с чек-листом — [Как написать транспорт](/gorock-kit/rockhandlers-transport).

```go
// cmd/apps/queue_consumer/app.go
package queue_consumer

// Consumer — контракт для Realm: очередь и маршрут rockhandlers.
type Consumer struct {
    Queue   string
    Handler rockhandlers.Route
}

type App struct {
    cfg       Configs
    consumers []Consumer
    conn      *queue.Conn
    started   atomic.Bool
}

func NewApp(cfg Configs, consumers ...Consumer) *App {
    return &App{cfg: cfg, consumers: consumers}
}

func (a *App) Init(ctx context.Context) error {
    conn, err := queue.Dial(ctx, a.cfg.URL)
    if err != nil {
        return fmt.Errorf("dial: %w", err)
    }
    a.conn = conn
    a.started.Store(false) // Init вызывается заново при рестарте
    return nil
}

func (a *App) Exec(ctx context.Context) error {
    a.started.Store(true)
    defer a.conn.Close() // с момента запуска соединением владеет Exec
    for {
        msg, err := a.conn.Receive(ctx) // возвращается при отмене ctx
        if err != nil {
            return err // после отмены ctx движок считает это чистой остановкой
        }
        a.handle(ctx, msg) // текущее сообщение дообрабатывается до конца
    }
}

func (a *App) handle(ctx context.Context, msg *queue.Message) {
    route := a.route(msg.Queue)
    // Разбор — часть цепочки: middleware увидят и его ошибки. Паники ловит rockhandlers.
    _, err := route.Serve(ctx, func(params any) error {
        if err := json.Unmarshal(msg.Body, params); err != nil {
            return err // → Invalid
        }
        return bindHeaders(msg.Headers, params) // заголовки сообщения → теги Params
    })
    switch rockhandlers.KindOf(err) {
    case rockhandlers.KindNone:
        msg.Ack()
    case rockhandlers.KindUnavailable, rockhandlers.KindInternal, rockhandlers.KindCanceled, rockhandlers.KindTimeout:
        msg.Requeue() // временная проблема или остановка — повторить
    default:
        msg.Reject() // плохое сообщение — повтор не поможет, в DLQ
    }
}

// Exec сам реагирует на отмену ctx, поэтому разблокировать его не нужно —
// движок отменит ctx и дождётся возврата Exec. Stop убирает за собой только
// если Exec так и не запустился (например, на старте упал другой App).
func (a *App) Stop() []error {
    if a.conn == nil || a.started.Load() {
        return nil
    }
    return []error{a.conn.Close()}
}
```

`queue` здесь — условный клиент вашей очереди. Один и тот же маршрут можно повесить и на HTTP, и на очередь:

```go
e.MustRegister("consumer", queue_consumer.NewApp(cfg.Consumer,
    queue_consumer.Consumer{Queue: "tasks.create", Handler: create_task.Route}))
```

::: warning Stop не должен ждать Exec
`Stop` вызывается и тогда, когда `Exec` не запускался: движок останавливает уже инициализированные apps, если другой упал на старте. `Stop`, который ждёт сигнала от `Exec`, в этом случае зависнет навсегда.
:::

---

## Short-running

CLI, миграции, разовые скрипты выполняют задачу и завершаются. Движок рассчитан на long-running процессы: `Run` ждёт сигнала ОС, даже когда все App закончили работу. Для short-running задачи движок не нужен — поднимите сервисы и сделайте дело:

```go
// cmd/migrate/main.go
services.Register(rockservices.Default(), cfg.Services)
if err := rockservices.Init(ctx); err != nil {
    return err
}
defer rockservices.Stop()

return migrations.Run(ctx, repository.Get())
```

---

## Правила

- **App решает когда и как, Realm решает что.** Протокол и lifecycle — App. Бизнес-логика — маршруты в delivery.
- **App не повторяет rockhandlers.** Валидация, паники, middleware, Locals — уже в цепочке; App только заполняет `Params` и переводит результат.
- **Готовый App из каталога — по умолчанию.** Свой — только когда готового нет.
- **Apps не общаются между собой** напрямую — только через `pkg/services` или шину событий.
- **Порядок регистрации = обратный порядок остановки.** То, от чего зависят другие apps, регистрируйте раньше.
- **`Exec` возвращается при отмене `ctx`.** Если `Exec` блокирующий, `Stop` должен его разблокировать.
