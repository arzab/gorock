# rocktelebot

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rocktelebot/README_RU.md — правьте README в репозитории модуля. -->

::: tip Место в GOROCK · Engine · Apps
Telegram-бот — App в `cmd/apps`. Команды и обработчики описываются в `internal/delivery`. → [Apps](/architecture/apps)
:::

::: warning Перенесён из gorock-kit v0.2.0 как есть
Код и API — те же, что были в `github.com/arzab/gorock-kit/rocktelebot@v0.2.0`, путь модуля теперь `github.com/arzab/gorock-catalog/rocktelebot`. Модуль ещё не переведён на новое ядро: обработчики пока не на rockhandlers.
:::

```sh
go get github.com/arzab/gorock-catalog/rocktelebot@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-catalog/tree/main/rocktelebot)

Обёртка Telegram-бота на базе [telebot.v3](https://gopkg.in/telebot.v3) с интеграцией [rockengine](/gorock-kit/rockengine), структурированной регистрацией хендлеров, встроенными мидлварами и билдерами клавиатур.

Реализует интерфейс rockengine `App` (`Init / Exec / Stop`).

## Быстрый старт

```go
app := rocktelebot.NewApp(cfg,
    []tele.MiddlewareFunc{
        rocktelebot.Recovery(func(c tele.Context, err interface{}) {
            log.Error("panic", rocklog.Any("err", err))
        }),
        rocktelebot.Logger(func(c tele.Context) {
            log.Info("update", rocklog.Int64("user_id", c.Sender().ID), rocklog.Str("text", c.Text()))
        }),
    },
    rocktelebot.Command("/start", onStart).WithDesc("Начать работу"),
    rocktelebot.Command("/help",  onHelp).WithDesc("Помощь"),
    rocktelebot.GetHandler(tele.OnText,     onText),
    rocktelebot.GetHandler(tele.OnCallback, onCallback),
)

engine.MustRegister("bot", app, rockengine.RestartPolicy{})
engine.Run()
```

## Конфигурация

```go
type Config struct {
    // Токен Telegram Bot API от @BotFather. Обязателен.
    Token string

    // Webhook настраивает режим вебхука.
    // Если Listen пустой — используется long polling.
    Webhook WebhookConfig `config:",omitempty"`
}

type WebhookConfig struct {
    // Адрес на котором слушает вебхук-сервер, например ":8443".
    Listen string `config:",omitempty"`

    // Публичный HTTPS URL куда Telegram будет слать обновления.
    PublicURL string `config:"public_url,omitempty"`
}
```

```yaml
# config.yaml
token: "123456:ABC-DEF..."

# опционально — по умолчанию используется long polling
webhook:
  listen: ":8443"
  public_url: "https://example.com/bot"
```

## Жизненный цикл

```
NewApp(cfg, middlewares, handlers...) → Init → Exec (блокирует) → Stop
```

- **Init** создаёт бота, регистрирует мидлвары и хендлеры, отправляет описания команд в Telegram.
- **Exec** запускает polling/webhook и блокирует до отмены `ctx` или вызова `Stop`.
- **Stop** идемпотентен и безопасен для вызова до `Init`.

## Хендлеры

### Command

Срабатывает когда пользователь отправляет команду боту.

```go
rocktelebot.Command("/start", onStart)
rocktelebot.Command("/help",  onHelp, authMiddleware) // с мидлваром на конкретный хендлер
```

### GetHandler

Срабатывает на любой telebot-entity — текст, медиа, callback и прочее.

```go
rocktelebot.GetHandler(tele.OnText,        onText)
rocktelebot.GetHandler(tele.OnPhoto,       onPhoto)
rocktelebot.GetHandler(tele.OnDocument,    onDocument)
rocktelebot.GetHandler(tele.OnVoice,       onVoice)
rocktelebot.GetHandler(tele.OnCallback,    onCallback)
rocktelebot.GetHandler(tele.OnUserJoined,  onUserJoined)
rocktelebot.GetHandler(tele.OnQuery,       onInlineQuery)
```

Все доступные entity определены в пакете telebot как константы `tele.OnXxx`.

### Меню команд

Добавь `.WithDesc` к `Command` чтобы команда появилась в меню бота (список при вводе `/`):

```go
rocktelebot.Command("/start",  onStart).WithDesc("Начать работу")
rocktelebot.Command("/help",   onHelp).WithDesc("Показать помощь")
rocktelebot.Command("/cancel", onCancel).WithDesc("Отменить действие")
```

Описания автоматически отправляются в Telegram во время `Init`.

## Мидлвары

Мидлвары выполняются до каждого хендлера. Передаются вторым аргументом в `NewApp` — применяются глобально.

### Встроенные

**Recovery** — перехватывает паники в хендлерах, бот не падает:

```go
rocktelebot.Recovery(func(c tele.Context, err interface{}) {
    log.Error("паника в хендлере", rocklog.Any("err", err))
})

// nil = паника пишется в stderr
rocktelebot.Recovery(nil)
```

**Logger** — вызывается для каждого входящего обновления перед хендлером:

```go
rocktelebot.Logger(func(c tele.Context) {
    log.Info("update",
        rocklog.Int64("user_id", c.Sender().ID),
        rocklog.Str("text", c.Text()),
    )
})
```

### Мидлвар на конкретный хендлер

Передаётся дополнительными аргументами в `Command` или `GetHandler`:

```go
rocktelebot.Command("/admin", onAdmin, adminOnly)
rocktelebot.GetHandler(tele.OnText, onText, rateLimiter)
```

## Клавиатуры

### Inline-клавиатура

Кнопки прикреплены к конкретному сообщению. Нажатие генерирует callback.

```go
kb := rocktelebot.NewInlineKeyboard()

c.Send("Подтвердить?", kb.
    Row(kb.Data("✅ Да", "confirm"), kb.Data("❌ Нет", "cancel")).
    Row(kb.URL("Открыть сайт", "https://example.com")).
    Markup(),
)

// Обработка callback
rocktelebot.GetHandler(tele.OnCallback, func(c tele.Context) error {
    data := c.Callback().Data // "confirm" или "cancel"
    return c.Respond()        // убирает часики с кнопки
})
```

### Reply-клавиатура

Кнопки заменяют обычную клавиатуру пользователя. Нажатие отправляет текст.

```go
kb := rocktelebot.NewReplyKeyboard()

c.Send("Выбери раздел:", kb.
    Row(kb.Text("📋 Список"), kb.Text("➕ Добавить")).
    Row(kb.Text("❌ Отмена")).
    Markup(),
)

// Убрать клавиатуру
c.Send("Готово", rocktelebot.RemoveKeyboard())

// Скрыть после одного нажатия
kb.OneTime()
```

## Polling vs Webhook

| Режим | Когда использовать |
|---|---|
| Long polling (по умолчанию) | Разработка, небольшие боты, простой деплой |
| Webhook | Продакшн, высокая нагрузка, есть HTTPS-сервер |

Long polling не требует инфраструктуры — бот сам подключается к Telegram.
Webhook требует публичного HTTPS-адреса и валидного TLS-сертификата.
