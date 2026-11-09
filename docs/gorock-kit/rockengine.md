# rockengine

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockengine/README_RU.md — правьте README в репозитории модуля. -->

::: tip Место в GOROCK · Engine
Сердце `cmd/main`: запускает и останавливает **apps** — HTTP, шину событий, планировщик. Сервисы — отдельный слой, его поднимает [rockservices](./rockservices) до движка. Контракт `App` — это интерфейс слоя Apps. → [Main](/architecture/main), [Apps](/architecture/apps)
:::

```sh
go get github.com/arzab/gorock-kit/rockengine@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-kit/tree/main/rockengine)

Движок для оркестрирования жизненного цикла приложений на Go.

## Обзор

`rockengine` управляет жизненным циклом нескольких именованных приложений: последовательная инициализация, параллельное исполнение, политики перезапуска и graceful shutdown.

**Жизненный цикл app:**

```
Init → Exec → Stop
```

- **Init** — вызывается последовательно в порядке регистрации; инициализирует ресурсы.
- **Exec** — выполняется параллельно в отдельной горутине; должен уважать отмену `ctx`.
- **Stop** — вызывается при остановке; должен разблокировать `Exec` и дождаться текущих операций.

## Реализация App

```go
type App interface {
    Init(ctx context.Context) error
    Exec(ctx context.Context) error
    Stop() []error
}
```

**Пример:**

```go
type WorkerApp struct {
    ticker *time.Ticker
    done   chan struct{}
}

func (a *WorkerApp) Init(_ context.Context) error {
    a.ticker = time.NewTicker(time.Second)
    a.done = make(chan struct{})
    return nil
}

func (a *WorkerApp) Exec(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-a.ticker.C:
            fmt.Println("tick")
        }
    }
}

func (a *WorkerApp) Stop() []error {
    a.ticker.Stop()
    close(a.done)
    return nil
}
```

## Быстрый старт

```go
package main

import (
    "log"
    "github.com/arzab/gorock-kit/rockengine"
)

func main() {
    rockengine.Register("worker", &WorkerApp{})
    rockengine.Register("http",   &HTTPApp{})

    if err := rockengine.Run(); err != nil {
        log.Fatal(err)
    }
}
```

`Run` блокируется до получения `SIGINT` / `SIGTERM` (или падения app, см. ниже), после чего gracefully останавливает все apps в обратном порядке регистрации.

## Политика перезапуска

По умолчанию app, чей `Exec` вернул ошибку (или запаниковал), не перезапускается: он помечается как `failed`, у него вызывается `Stop`, и **весь движок останавливается** — `Run` возвращает ошибку с именем app. Так оркестратор (systemd, Kubernetes) перезапустит процесс, потерявший компонент, вместо того чтобы оставить его полуживым.

Для автоматического перезапуска:

```go
rockengine.Register("worker", &WorkerApp{}, rockengine.RestartPolicy{
    MaxRetries: -1,               // -1 = бесконечно, 0 = не перезапускать (по умолчанию), N = до N раз
    Delay:      2 * time.Second,
    ResetAfter: time.Minute,      // запуск, проработавший >=1m, сбрасывает счётчик попыток; 0 = не сбрасывать
    OnFatal: func(err error) {   // вызывается когда app помечен failed (переопределяет глобальный обработчик)
        log.Printf("worker упал окончательно: %v", err)
    },
})
```

- Перезапуск — это `Stop → Delay → Init → Exec`. Каждому успешному `Init` соответствует ровно один `Stop`.
- Паника в `Exec` считается падением и подчиняется политике. Паники в `Init`, `Stop` и обработчиках фатальных ошибок перехватываются и возвращаются как ошибки.
- Что бы `Exec` ни вернул после отмены его `ctx` (`nil`, `ctx.Err()`, `http.ErrServerClosed`, ...) — это чистая остановка, а не падение.
- Без `ResetAfter` `MaxRetries` считает падения за всё время жизни процесса.

Чтобы движок продолжал работу при падении app, используйте `SetStopOnFatal(false)` / `WithStopOnFatal(false)` и отслеживайте падения через обработчик фатальных ошибок или `AppStatuses`.

## Собственный Engine

Для большего контроля создайте `Engine` напрямую:

```go
engine := rockengine.NewEngine().
    WithShutdownTimeout(15 * time.Second).
    WithStopOnFatal(false). // остальные apps продолжают работу, если один упал
    WithFatalHandler(func(name string, err error) {
        log.Printf("app %s упал: %v", name, err)
    })

