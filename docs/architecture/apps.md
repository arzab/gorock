# Apps

**Engine** · `cmd/apps/`

Здесь вы реализуете логику вашего приложения. App оборачивает технологию взаимодействия — HTTP-сервер, consumer, scheduler, CLI — и умеет принимать delivery-сущности из Realm и обрабатывать их результаты.

```
cmd/apps/
├── configs.go          — Configs: агрегирует конфиги всех Apps
├── main.go             — список App, Init(configs Configs), Exec() []error
├── interface.go        — интерфейс App
├── http_server/
│   ├── configs.go      — Configs (port, timeout, api_path, ...)
│   └── app.go          — реализация, NewApp(configs Configs) apps.Interface
└── rabbit_consumer/
    ├── configs.go      — Configs (queue, prefetch, ...)
    └── app.go          — реализация, NewApp(configs Configs) apps.Interface
```

---

## Что такое App

App — обёртка над технологией взаимодействия. HTTP, очередь, CLI, таймер — это технологии через которые кто-то хочет что-то от системы. App оборачивает эту технологию и берёт на себя три ответственности:

**1. Гарантии работоспособности транспорта**
Panic recovery, логирование, трассировка, метрики — всё что должно быть в любом App независимо от бизнеса. Разработчик не думает об этом: App даёт это из коробки.

**2. Обработка результатов Realm**
Realm возвращает типизированную ошибку с конкретным кодом или обычную ошибку. App знает язык протокола — HTTP коды, ack/nack, exit codes. Realm знает язык бизнеса. App переводит одно в другое.

**3. Контракт с Realm через самоописывающиеся сущности**
Realm описывает свои обработчики сам — метод, путь, handlers. App регистрирует их не зная деталей. Не App диктует Realm как писать — Realm говорит App что зарегистрировать.

---

## Интерфейс

```go
// cmd/apps/interface.go
type Interface interface {
    Init() error
    Exec() error
    Shutdown() error // сигнал "заканчивай текущее, новое не принимай"
    Stop() []error   // моментальная остановка
}
```

`Init` — подготовка: регистрация delivery-сущностей из Realm.

`Exec` — запуск. Для long-running блокирующий, для short-running завершается когда работа сделана.

`Shutdown` — начало мягкой остановки: перестать принимать новое, дождаться завершения текущего.

`Stop` — финальная очистка ресурсов.

Оркестрацией занимается Main — запуск, ожидание сигнала ОС, вызов `Shutdown`/`Stop`. App не знает о других App.

---

## Обработка результатов Realm

App знает язык своего протокола, Realm знает язык бизнеса. App переводит одно в другое.

Если delivery вернул обычную `error` — это непредвиденная ошибка, возвращаем 500. Если delivery вернул типизированный `ResponseError` с кодом — возвращаем именно его:

```go
func (a *app) handleError(ctx *fiber.Ctx, err error) error {
    var respErr *responses.ErrorResponse
    if errors.As(err, &respErr) {
        // Delivery явно сказал что вернуть — используем его код
        return ctx.Status(respErr.Code).JSON(respErr)
    }
    // Непредвиденная ошибка — 500
    return ctx.Status(500).JSON(responses.NewError(500, "internal error"))
}
```

Delivery возвращает типизированную ошибку так:

```go
// internal/delivery/endpoints/create_user/handlers.go
func createUser() fiber.Handler {
    return func(ctx *fiber.Ctx) error {
        params, _ := GetParams(ctx)
        if params.Email == "" {
            return responses.NewError(400, "email is required")
        }
        if err := services.Repository().Save(&models.User{Email: params.Email}); err != nil {
            return responses.NewError(500, "failed to create user")
        }
        return ctx.Next()
    }
}
```

---

## Два типа App

**Long-running** — HTTP сервер, consumer, scheduler. Работают пока не получат сигнал остановки. `Exec` блокирующий — Main запускает его в горутине.

**Short-running** — CLI, скрипты, миграции. Выполнили задачу и завершились. `Exec` возвращает управление когда работа сделана. Retry loop и ожидание сигнала ОС не нужны.

---

## Готовые App из gorock-kit

Для большинства проектов своё App писать не нужно. gorock-kit содержит готовые реализации с батареей включённой:

| App | Пакет | Что делает |
|-----|-------|------------|
| HTTP сервер | `gorock/kit/apps/http` | Fiber с middleware, panic recovery, graceful shutdown |
| Consumer | `gorock/kit/apps/consumer` | Pub/Sub consumer с ack/nack логикой |
| Scheduler | `gorock/kit/apps/scheduler` | Планировщик задач по расписанию |
| CLI | `gorock/kit/apps/cli` | CLI приложение с командами |

---

## Как написать свой App

Свой App нужен когда готовые из gorock-kit не подходят — нестандартный транспорт, особая логика обработки ошибок, специфичные гарантии.

При написании своего App нужно ответить на три вопроса:

1. **Что гарантирует мой транспорт?** — какие middleware, какая обработка паник, какое логирование
2. **Как я обрабатываю результаты Realm?** — как типизированные ошибки превращаются в ответ протокола
3. **Какой контракт я даю Realm?** — какой интерфейс должны реализовывать delivery-сущности чтобы App мог их зарегистрировать

**Минимальный пример:**

```go
// cmd/apps/my_app/app.go
type app struct {
    configs Configs
}

func (a *app) Init() error {
    // подготовка: соединения, регистрация обработчиков из Realm
    return nil
}

func (a *app) Exec() error {
    // запуск — блокирующий для long-running, обычный для short-running
    return nil
}

func (a *app) Shutdown() error {
    // перестать принимать новое, дать завершиться текущему
    return nil
}

func (a *app) Stop() []error {
    // финальная очистка ресурсов
    return nil
}

func NewApp(configs Configs) apps.Interface {
    return &app{configs: configs}
}
```

---

## Правила

- **App решает когда, Realm решает что.** Транспорт и lifecycle — App. Бизнес-логика — delivery.
- **Apps не общаются между собой** напрямую — только через `pkg/services` или `pkg/infra/eventbus`.
- **Graceful shutdown — ответственность App.** Он знает как корректно завершить свою работу.
- **Short-running App не нуждается в retry loop.** Выполнил задачу — вернул управление.
