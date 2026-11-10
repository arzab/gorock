# rocklog

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rocklog/README_RU.md — правьте README в репозитории модуля. -->

::: tip Место в GOROCK · Services · логгер
Логгер проекта на `log/slog`. Настраивается сервисом-логгером, который регистрируется в каталоге первым; дальше все пишут через `rocklog.Info`, `rocklog.FromContext(ctx)`. → [Как писать Service](/guide/services#логгер-тоже-сервис)
:::

::: warning Перенесён из gorock-kit v0.2.0 как есть
Код и API — те же, что были в `github.com/arzab/gorock-kit/rocklog@v0.2.0`, путь модуля теперь `github.com/arzab/gorock-catalog/rocklog`. Модуль ещё не переведён на новое ядро: API не меняется; в README ниже логгер настраивается из `pkg/infra` — теперь это `Init` сервиса-логгера.
:::

```sh
go get github.com/arzab/gorock-catalog/rocklog@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-catalog/tree/main/rocklog)

Структурированное логирование для Go с чистым интерфейсом, бэкендом на `log/slog` и поддержкой глобального и инстанс-логгера.

## Обзор

`rocklog` скрывает любой логгер-бэкенд за интерфейсом `Logger`. По умолчанию используется [`log/slog`](https://pkg.go.dev/log/slog) из стандартной библиотеки — без сторонних зависимостей. Все логи структурированы — поля типизированы, строки формата не нужны. Уровень лога всегда присутствует в выводе автоматически.

## Быстрый старт

```go
// Глобальный логгер — готов к использованию без настройки.
rocklog.Info("server started", rocklog.Int("port", 8080))
rocklog.Error("request failed", rocklog.Err(err), rocklog.Str("path", "/api/v1"))

// Настройка глобального логгера.
rocklog.Init(rocklog.Config{
    Level:      rocklog.LevelInfo,
    Format:     rocklog.FormatJSON,
    TimeFormat: time.RFC3339,
})
```

## Config

```go
type Config struct {
    Level      Level     // LevelDebug/Info/Warn/Error/Fatal; нулевое значение = LevelInfo
    Format     Format    // FormatText (по умолчанию) или FormatJSON
    TimeFormat string    // например time.RFC3339, "2006-01-02 15:04:05"; пустая = дефолт бэкенда
    Output     io.Writer // по умолчанию os.Stdout
    Caller     bool      // добавлять file:line в каждую запись
}
```

### Уровни

| Константа    | Когда использовать                         |
|--------------|--------------------------------------------|
| `LevelDebug` | Детальное внутреннее состояние, только dev |
| `LevelInfo`  | Штатные операционные события (по умолчанию)|
| `LevelWarn`  | Неожиданное, но не критичное               |
| `LevelError` | Ошибки, требующие внимания                 |
| `LevelFatal` | Неустранимая ошибка — вызывает `os.Exit(1)`|

### Форматы

| Константа    | Вывод                              | Типичное применение     |
|--------------|------------------------------------|-------------------------|
| `FormatText` | Человекочитаемый цветной вывод     | Локальная разработка    |
| `FormatJSON` | Структурированный JSON построчно   | Prod, GCP, Datadog и др |

## Инстанс-логгер

Используйте `New` когда нужен независимый логгер (например, на компонент или в тестах):

```go
log := rocklog.New(rocklog.Config{
    Level:  rocklog.LevelDebug,
    Format: rocklog.FormatJSON,
})
log.Info("hello")
```

## Глобальный логгер

Пакет поставляется с дефолтным логгером (уровень Info, текстовый формат, stdout). Все пакетные функции делегируют к нему:

```go
rocklog.Info("hello")
rocklog.Debug("query", rocklog.Str("sql", query))
```

Заменить дефолтный логгер при старте:

```go
rocklog.Init(rocklog.Config{
    Level:      rocklog.LevelInfo,
    Format:     rocklog.FormatJSON,
    TimeFormat: time.RFC3339,
    Caller:     true,
})
```

Подключить любую реализацию `Logger`:

```go
rocklog.SetDefault(myZapAdapter)
```

> **Важно:** если использовать `SetDefault(rocklog.New(cfg))` с `Caller: true`, в поле caller будет отображаться файл внутри `rocklog`, а не ваш код. Для логrus-бэкенда с caller используйте `Init(cfg)`.

## Структурированные поля

Каждый метод логирования принимает произвольное количество `Field`. Используйте готовые хелперы:

| Хелпер                         | Описание                              |
|--------------------------------|---------------------------------------|
| `F(key, val)` / `Any(key, val)` | Любое значение                        |
| `Err(err)`                     | Ошибка с ключом `"error"`             |
| `Str(key, val)`                | Строка                                |
| `Int(key, val)`                | int                                   |
| `Int64(key, val)`              | int64                                 |
| `Float64(key, val)`            | float64                               |
| `Bool(key, val)`               | bool                                  |
| `Dur(key, val)`                | `time.Duration` как строка `"1m30s"`  |
| `Time(key, val)`               | `time.Time`                           |
| `Stringer(key, val)`           | Любой `fmt.Stringer`                  |

```go
log.Info("payment processed",
    rocklog.Str("currency", "USD"),
    rocklog.Int("amount_cents", 9900),
    rocklog.Dur("latency", time.Since(start)),
    rocklog.F("metadata", map[string]any{"source": "stripe"}),
)
```

### Логирование структур и map

Любую структуру или map можно передать напрямую через `F`:

```go
log.Info("user created", rocklog.F("user", userStruct))
log.Warn("rate limited", rocklog.F("headers", headersMap))
```

В JSON-формате вложенные объекты сериализуются рекурсивно — удобно для GCP Cloud Logging, Datadog и других систем, индексирующих структурированные поля.

## Контекстные логгеры

### With — постоянные поля

`With` возвращает новый логгер с полями, прикреплёнными ко всем последующим записям:

```go
log := rocklog.New(cfg).With(
    rocklog.Str("service", "payments"),
    rocklog.Str("env", "prod"),
)
log.Info("charge created", rocklog.Int("amount", 100))
// → level=info msg="charge created" service=payments env=prod amount=100
```

Вызовы `With` можно цепочить:

```go
reqLog := log.With(rocklog.Str("request_id", rid))
reqLog.Info("handler started")
reqLog.Error("handler failed", rocklog.Err(err))
```

### Named — тег по компоненту

```go
db := rocklog.Named("database")
db.Info("query executed", rocklog.Dur("took", d))
// → level=info msg="query executed" logger=database took=2ms
```

> Повторный вызов `Named` на уже именованном логгере перезаписывает предыдущее имя.

### Context — логгер в контексте запроса

Передавайте логгер с полями запроса через `context.Context`, а не явным параметром:

```go
// в middleware
ctx = rocklog.WithContext(ctx, rocklog.Str("trace_id", traceId))

// где угодно ниже
rocklog.FromContext(ctx).Info("user created", rocklog.Str("id", id))
// → level=info msg="user created" trace_id=... id=...
```

- `WithContext` добавляет поля к логгеру, уже лежащему в `ctx` (или к логгеру по умолчанию).
- `ContextWithLogger(ctx, l)` кладёт конкретный логгер.
- `FromContext` никогда не возвращает nil — если логгера нет, возвращается логгер по умолчанию.

## Проверка уровня

Для дорогостоящих вычислений перед формированием лога проверяйте активность уровня:

```go
if rocklog.IsEnabled(rocklog.LevelDebug) {
    rocklog.Debug("state dump", rocklog.F("state", computeExpensiveState()))
}
```

## Кастомный бэкенд

Реализуйте интерфейс `Logger` для подключения любой лог-библиотеки:

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    Fatal(msg string, fields ...Field)

    IsEnabled(lvl Level) bool
    With(fields ...Field) Logger
    Named(name string) Logger
}
```

Затем установите как дефолтный:

```go
rocklog.SetDefault(&myZapLogger{})
```

## Caller (файл:строка)

Включите `Caller: true` в `Config` чтобы добавить `file:line` в каждую запись:

```go
rocklog.Init(rocklog.Config{Caller: true})
rocklog.Info("hello")
// → level=info msg=hello caller=main.go:42
```

Логгеры, созданные через `New`, и логгеры возвращённые `With` / `Named` всегда показывают правильное место вызова.

## Поведение Fatal

`Fatal` логирует сообщение и вызывает `os.Exit(1)`. Defer-функции **не выполняются**. Используйте только для действительно неустранимых ошибок при старте приложения, до запуска движка.

> **С rockengine:** не вызывайте `Fatal` внутри app — это пропускает graceful shutdown всех остальных apps (открытые запросы обрываются, события в очереди теряются). Верните ошибку из `Exec` — движок обработает её через политику рестартов / `StopOnFatal`.

## Ограничения

- **`Err(nil)`** логируется как `"error": null`, поле не опускается. Защищайтесь сами: `if err != nil { fields = append(fields, rocklog.Err(err)) }`.
- **`Stringer` с nil-pointer в интерфейсе** может вызвать панику если метод `String()` не обрабатывает nil-ресивер. Не передавайте nil-указатели, обёрнутые в ненулевой интерфейс.
- **`SetDefault` с кастомным Logger и `Caller: true`** будет показывать `rocklog/default.go` как место вызова. Используйте `Init` или реализуйте caller-skip внутри своего адаптера.
- **Конкурентный `Init` / `SetDefault`** во время активного логирования безопасен (`sync.RWMutex`), но лучше вызывать один раз при старте.
- **Формат вывода** — названия уровней (`debug`, `info`, `warning`, `error`, `fatal`) и перезапись полей с тем же ключом такие же, как у прежнего бэкенда на logrus. Отличия: в текстовом формате `msg` идёт перед полями, поля не сортируются; время по умолчанию — RFC 3339 с миллисекундами (формат slog).
