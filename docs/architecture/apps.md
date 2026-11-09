# Apps

**Engine** · `cmd/apps/`

App оборачивает технологию взаимодействия — HTTP-сервер, consumer, планировщик, бот — и умеет принимать delivery-сущности из Realm и обрабатывать их результаты.

```
cmd/apps/
├── apps.go            — Configs всех Apps + Register(engine, configs)
└── queue_consumer/    — свой App, только если готового в gorock-kit нет
    ├── configs.go
    └── app.go
```

Для HTTP, событий, расписания и Telegram папки не нужны: готовые App из gorock-kit создаются прямо в `apps.go`.

---

## Что такое App

HTTP, очередь, CLI, таймер — это технологии, через которые кто-то хочет что-то от системы. App оборачивает технологию и берёт на себя три ответственности:

**1. Гарантии работоспособности транспорта.**
Panic recovery, логирование, trace-id, метрики — всё, что должно быть в любом App независимо от бизнеса.

**2. Обработка результатов Realm.**
Realm возвращает типизированную ошибку с конкретным кодом или обычную ошибку. App знает язык протокола — HTTP-коды, ack/nack, exit codes — и переводит одно в другое.

**3. Контракт с Realm через самоописывающиеся сущности.**
Realm описывает свои обработчики сам — метод, путь, handlers. App регистрирует их, не зная деталей. Не App диктует Realm, как писать, — Realm говорит App, что зарегистрировать.

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
| `Init(ctx)` | после инициализации всех сервисов | подготовка: создать сервер или клиента, зарегистрировать delivery-сущности |
| `Exec(ctx)` | после `Init` всех apps, в своей горутине | работа. Для long-running — до отмены `ctx`; при отмене перестать принимать новое, завершить текущее и вернуться |
| `Stop()` | при остановке, сразу после отмены `ctx` | разблокировать `Exec`, если он блокирующий (например, `server.Shutdown`), и дождаться текущей работы |

Отдельного `Shutdown` нет: «заканчивай текущее, новое не принимай» — это отмена `ctx`, переданного в `Exec`. Движок вызывает `Stop` не больше одного раза на каждый успешный `Init` и сам следит за таймаутом. Что бы `Exec` ни вернул после отмены `ctx`, это чистая остановка, а не ошибка.

---

## `cmd/apps/apps.go`

Единственная точка входа в слой: собирает конфиги и регистрирует все App.

```go
package apps

type Configs struct {
    HTTP rockfiber.Config
    Bus  rockbus.Config  `config:",omitempty"`
    Cron rockcron.Config `config:",omitempty"`
}

// Порядок важен: apps останавливаются в обратном порядке, поэтому HTTP-сервер
// (зарегистрирован последним) остановится первым, пока шина ещё работает.
func Register(e *rockengine.Engine, cfg Configs) {
    bus := rockbus.NewApp(cfg.Bus, onBusError, events.Subscriptions()...)
    rockbus.SetDefault(bus)
    e.MustRegister("bus", bus)

    e.MustRegister("cron", rockcron.NewApp(cfg.Cron, onJobError, jobs.All()...))

    cfg.HTTP.OnError = func(traceId string, err *rockfiber.ErrorResponse) {
        log := rocklog.With(rocklog.Str("trace_id", traceId), rocklog.Int("code", err.Code))
        if err.Code >= 500 {
            log.Error("http error", rocklog.Str("error", err.Message)) // настоящее сообщение, даже если клиенту оно скрыто
            return
        }
        log.Warn("http client error", rocklog.Str("error", err.Message))
    }
    http := rockfiber.New(cfg.HTTP, endpoints.All()...)
    e.MustRegister("http", http, rockengine.RestartPolicy{MaxRetries: 3, Delay: time.Second, ResetAfter: time.Minute})
}

func onBusError(ctx context.Context, event rockbus.Event, err error) {
    rocklog.FromContext(ctx).Error("event failed", rocklog.Str("topic", string(event.Topic)), rocklog.Err(err))
}

func onJobError(ctx context.Context, job rockcron.Job, err error) {
    rocklog.Error("job failed", rocklog.Str("job", rockcron.JobName(job)), rocklog.Err(err))
}
```

App получает от Realm список сущностей — `endpoints.All()`, `events.Subscriptions()`, `jobs.All()` — и регистрирует их, не вникая в логику.

`RestartPolicy` — что делать, если `Exec` вернул ошибку или запаниковал: сколько раз перезапустить и с какой паузой. Без политики упавший App останавливает весь процесс — это безопаснее, чем работать без HTTP-сервера. Подробнее → [rockengine](/gorock-kit/rockengine).

