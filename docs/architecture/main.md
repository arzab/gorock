# Main

**Engine** · `cmd/main/`

Дирижёр всей системы. Единственное место в проекте которое знает обо всём — собирает конфиги всех слоёв, инициализирует их в правильном порядке, запускает Apps и корректно останавливает при завершении.

```
cmd/main/
├── configs.go  — Configs: агрегирует конфиги всех слоёв
├── exec.go     — Exec(): парсинг конфига, инициализация слоёв, запуск Apps
└── main.go     — main(): запускает Exec(), логирует ошибку
```

---

## Агрегированный Configs

`configs.go` собирает конфиги всех слоёв в одну структуру. Путь к файлу либо дефолтный, либо принимается аргументом при запуске:

```go
// cmd/main/configs.go
type Configs struct {
    Infra    infra.Configs    `json:"infra"`
    Services services.Configs `json:"services"`
    Apps     apps.Configs     `json:"apps"`
}
```

Открыл файл — увидел всю архитектуру приложения. Подробнее о конфигах → [Конфигурация](/architecture/configs).

---

## Порядок инициализации

```
infra.Init → services.Init → apps.Init → apps.Exec
                                               ↓
                                         [сигнал ОС]
                                               ↓
                             apps.Stop → services.Stop → infra.Stop
```

Порядок не случаен:

- **`infra` первая** — логгер нужен всем уже при старте. Services и Apps должны уметь логировать ошибки инициализации.
- **`services` после infra** — могут использовать логгер и другие инфраструктурные примитивы при подключении к БД или очередям.
- **`apps` последние** — зависят от готовности services. HTTP-сервер не должен принимать запросы пока репозиторий не подключён.

Остановка — зеркало инициализации, в обратном порядке.

---

## Пример

```go
// cmd/main/main.go
func main() {
    if err := Exec(); err != nil {
        log.Fatal(err)
    }
}
```

```go
// cmd/main/exec.go
func Exec() error {
    configsPath := "./configs/configs.json"
    if len(os.Args) > 1 {
        configsPath = os.Args[1]
    }

    cfg, err := configs.InitFromFile[Configs](configsPath)
    if err != nil {
        return fmt.Errorf("init configs: %w", err)
    }

    if err = infra.Init(cfg.Infra); err != nil {
        return fmt.Errorf("init infra: %w", err)
    }
    defer infra.Stop()

    if err = services.Init(cfg.Services); err != nil {
        return fmt.Errorf("init services: %w", err)
    }
    defer services.Stop()

    return apps.Exec(cfg.Apps)
}
```
