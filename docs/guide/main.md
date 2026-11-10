# Main

Main — одинаковая в каждом проекте сборка: загрузить настройки, поднять сервисы, запустить App, по сигналу остановить всё в обратном порядке. Почему именно такой порядок — на странице [Архитектура](/architecture/#_4-main-и-engine).

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

    ctx := context.Background()
    if err := services.Init(ctx, cfg.Services); err != nil {
        return fmt.Errorf("init services: %w", err) // ни один App не стартует
    }
    defer func() {
        if err := services.Stop(); err != nil {
            log.Print(err)
        }
    }()

    return apps.Run(ctx, cfg.Apps) // до SIGINT/SIGTERM или падения App
}
```

`Exec` возвращает ошибку, а не завершает процесс сам. Так `defer` всегда закроет сервисы, а решение «упасть» принимается в одном месте — в `main()`.

---

## Что происходит

1. **Настройки.** [rockconfig](/gorock-kit/rockconfig) читает файл и проверяет его целиком. Опечатка или пропуск — и проект не стартует, назвав все проблемы сразу.
2. **Сервисы.** `services.Init` собирает каталог [rockservices](/gorock-kit/rockservices) и запускает сервисы в заданном порядке. Если один не поднялся, уже запущенные останавливаются, и ошибка называет сервис.
3. **App.** `apps.Run` отдаёт App движку [rockengine](/gorock-kit/rockengine). Движок вызывает `Init` каждого по порядку, затем `Exec` — каждый в своей горутине.
4. **Работа.** Если App упал, движок останавливает остальные, и `Run` возвращает ошибку с именем App. Процесс перезапустит оркестратор (systemd, Kubernetes, Docker). Без оркестратора — `WithRestarts(n, delay)`.
5. **Остановка.** По `SIGINT`/`SIGTERM` движок останавливает App по одному, в обратном порядке: отменяет контекст, вызывает `Stop`, ждёт, пока `Exec` дообработает начатое. Время ограничено — по умолчанию 30 секунд.
6. **Сервисы закрываются** через `defer` в обратном порядке, тоже с таймаутом.

| Чем закончился `Run` | Что вернул |
|---|---|
| сигнал, всё остановилось чисто | `nil` |
| App упал | ошибку с именем App |
| остановка не уложилась в таймаут | ошибку с именем зависшего App |

---

## Разовые команды

Миграции, импорт, CLI-команды выполняют задачу и завершаются — App и движок им не нужны. Поднимите сервисы, сделайте дело, закройте:

```go
// cmd/migrate/main.go
if err := services.Init(ctx, cfg.Services); err != nil {
    return err
}
defer services.Stop()

return runMigrations(ctx, services.Repository())
```

---

## Чего не делать

- **Не писать бизнес-логику и настройку технологий в Main.** Только сборка.
- **Не открывать сервисы вручную в обход `services.Init`.**
- **Не вызывать `log.Fatal` нигде, кроме `main()`.** Возвращайте ошибки — иначе пропадёт аккуратная остановка.

→ Дальше: [Как писать Config](/guide/configs)
