# rockbus

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockbus/README_RU.md — правьте README в репозитории модуля. -->

::: tip Место в GOROCK · App
Шина событий внутри процесса — App в `cmd/apps`. Публикация — `rockbus.PublishAsync`. → [Как писать App](/guide/apps)
:::

::: warning Перенесён из gorock-kit v0.2.0 как есть
Код и API — те же, что были в `github.com/arzab/gorock-kit/rockbus@v0.2.0`, путь модуля теперь `github.com/arzab/gorock-catalog/rockbus`. Модуль ещё не переведён на новое ядро: его подписчики — не обработчики rockhandlers.
:::

```sh
go get github.com/arzab/gorock-catalog/rockbus@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-catalog/tree/main/rockbus)

Внутрипроцессная шина событий для Go-приложений. Позволяет слоям приложения общаться через типизированные события без прямых зависимостей друг от друга.

Каждый топик получает собственную горутину и буферизованную очередь, что гарантирует обработку событий одного топика **строго по порядку**. Разные топики обрабатываются **конкурентно** и не блокируют друг друга.

Реализует интерфейс rockengine `App` (`Init / Exec / Stop`).

## Быстрый старт

```go
// Определяем подписки в delivery-слое
var Subscriptions = []rockbus.Subscription{
    rockbus.On("user.created",  onUserCreated),
    rockbus.On("order.placed",  onOrderPlaced),
}

// Создаём App с подписками
app := rockbus.NewApp(
    rockbus.Config{QueueSize: 1024},
    func(ctx context.Context, event rockbus.Event, err error) {
        log.Error("bus error", rocklog.Str("topic", string(event.Topic)), rocklog.Err(err))
    },
    Subscriptions...,
)

// Устанавливаем как глобальный экземпляр
rockbus.SetDefault(app)

// Регистрируем в rockengine
engine.MustRegister("bus", app, rockengine.RestartPolicy{})
engine.Run()
```

## Конфигурация

```go
type Config struct {
    // Ёмкость буфера очереди каждого топика. По умолчанию: 1024.
    // PublishAsync дропает события при заполнении и вызывает OnError.
    QueueSize int

    // Вызывается при ошибке/панике хендлера, переполнении очереди,
    // остановке приложения или отсутствии воркера для топика. Nil = игнорировать.
    OnError func(ctx context.Context, event Event, err error)
}
```

## Жизненный цикл

```
NewApp(cfg, onError, subs...) → SetDefault → Init → Exec (блокирует) → Stop
```

- **NewApp** принимает подписки прямо в конструктор — все топики известны до старта.
- **Subscribe** на новый топик нужно вызвать **до Init** — Init создаёт по очереди на топик. Добавлять хендлеры в существующий топик можно в любой момент.
- **Init** создаёт очереди и сбрасывает состояние. `PublishAsync` работает сразу после `Init`: события буферизуются, пока `Exec` не запустит воркеры, поэтому другие apps могут публиковать, как только движок инициализировал шину. Безопасно вызывать повторно после `Stop` для сценариев рестарта.
- **Exec** блокирует до отмены `ctx` или вызова `Stop`. Воркеры дрейнят очереди перед выходом.
- **Stop** идемпотентен и безопасен для вызова до `Init`. Если `Exec` так и не запускался, буферизованные события передаются в `OnError` с `ErrAppStopped`.

Каждое событие, переданное в `PublishAsync`, либо доставляется хендлерам, либо попадает в `OnError` — молча не теряется, в том числе при остановке.

## Подписка

### Через конструктор (рекомендуется)

```go
type UserCreated struct {
    UserID int
    Email  string
}

func onUserCreated(ctx context.Context) error {
    p, err := rockbus.Payload[UserCreated](ctx)
    if err != nil {
        return err
    }
    fmt.Println("user created:", p.UserID)
    return nil
}

var Subscriptions = []rockbus.Subscription{
    rockbus.On("user.created", onUserCreated),
    rockbus.On("order.placed", onOrderPlaced),
}

app := rockbus.NewApp(cfg, onError, Subscriptions...)
```

### Через Subscribe (до Init)

```go
err := app.Subscribe("user.created", func(ctx context.Context) error {
    p, err := rockbus.Payload[UserCreated](ctx)
    if err != nil {
        return err
    }
    return nil
})

