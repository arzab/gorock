# Infra

**Toolkit** · `pkg/infra/`

Инфраструктурные примитивы, которые обслуживают сам проект. Инициализируются первыми — раньше services и apps.

```
pkg/infra/
├── infra.go       — Configs + Init
├── logs/          — настройка глобального логгера
├── tracer/        — distributed tracing
└── vars/          — глобальные переменные и feature flags
```

---

## Что такое infra

Обёртки над системами, которые обслуживают **сам проект**, — логирование, трассировка, глобальные настройки. В отличие от `pkg/services`, которыми пользуется бизнес-логика, `pkg/infra` нужна всем слоям независимо от того, что делает приложение.

Infra не обязана следовать паттерну сервиса. Примитив может не иметь `Configs`, `Init` и интерфейса — достаточно набора функций пакета. Форма определяется потребностью конкретного примитива.

---

## `pkg/infra/infra.go`

```go
package infra

import (
    "fmt"

    "github.com/yourname/taskapi/pkg/infra/logs"
)

type Configs struct {
    Logs logs.Configs `config:",omitempty"`
}

// Init настраивает примитивы проекта. Вызывается первым, до движка.
func Init(cfg Configs) error {
    if err := logs.Init(cfg.Logs); err != nil {
        return fmt.Errorf("init logs: %w", err)
    }
    return nil
}
```

Infra не регистрируется в движке: у логгера нет соединений, которые нужно закрывать. Если у примитива они есть (например, экспортёр трейсов), сделайте его ресурсом — как сервис.

---

## logs

Сам логгер уже есть — [rocklog](/gorock-kit/rocklog). Он сам по себе интерфейс (`rocklog.Logger`), поэтому отдельная обёртка над ним не нужна. Задача `pkg/infra/logs` — перевести настройки из файла конфига в `rocklog.Init`:

```go
package logs

import (
    "fmt"

    "github.com/arzab/gorock-kit/rocklog"
)

// Configs — то, что можно задать в файле.
type Configs struct {
    Level  string `config:",default=info"` // debug | info | warn | error
    Format string `config:",default=json"` // json | text
    Caller bool   `config:",omitempty"`
}

func Init(cfg Configs) error {
    levels := map[string]rocklog.Level{
        "debug": rocklog.LevelDebug, "info": rocklog.LevelInfo,
        "warn": rocklog.LevelWarn, "error": rocklog.LevelError,
    }
    level, ok := levels[cfg.Level]
    if !ok {
        return fmt.Errorf("unknown log level %q", cfg.Level)
    }

    format := rocklog.FormatJSON
    if cfg.Format == "text" {
        format = rocklog.FormatText
    }

    rocklog.Init(rocklog.Config{Level: level, Format: format, Caller: cfg.Caller})
    return nil
}
```

Дальше все слои пишут через rocklog:

```go
rocklog.Info("task created", rocklog.Str("id", task.ID))

// логгер запроса с trace_id — передаётся вместе с ctx в сервисы и события
ctx = rocklog.WithContext(ctx, rocklog.Str("trace_id", traceId))
rocklog.FromContext(ctx).Error("insert failed", rocklog.Err(err))
```

::: warning
Не вызывайте `rocklog.Fatal` внутри apps и сервисов: он завершает процесс сразу и пропускает graceful shutdown всех остальных компонентов. Верните ошибку — движок решит, перезапустить компонент или остановить приложение.
:::

---

## vars

Значения, которые не принадлежат ни одному сервису, — имя окружения, feature flags, shared-параметры.

```go
package vars

type Configs struct {
    Env          string          `config:",default=dev"`
    FeatureFlags map[string]bool `config:",omitempty"`
}

var cfg Configs

func Init(c Configs) { cfg = c }

func Get() Configs { return cfg }
```

Использование из любого слоя:

```go
if vars.Get().Env == "prod" { ... }
```

---

## tracer

Distributed tracing — трассировка запроса через все слои. Конкретная реализация (OpenTelemetry и др.) скрыта за функциями пакета. Готового модуля в gorock-kit пока нет.

---

## События внутри процесса

Раньше здесь жил `eventbus`. Теперь это готовая шина [rockbus](/gorock-kit/rockbus) — и она не infra, а **App**: у неё есть собственная работа (воркеры, очереди) и жизненный цикл. Регистрируется в `cmd/apps`, обработчики описываются в `internal/delivery/events`, публиковать можно из любого delivery:

```go
rockbus.PublishAsync(ctx, rockbus.Event{Topic: events.TaskCreated, Payload: *task})
```

В микросервисах шина внутри процесса обычно не нужна — event-driven выносится на уровень брокера (Kafka, NATS, Pub/Sub).

→ [Apps](/architecture/apps) · [Delivery](/architecture/delivery)

---

## Правила

- **Infra инициализируется первой.** Логгер нужен всем уже при старте остальных слоёв.
- **Доступ через функции пакета**, не через геттеры как в services: `rocklog.Info(...)`, `vars.Get()`.
- **Форма свободная.** Не нужно натягивать паттерн сервиса, если примитив в нём не нуждается.
- **Нет привязки к предметной области.** Если примитив знает о конкретных сущностях проекта — он не infra.
