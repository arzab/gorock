# Быстрый старт

Соберём HTTP-сервис задач на PostgreSQL: создать задачу и получить её.

::: info CLI
Генератор слоёв `gorock` [в разработке](/cli/). Пока файлы создаются вручную — это недолго, и так понятнее, что где лежит.
:::

Нужны Go 1.26+ (лучше последний патч-релиз — часть исправлений безопасности приходит с обновлением Go) и Docker (для PostgreSQL).

---

## 1. Модуль и зависимости

```sh
mkdir taskapi && cd taskapi
go mod init github.com/yourname/taskapi

# ядро
go get github.com/arzab/gorock-kit/rockengine@latest \
       github.com/arzab/gorock-kit/rockconfig@latest \
       github.com/arzab/gorock-kit/rockservices@latest \
       github.com/arzab/gorock-kit/rockhandlers@latest
# каталог: HTTP, PostgreSQL, логгер
go get github.com/arzab/gorock-catalog/rockhttp@latest \
       github.com/arzab/gorock-catalog/rockbun@latest \
       github.com/arzab/gorock-catalog/rocklog@latest \
       github.com/google/uuid@latest
```

---

## 2. Структура

```
taskapi/
├── cmd/
│   ├── main/main.go                      — Configs, загрузка, сервисы, движок
│   └── apps/apps.go                      — HTTP App
├── internal/
│   ├── delivery/
│   │   ├── endpoints/                    — POST /tasks, GET /tasks/{id}
│   │   └── middleware/log.go             — логирование ошибок
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

Репозиторий описывает функционал (сохранить, найти, обновить), а работу с PostgreSQL берёт у [rockbun](/catalog/rockbun). Полный код — на странице [Services](/architecture/services#реализация-сервиса), коротко:

```go
// pkg/services/repository/interface.go
const Name = "repository"

type Service interface {
    rockservices.Service
    Insert(ctx context.Context, model any) error
    GetByID(ctx context.Context, model any, id string) error
}

func Get() Service { return rockservices.Get[Service](Name) }
```

```go
// pkg/services/repository/service.go
func NewService(cfg Configs) Service { return &service{db: rockbun.NewService(cfg.Postgres)} }

func (s *service) Name() string                    { return Name }
func (s *service) Init(ctx context.Context) error { return s.db.Init(ctx) }
func (s *service) Stop() error                    { return s.db.Stop() }
```

```go
// pkg/services/services.go
type Configs struct {
    Repository repository.Configs
}

func Register(c *rockservices.Catalog, cfg Configs) {
    rockservices.Add(c, repository.NewService, cfg.Repository)
}
```

Логгер — [rocklog](/catalog/rocklog), `pkg/infra/logs` только переводит настройки из файла в `rocklog.Init` — см. [Infra](/architecture/infra#logs).

---

## 4. Realm: модель и маршруты

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
// internal/delivery/endpoints/create_task/route.go
type Params struct {
    Title string `json:"title"`
}

func (p *Params) Validate() error {
    if strings.TrimSpace(p.Title) == "" {
        return rockhandlers.Invalid("title is required")
    }
    return nil
}

type Result struct {
    Task *models.Task `json:"task"`
}

var Route = rockhandlers.Chain("tasks.create", func(c *rockhandlers.Ctx[*Params, *Result]) error {
    task := &models.Task{ID: uuid.NewString(), Title: c.Params.Title}
    if err := repository.Get().Insert(c, task); err != nil {
        return fmt.Errorf("insert task: %w", err) // → 500, текст скрыт
    }
    c.Result.Task = task
    return nil
})
```

```go
// internal/delivery/endpoints/get_task/route.go
type Params struct {
    ID string `path:"id"`
}

var Route = rockhandlers.Chain("tasks.get", func(c *rockhandlers.Ctx[*Params, *models.Task]) error {
    err := repository.Get().GetByID(c, c.Result, c.Params.ID)
    if errors.Is(err, repository.ErrNotFound) {
        return rockhandlers.NotFound("task not found") // → 404
    }
    return err
})
```

```go
// internal/delivery/endpoints/endpoints.go
func All() []rockhttp.Endpoint {
    return []rockhttp.Endpoint{
        {Method: "POST", Path: "/tasks", Handler: create_task.Route, Status: http.StatusCreated},
        {Method: "GET", Path: "/tasks/{id}", Handler: get_task.Route},
    }
}
```

```go
// internal/delivery/middleware/log.go
func Log(c rockhandlers.Context) error {
    err := c.Next()
    if err != nil {
        rocklog.Error("request failed", rocklog.Str("route", c.Route()), rocklog.Err(err))
    }
    return err
}
```

Цепочки шагов, переиспользуемые шаги, Locals и группы — на странице [Delivery](/architecture/delivery).

---

## 5. Engine: apps и main

```go
// cmd/apps/apps.go
type Configs struct {
    HTTP rockhttp.Config
}

func Register(e *rockengine.Engine, cfg Configs) {
    e.MustRegister("http", rockhttp.New(cfg.HTTP, endpoints.All()...).Use(middleware.Log))
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

    services.Register(rockservices.Default(), cfg.Services)
    if err := rockservices.Init(context.Background()); err != nil {
        return fmt.Errorf("init services: %w", err)
    }
    defer rockservices.Stop()

    engine := rockengine.NewEngine()
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
    prefix: /api/v1
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
# 201 {"task":{"id":"8121...","title":"write docs","done":false,"created_at":"..."}}

curl -X POST localhost:8080/api/v1/tasks -H 'Content-Type: application/json' -d '{"title":""}'
# 400 {"code":400,"status":"Bad Request","kind":"invalid","message":"title is required"}

curl localhost:8080/api/v1/tasks/nope
# 404 {"code":404,"status":"Not Found","kind":"not_found","message":"task not found"}
```

`Ctrl+C` — HTTP-сервер дообработает текущие запросы, затем закроется пул PostgreSQL.

Без `DB_PASSWORD` приложение не стартует и скажет, чего не хватает:

```
services.repository.postgres.password: env var DB_PASSWORD is not set
```

---

## Следующий шаг

- [Концепты](/architecture/concepts) — почему всё устроено именно так
- [Toolkit](/architecture/toolkit) → [Realm](/architecture/realm) → [Engine](/architecture/engine) — слои по порядку