---

## Обработка результатов Realm

App знает язык своего протокола, Realm знает язык бизнеса. В [rockfiber](/gorock-kit/rockfiber) перевод уже встроен:

| Delivery вернул | Клиент получит |
|---|---|
| `rockfiber.NewError(404, "task not found")` | `404 {"code":404,"status":"Not Found","message":"task not found"}` |
| любую другую `error` | `500`; текст заменяется на `mask_internal_server_error_message`, если он задан, а настоящий уходит в `OnError` |
| паника | `500`, стек в stderr; сервер продолжает работать |

```go
// internal/delivery/endpoints/get_task/endpoint.go
err = services.Repository().GetByID(c.UserContext(), task, params.ID)
if errors.Is(err, repository.ErrNotFound) {
    return rockfiber.NewError(http.StatusNotFound, "task not found") // бизнес-решение: 404
}
if err != nil {
    return err // непредвиденное: 500
}
```

---

## Готовые App из gorock-kit

| App | Модуль | Delivery-сущности |
|-----|-------|------------|
| HTTP-сервер | [rockfiber](/gorock-kit/rockfiber) | `[]rockfiber.FiberEndpoint` из `internal/delivery/endpoints` |
| Шина событий | [rockbus](/gorock-kit/rockbus) | `[]rockbus.Subscription` из `internal/delivery/events` |
| Планировщик | [rockcron](/gorock-kit/rockcron) | `[]rockcron.Job` из `internal/delivery/jobs` |
| Telegram-бот | [rocktelebot](/gorock-kit/rocktelebot) | `[]rocktelebot.Handler` из `internal/delivery/bot` |

---

## Как написать свой App

Свой App нужен, когда готового нет — например, consumer очереди или gRPC-сервер. Ответьте на три вопроса:

1. **Что гарантирует мой транспорт?** — panic recovery, логирование, ack/nack.
2. **Как я перевожу результаты Realm?** — какая ошибка во что превращается.
3. **Какой контракт я даю Realm?** — какой тип должны иметь delivery-сущности.

```go
// cmd/apps/queue_consumer/app.go
package queue_consumer

// Handler — контракт для Realm: delivery описывает обработчики этого типа.
type Handler func(ctx context.Context, body []byte) error

type App struct {
    cfg      Configs
    handlers map[string]Handler
    conn     *queue.Conn
    started  atomic.Bool
}

func NewApp(cfg Configs, handlers map[string]Handler) *App {
    return &App{cfg: cfg, handlers: handlers}
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
    defer func() { // гарантия №1: паника в Realm не роняет consumer
        if r := recover(); r != nil {
            rocklog.Error("handler panic", rocklog.Any("panic", r))
            msg.Nack()
        }
    }()
    if err := a.handlers[msg.Topic](ctx, msg.Body); err != nil { // гарантия №2: ошибка → nack
        rocklog.FromContext(ctx).Error("handle message", rocklog.Err(err))
        msg.Nack()
        return
    }
    msg.Ack()
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

`queue` здесь — условный клиент вашей очереди. Регистрация — как у готовых: `e.MustRegister("consumer", queue_consumer.NewApp(cfg.Consumer, consumers.All()))`.

::: warning Stop не должен ждать Exec
`Stop` вызывается и тогда, когда `Exec` не запускался: движок останавливает уже инициализированные компоненты, если другой упал на старте. `Stop`, который ждёт сигнала от `Exec`, в этом случае зависнет навсегда.
:::

---

## Short-running

CLI, миграции, разовые скрипты выполняют задачу и завершаются. Движок рассчитан на long-running процессы: `Run` ждёт сигнала ОС, даже когда все App закончили работу. Для short-running задачи движок не нужен — вызовите сервисы напрямую:

```go
// cmd/migrate/main.go
repo := repository.NewService(cfg.Services.Repository)
if err := repo.Init(ctx); err != nil {
    return err
}
defer repo.Stop()

return migrations.Run(ctx, repo)
```

---

## Правила

- **App решает когда, Realm решает что.** Транспорт и lifecycle — App. Бизнес-логика — delivery.
- **Готовый App из gorock-kit — по умолчанию.** Свой — только когда готового нет.
- **Apps не общаются между собой** напрямую — только через `pkg/services` или шину событий ([rockbus](/gorock-kit/rockbus)).
- **Порядок регистрации = обратный порядок остановки.** То, от чего зависят другие apps, регистрируйте раньше.
- **`Exec` возвращается при отмене `ctx`.** Если `Exec` блокирующий, `Stop` должен его разблокировать.