// Или через глобальную функцию
err = rockbus.Subscribe("user.created", onUserCreated)
```

`Subscribe` возвращает ошибку с `ErrNoWorker` для **нового** топика после `Init` (у него не было бы очереди) и для nil-хендлера.

На один топик можно зарегистрировать несколько хендлеров — все вызываются в порядке регистрации.

## Payload в контексте

Каждый хендлер получает `ctx` с инжектированным payload и именем топика:

```go
func onUserCreated(ctx context.Context) error {
    // Типобезопасное извлечение payload
    p, err := rockbus.Payload[UserCreated](ctx)
    if err != nil {
        return err
    }

    // Текущий топик (если нужен)
    topic := rockbus.CurrentTopic(ctx) // "user.created"

    fmt.Println("user created:", p.UserID, p.Email)
    return nil
}
```

`Payload[T]` возвращает `ErrPayload` если payload отсутствует или тип не совпадает.

## Публикация

### Синхронная

Выполняет хендлеры **в горутине вызывающего**, по порядку регистрации. Блокирует до завершения всех хендлеров. Все хендлеры вызываются даже если один упал.

```go
err := rockbus.Publish(ctx, rockbus.Event{
    Topic:   "user.created",
    Payload: UserCreated{UserID: 1, Email: "alice@example.com"},
})
// err — объединённая ошибка всех упавших хендлеров (errors.Join)
```

Используй `Publish` когда:
- Результат важен вызывающему
- Нужен гарантированный порядок со следующей операцией
- Операция является частью транзакции

### Асинхронная

Кладёт событие в очередь топика и **возвращается немедленно**. Воркер-горутина забирает и обрабатывает.

```go
rockbus.PublishAsync(ctx, rockbus.Event{
    Topic:   "user.created",
    Payload: UserCreated{UserID: 1, Email: "alice@example.com"},
})
```

Используй `PublishAsync` для side-эффектов, которые не должны блокировать: отправка email, обновление кеша, аналитика, уведомления.

Отмена контекста вызывающего и его дедлайн **отвязываются** — воркер выполняется до конца независимо от того, завершился ли HTTP-запрос, который породил событие.

## Гарантия порядка

В рамках одного топика события обрабатываются **строго в порядке публикации**:

```
PublishAsync "user.updated" {name: "Alice"} ──► воркер обрабатывает первым
PublishAsync "user.updated" {name: "Bob"}   ──► воркер обрабатывает вторым (гарантировано)
```

Разные топики независимы и выполняются конкурентно:

```
"user.updated"  ──► Воркер A  (упорядочен)
"order.placed"  ──► Воркер B  (упорядочен, независим от A)
```

## Значения в контексте

Передавай метаданные через события с помощью хелперов контекста:

```go
// Перед публикацией
ctx = rockbus.WithValue(ctx, "traceId", "abc-123")
ctx = rockbus.WithValue(ctx, "userID", 42)

rockbus.PublishAsync(ctx, event)

// В хендлере
func onUserCreated(ctx context.Context) error {
    traceId, _ := rockbus.GetValue[string](ctx, "traceId")
    userID, err := rockbus.GetValue[int](ctx, "userID")
    // ...
}
```

`GetValue[T]` возвращает ошибку если ключ отсутствует или тип не совпадает.

## Обработка ошибок

### Sentinel-ошибки

```go
errors.Is(err, rockbus.ErrQueueFull)  // очередь топика заполнена, событие дропнуто
errors.Is(err, rockbus.ErrAppStopped) // приложение остановлено, событие дропнуто
errors.Is(err, rockbus.ErrNoWorker)   // у топика нет воркера (Subscribe после Init)
errors.Is(err, rockbus.ErrPanic)      // хендлер запаниковал, паника перехвачена
errors.Is(err, rockbus.ErrPayload)    // payload отсутствует или тип не совпадает
```

### Хук OnError

```go
onError := func(ctx context.Context, event rockbus.Event, err error) {
    switch {
    case errors.Is(err, rockbus.ErrQueueFull):
        metrics.Inc("bus.queue_full", "topic", string(event.Topic))
    case errors.Is(err, rockbus.ErrPanic):
        log.Error("handler panic", rocklog.Str("topic", string(event.Topic)), rocklog.Err(err))
    default:
        log.Warn("bus error", rocklog.Str("topic", string(event.Topic)), rocklog.Err(err))
    }
}

app := rockbus.NewApp(cfg, onError, subs...)
```

`OnError` вызывается только для асинхронных ошибок. `Publish` (sync) возвращает ошибки напрямую вызывающему.

Паника внутри самого `OnError` перехватывается и пишется в `stderr` — она никогда не уронит воркер-горутину.

## Восстановление после паники

Оба метода — `Publish` и `PublishAsync` — перехватывают паники из хендлеров. Паника конвертируется в ошибку, обёрнутую в `ErrPanic`:

- **Publish**: возвращается как часть объединённой ошибки
- **PublishAsync**: передаётся в `OnError`

Стектрейс включается в сообщение ошибки.

## Ограничения

- **In-memory, at-most-once**: события, оставшиеся в очереди при падении процесса, теряются. Если нужна надёжность — используйте брокер.
- **Новые топики после Init**: топики, впервые подписанные после `Init`, не получают очереди. `PublishAsync` вызовет `OnError` с `ErrNoWorker`. `Publish` по-прежнему работает — он выполняется в горутине вызывающего.
- **Один воркер на топик**: если один топик получает события быстрее чем хендлер успевает обработать — очередь заполняется. Это сигнал вынести обработку в отдельный сервис. См. `ErrQueueFull`.
- **Нет wildcard-топиков**: подписка на `"user.*"` не поддерживается. Используй явные имена топиков.
- **Нет отписки**: хендлеры нельзя удалить после регистрации.