engine.Register("worker", &WorkerApp{}, rockengine.RestartPolicy{MaxRetries: 3, Delay: time.Second})
engine.Register("http",   &HTTPApp{})

if err := engine.Run(); err != nil {
    log.Fatal(err)
}
```

## Оркестрирование

Отдельными apps можно управлять во время работы движка, не затрагивая остальные:

```go
// Остановить один app
rockengine.StopApp("worker")

// Перезапустить один app (Stop → Init → Exec)
rockengine.RestartApp("worker")

// Посмотреть статус одного app
info, err := rockengine.AppStatus("worker")
fmt.Println(info.State, info.Retries, info.Err)

// Посмотреть статус всех apps
for _, info := range rockengine.AppStatuses() {
    fmt.Printf("%s: %s\n", info.Name, info.State)
}
```

## Состояния App

| Состояние    | Описание                                             |
|--------------|------------------------------------------------------|
| `idle`       | Зарегистрирован, ещё не запущен                      |
| `running`    | `Exec` активен                                       |
| `restarting` | Ожидание перед следующей попыткой запуска            |
| `stopped`    | Остановлен чисто                                     |
| `failed`     | Исчерпан лимит попыток или повторный `Init` не удался |

## Справочник API

### Пакетные функции (движок по умолчанию)

| Функция                                           | Описание                                   |
|---------------------------------------------------|--------------------------------------------|
| `Register(name, app, policy?)`                   | Зарегистрировать app                       |
| `Run() error`                                     | Запустить и заблокироваться                |
| `RunContext(ctx) error`                           | Запустить с контекстом                     |
| `StopApp(name) error`                             | Остановить один app                        |
| `RestartApp(name) error`                          | Перезапустить один app                     |
| `AppStatus(name) (AppInfo, error)`                | Статус одного app                          |
| `AppStatuses() []AppInfo`                         | Статус всех apps                           |
| `Shutdown() []error`                              | Остановить все apps                        |
| `SetShutdownTimeout(d)`                           | Таймаут остановки (по умолчанию 30s)       |
| `SetFatalHandler(fn)`                             | Обработчик фатального падения app          |
| `SetStopOnFatal(bool)`                            | Останавливать движок при падении app (по умолчанию `true`) |

### Методы Engine

Те же, что и выше, плюс:

| Метод                         | Описание                        |
|-------------------------------|---------------------------------|
| `NewEngine() *Engine`         | Создать новый движок            |
| `WithShutdownTimeout(d)`      | Задать таймаут остановки        |
| `WithFatalHandler(fn)`        | Задать обработчик фатальных ошибок |
| `WithStopOnFatal(bool)`       | Останавливаться при падении app (по умолчанию `true`) |

## Контракт остановки

Apps останавливаются **по одному** в обратном порядке регистрации. Для каждого app:

1. отменяется его `ctx` — `Exec` должен прекратить приём новой работы и завершить текущие операции;
2. сразу после вызывается `Stop`, чтобы разблокировать блокирующий `Exec` (например, `server.Shutdown`);
3. движок ждёт возврата `Exec` и только потом переходит к предыдущему app.

Так HTTP-сервер, зарегистрированный после шины событий, полностью останавливается до того, как шина будет затронута.

- `Stop` не должен форсированно обрывать текущую работу. Таймаут остановки (по умолчанию 30s) покрывает всю последовательность, включая вызовы `Stop`; по таймауту `Run` возвращает ошибку с именем зависшего app.
- Движок вызывает `Stop` не более одного раза на каждый успешный `Init`, но `Stop` может выполняться одновременно с `Exec` — он должен быть к этому готов.
