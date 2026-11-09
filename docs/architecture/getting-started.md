# Быстрый старт

Соберём HTTP-сервис задач на PostgreSQL: создать задачу, получить её, отправить событие, проверять базу по расписанию. Готовый результат — [examples/taskapi](https://github.com/arzab/gorock/tree/main/examples/taskapi).

::: info CLI
Генератор слоёв `gorock` [в разработке](/cli/). Пока файлы создаются вручную — это недолго, и так понятнее, что где лежит.
:::

Нужны Go 1.26+ (лучше последний патч-релиз — часть исправлений безопасности приходит с обновлением Go) и Docker (для PostgreSQL).

---

## 1. Модуль и зависимости

```sh
mkdir taskapi && cd taskapi
go mod init github.com/yourname/taskapi

go get github.com/arzab/gorock-kit/rockengine@latest \
       github.com/arzab/gorock-kit/rockconfig@latest \
       github.com/arzab/gorock-kit/rocklog@latest \
       github.com/arzab/gorock-kit/rockfiber@latest \
       github.com/arzab/gorock-kit/rockbun@latest \
       github.com/arzab/gorock-kit/rockbus@latest \
       github.com/arzab/gorock-kit/rockcron@latest \
       github.com/google/uuid@latest
```

---

## 2. Структура

```
taskapi/
├── cmd/
│   ├── main/main.go                      — Configs, загрузка, движок
│   └── apps/apps.go                      — HTTP, шина событий, планировщик
├── internal/
│   ├── delivery/
│   │   ├── endpoints/                    — POST /tasks, GET /tasks/:id
│   │   ├── events/events.go              — обработчик task.created
│   │   └── jobs/jobs.go                  — ping-db раз в минуту
│   └── models/task.go
├── pkg/
│   ├── infra/                            — настройка логгера
│   └── services/                         — repository поверх PostgreSQL
├── configs/configs.yaml
└── migrations/001_tasks.sql
```

Порядок, в котором стоит писать файлы, — снизу вверх: Toolkit → Realm → Engine.

---

## 3. Toolkit: репозиторий и логгер

Репозиторий описывает функционал (сохранить, найти, обновить), а работу с PostgreSQL берёт у [rockbun](/gorock-kit/rockbun). Полный код — на странице [Services](/architecture/services#реализация-сервиса), коротко:

```go
// pkg/services/repository/service.go
func NewService(cfg Configs) Service {
    return &service{db: rockbun.NewService(cfg.Postgres)}
}

func (s *service) Init(ctx context.Context) error { return s.db.Init(ctx) }
func (s *service) Stop() error                    { return s.db.Stop() }

func (s *service) Insert(ctx context.Context, model any) error {
    _, err := s.db.DB().NewInsert().Model(model).Exec(ctx)
    return err
}
```

```go
// pkg/services/services.go
var repo repository.Service

func Repository() repository.Service { return repo }

func Register(e *rockengine.Engine, cfg Configs) {
    repo = repository.NewService(cfg.Repository)
    e.MustRegisterResource("repository", repo)
}
```

Логгер — [rocklog](/gorock-kit/rocklog), `pkg/infra/logs` только переводит настройки из файла в `rocklog.Init` — см. [Infra](/architecture/infra#logs).

---

## 4. Realm: модель и endpoints

```go
// internal/models/task.go
type Task struct {
    bun.BaseModel `bun:"table:tasks" json:"-"`

    ID        string    `bun:"id,pk" json:"id"`
    Title     string    `bun:"title,notnull" json:"title"`
    Done      bool      `bun:"done,notnull" json:"done"`
    CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
}
```

```go
// internal/delivery/endpoints/create_task/endpoint.go
func Endpoint() rockfiber.FiberEndpoint {
    return rockfiber.POST("/tasks", handlers()...)
}
```

Цепочка handlers, параметры, ответ, события и задачи — на странице [Delivery](/architecture/delivery).

---

## 5. Engine: apps и main

```go
// cmd/apps/apps.go
func Register(e *rockengine.Engine, cfg Configs) {
    bus := rockbus.NewApp(cfg.Bus, onBusError, events.Subscriptions()...)
    rockbus.SetDefault(bus)
    e.MustRegister("bus", bus)

    e.MustRegister("cron", rockcron.NewApp(cfg.Cron, onJobError, jobs.All()...))
    e.MustRegister("http", rockfiber.New(cfg.HTTP, endpoints.All()...))
}
```

```go
// cmd/main/main.go
func Exec() error {
    cfg, err := rockconfig.InitFromFile[Configs]("./configs/configs.yaml")
    if err != nil {
        return fmt.Errorf("init configs: %w", err)
    }
    if err := infra.Init(cfg.Infra); err != nil {
        return fmt.Errorf("init infra: %w", err)
    }

    engine := rockengine.NewEngine()
    services.Register(engine, cfg.Services)
    apps.Register(engine, cfg.Apps)
    return engine.Run()
}
```

Подробнее — [Apps](/architecture/apps) и [Main](/architecture/main).

---

## 6. Конфиг

```yaml
# configs/configs.yaml
infra:
  logs:
    level: info
    format: json

services:
  repository:
    postgres:
      host: ${DB_HOST:-localhost}
      user: ${DB_USER:-postgres}
      password: ${DB_PASSWORD}
      database: ${DB_NAME:-tasks}

apps:
  http:
    port: ${PORT:-8080}
    endpoints_path_prefix: /api/v1
    use_trace_id: true
    request_timeout: 5s
    mask_internal_server_error_message: internal server error
```

Каждая секция — `Configs` своего слоя. Подробнее → [Конфигурация](/architecture/configs).

---

## 7. Запуск

```sh
docker run -d --name tasks-db -p 5432:5432 \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=tasks \
  -v "$PWD/migrations:/docker-entrypoint-initdb.d:ro" postgres:16-alpine

DB_PASSWORD=postgres go run ./cmd/main
```

```sh
curl -X POST localhost:8080/api/v1/tasks -H 'Content-Type: application/json' -d '{"title":"write docs"}'
# {"task":{"id":"8121...","title":"write docs","done":false,"created_at":"..."}}

curl localhost:8080/api/v1/tasks/nope
# {"code":404,"status":"Not Found","message":"task not found"}
```

В логе появится событие с тем же trace_id, что у запроса:

```json
{"level":"info","msg":"task created","trace_id":"...","id":"8121...","title":"write docs"}
```

`Ctrl+C` — HTTP-сервер дообработает текущие запросы, затем остановятся планировщик и шина, последним закроется пул PostgreSQL.

Без `DB_PASSWORD` приложение не стартует и скажет, чего не хватает:

```
services.repository.postgres.password: env var DB_PASSWORD is not set
```

---

## Следующий шаг

- [Концепты](/architecture/concepts) — почему всё устроено именно так
- [Toolkit](/architecture/toolkit) → [Realm](/architecture/realm) → [Engine](/architecture/engine) — слои по порядку
