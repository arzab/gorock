# Main

**Engine** · `cmd/main/`

Финальный шаг. Toolkit готов, Realm описан, Apps собраны. В `cmd/main` вы загружаете конфиг, регистрируете слои в движке — и проект работает.

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
    "fmt"
    "log"
    "os"

    "github.com/arzab/gorock-kit/rockconfig"
    "github.com/arzab/gorock-kit/rockengine"

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

    engine := rockengine.NewEngine()
    services.Register(engine, cfg.Services) // ресурсы: init первыми, stop последними
    apps.Register(engine, cfg.Apps)         // apps: старт после ресурсов, остановка в обратном порядке

    return engine.Run() // блокирует до SIGINT/SIGTERM или падения App
}
```

`Exec` возвращает ошибку, а не вызывает `log.Fatal` сам: так graceful shutdown всегда отрабатывает, а решение «упасть» принимается в одном месте — в `main()`.

---

## Порядок

Порядок не случаен:

- **`infra` первая** — логгер нужен всем уже при старте. Services и Apps должны уметь логировать ошибки инициализации. Infra не регистрируется в движке: ей нечего останавливать.
- **`services` — ресурсы движка.** [rockengine](/gorock-kit/rockengine) инициализирует их раньше всех apps и закрывает после всех apps. HTTP-сервер не начнёт принимать запросы, пока не подключился репозиторий, и не потеряет его, пока дообрабатывает последние запросы.
- **`apps` — в порядке регистрации.** Останавливаются в обратном порядке, по одному: отмена контекста → `Stop` → ожидание `Exec`. Регистрируйте HTTP-сервер последним — он остановится первым, пока шина событий, в которую он публикует, ещё работает.

Если `Init` любого компонента падает, движок закрывает всё, что уже успел поднять, и `Run` возвращает ошибку с именем компонента.

---

## Когда процесс завершается

`engine.Run()` возвращается, когда:

| Причина | Что возвращает `Run` |
|---|---|
| `SIGINT` / `SIGTERM` | `nil`, если всё остановилось чисто |
| App упал окончательно (исчерпал рестарты) | ошибку с именем App — оркестратор (systemd, Kubernetes) перезапустит процесс |
| Остановка не уложилась в таймаут (30s по умолчанию) | ошибку с именем зависшего компонента |

Настройки движка — `rockengine.NewEngine().WithShutdownTimeout(...)`, `.WithStopOnFatal(false)`, `.WithFatalHandler(...)`. Подробнее → [rockengine](/gorock-kit/rockengine).
