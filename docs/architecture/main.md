# Main

**Engine** · `cmd/main/`

Финальный шаг. Toolkit готов, Realm описан, Apps собраны. В `cmd/main` вы загружаете конфиг, поднимаете сервисы, запускаете apps — и проект работает.

```
cmd/main/
└── main.go     — Configs, main(), Exec()
```

Если файл разрастается — `Configs` выносится в `configs.go`.

---

## Агрегированный Configs

`Configs` собирает конфиги всех слоёв в одну структуру. Файл конфига повторяет её один к одному:

```go
// cmd/main/main.go
type Configs struct {
    Infra    infra.Configs
    Services services.Configs
    Apps     apps.Configs
}
```

Теги `json` не нужны: [rockconfig](/gorock-kit/rockconfig) сам переводит имена полей в `snake_case` (`Services` → `services`, `HTTPServer` → `http_server`). Открыл файл — увидел всю архитектуру приложения. Подробнее → [Конфигурация](/architecture/configs).

---

## Exec

```go
// cmd/main/main.go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/arzab/gorock-kit/rockconfig"
    "github.com/arzab/gorock-kit/rockengine"
    "github.com/arzab/gorock-kit/rockservices"

    "github.com/yourname/taskapi/cmd/apps"
    "github.com/yourname/taskapi/pkg/infra"
    "github.com/yourname/taskapi/pkg/services"
)

func main() {
    if err := Exec(); err != nil {
        log.Fatal(err)
    }
}

func Exec() error {
    path := "./configs/configs.yaml"
    if len(os.Args) > 1 {
        path = os.Args[1]
    }

    cfg, err := rockconfig.InitFromFile[Configs](path)
    if err != nil {
        return fmt.Errorf("init configs: %w", err)
    }

    // infra первой: всё, что дальше, может логировать.
    if err := infra.Init(cfg.Infra); err != nil {
        return fmt.Errorf("init infra: %w", err)
    }

    // services — до apps: HTTP не примет запрос, пока не подключился репозиторий.
    services.Register(rockservices.Default(), cfg.Services)
    if err := rockservices.Init(context.Background()); err != nil {
        return fmt.Errorf("init services: %w", err)
    }
    defer rockservices.Stop() // после остановки всех apps

    engine := rockengine.NewEngine()
    apps.Register(engine, cfg.Apps)
    return engine.Run() // блокирует до SIGINT/SIGTERM или падения App
}
```

`Exec` возвращает ошибку, а не вызывает `log.Fatal` сам: так `defer rockservices.Stop()` и graceful shutdown всегда отрабатывают, а решение «упасть» принимается в одном месте — в `main()`.

---

## Порядок

```
infra.Init → rockservices.Init → rockengine.Run (apps) → rockservices.Stop
```

Порядок не случаен:

- **`infra` первая** — логгер нужен всем уже при старте. Infra не останавливается: ей нечего закрывать.
- **`services` — отдельный слой до apps.** [rockservices](/gorock-kit/rockservices) поднимает сервисы в порядке добавления; если один упал, уже поднятые закрываются, и ни один app не стартует. Останавливаются сервисы **после** `engine.Run`, то есть когда все apps уже дообработали последние запросы.
- **`apps` — в порядке регистрации.** [rockengine](/gorock-kit/rockengine) останавливает их в обратном порядке, по одному: отмена контекста → `Stop` → ожидание `Exec`. Регистрируйте HTTP-сервер последним — он остановится первым, пока шина событий, в которую он публикует, ещё работает.

Движок управляет только apps: их может быть несколько, и он отвечает за жизненный цикл каждого, а не всей программы. Поэтому сервисы — не его забота, а отдельный слой в `cmd/main`.

---

## Когда процесс завершается

`engine.Run()` возвращается, когда:

| Причина | Что возвращает `Run` |
|---|---|
| `SIGINT` / `SIGTERM` | `nil`, если всё остановилось чисто |
| App упал окончательно (исчерпал рестарты) | ошибку с именем App — оркестратор (systemd, Kubernetes) перезапустит процесс |
| Остановка не уложилась в таймаут (30s по умолчанию) | ошибку с именем зависшего App |

После этого `defer rockservices.Stop()` закрывает сервисы в обратном порядке, тоже с таймаутом (30s, `rockservices.WithStopTimeout`).

Настройки движка — `rockengine.NewEngine().WithShutdownTimeout(...)`, `.WithStopOnFatal(false)`, `.WithFatalHandler(...)`. Подробнее → [rockengine](/gorock-kit/rockengine).
