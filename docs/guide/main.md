# Main

Main — одинаковая в каждом проекте сборка: загрузить настройки, открыть сервисы, запустить App, по сигналу остановить всё в обратном порядке. Почему именно такой порядок — на странице [Архитектура](/architecture/#_4-main-и-engine).

```
cmd/main/
└── main.go
```

---

## main.go целиком

```go
// cmd/main/main.go
package main

// Configs собирает настройки всех слоёв; configs/configs.yaml повторяет её.
type Configs struct {
    Services services.Configs
    Apps     apps.Configs
}

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

    services.Register(rockservices.Default(), cfg.Services)
    if err := rockservices.Init(context.Background()); err != nil {
        return fmt.Errorf("init services: %w", err) // ни один App не стартует
    }
    defer func() {
        if err := rockservices.Stop(); err != nil {
            log.Print(err)
        }
    }()

    engine := rockengine.NewEngine()
    apps.Register(engine, cfg.Apps)
    return engine.Run() // до SIGINT/SIGTERM или окончательного падения App
}
```

`Exec` возвращает ошибку, а не завершает процесс сам. Так `defer` всегда закроет сервисы, а решение «упасть» принимается в одном месте — в `main()`.

---

## Что происходит

1. **Настройки.** [rockconfig](/gorock-kit/rockconfig) читает файл и проверяет его целиком. Опечатка или пропуск — и проект не стартует, называя все проблемы сразу.
2. **Сервисы.** [rockservices](/gorock-kit/rockservices) открывает их в порядке регистрации. Если один не открылся, уже открытые закрываются, и ошибка называет сервис.
3. **App.** Движок [rockengine](/gorock-kit/rockengine) вызывает `Init` каждого App по порядку, затем запускает их `Exec` — каждый в своей горутине.
4. **Работа.** Упавший App перезапускается по его `RestartPolicy`. Если он не поднимается, движок останавливает остальные, и `Run` возвращает ошибку с именем App. Оркестратор (systemd, Kubernetes) перезапустит процесс.
5. **Остановка.** По `SIGINT`/`SIGTERM` движок останавливает App по одному, в обратном порядке: отменяет контекст, вызывает `Stop`, ждёт, пока `Exec` дообработает начатое. Время ограничено — по умолчанию 30 секунд.
6. **Сервисы закрываются** в обратном порядке, тоже с таймаутом.

| Чем закончился `Run` | Что вернул |
|---|---|
| сигнал, всё остановилось чисто | `nil` |
| App упал окончательно | ошибку с именем App |
| остановка не уложилась в таймаут | ошибку с именем зависшего App |

Настройки движка: `rockengine.NewEngine().WithShutdownTimeout(d)`, `.WithStopOnFatal(false)`, `.WithFatalHandler(fn)`.

---

## Разовые команды

Миграции, импорт, CLI-команды выполняют задачу и завершаются — движок им не нужен. Откройте сервисы, сделайте дело, закройте:

```go
// cmd/migrate/main.go
services.Register(rockservices.Default(), cfg.Services)
if err := rockservices.Init(ctx); err != nil {
    return err
}
defer rockservices.Stop()

return runMigrations(ctx, services.Orders())
```

---

## Чего не делать

- **Не писать бизнес-логику и настройку технологий в Main.** Только сборка.
- **Не открывать сервисы вручную в обход каталога.**
- **Не вызывать `log.Fatal` нигде, кроме `main()`.** Возвращайте ошибки — иначе пропадёт аккуратная остановка.

→ Дальше: [Как писать Config](/guide/configs)
