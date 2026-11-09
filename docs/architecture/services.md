# Services

**Toolkit** · `pkg/services/`

Обёртки над внешними технологиями. В services вы описываете работу с конкретной системой: `repository` (БД), `cache` (Redis), `storage` (S3), `mailer` — что угодно, что нужно проекту.

Сервис не знает о бизнес-логике — не импортирует `internal/models`, описывает собственные интерфейсы. Написали один раз — переносите между проектами без изменений.

```
pkg/services/
├── services.go              — Configs + Register: добавляет сервисы в каталог
└── repository/
    ├── configs.go           — Configs
    ├── interface.go         — Name, Service interface, Get(), собственные ошибки
    └── service.go           — реализация + NewService(cfg Configs) Service
```

---

## Что такое service

Сервис — обёртка над технологией. Живёт в `pkg/services/`, создаётся один раз и доступен по имени из любого слоя.

Жизненным циклом сервисов управляет [rockservices](/gorock-kit/rockservices) — это **отдельный слой**, не часть движка. `cmd/main` поднимает сервисы до apps и закрывает после них: HTTP не примет запрос, пока не подключился репозиторий, и не потеряет его, пока дообрабатывает последние запросы.

Ключевое свойство: **сервис независим от проекта**. Он не импортирует `internal/models`. Вместо этого описывает контракты через интерфейсы внутри себя или принимает `any` там, где технология умеет работать с любой структурой (как ORM). Если модели из `internal/` реализуют эти интерфейсы — сервис использует их неявно.

---

## Контракт

```go
type Service interface {
    Name() string                   // ключ в каталоге
    Init(ctx context.Context) error // открыть соединения
    Stop() error                    // освободить то, что открыл Init
}
```

Свой интерфейс сервиса встраивает `rockservices.Service` и добавляет функционал.

---

## Именование

Интерфейс всегда называется `Service`, структура-реализация — `service` (строчная), имя в каталоге — константа `Name` пакета.

```go
const Name = "repository"
type Service interface { ... }  // публичный интерфейс
type service struct { ... }     // приватная реализация
```

Это исключает дублирование имени пакета при обращении снаружи: `repository.Service`, а не `repository.Repository`.

---

## `pkg/services/services.go`

Единственная точка подключения слоя. Добавляет сервисы в каталог — порядок `Add` и есть порядок `Init`:

```go
package services

import (
    "github.com/arzab/gorock-kit/rockservices"

    "github.com/yourname/taskapi/pkg/services/repository"
)

type Configs struct {
    Repository repository.Configs
}

func Register(c *rockservices.Catalog, cfg Configs) {
    rockservices.Add(c, repository.NewService, cfg.Repository)
}
```

`cmd/main` вызывает `services.Register(rockservices.Default(), cfg.Services)`, затем `rockservices.Init(ctx)` и `defer rockservices.Stop()` → [Main](/architecture/main).

---

## Реализация сервиса

Пример — репозиторий поверх PostgreSQL. Работу с базой даёт [rockbun](/catalog/rockbun) из каталога, repository описывает функционал проекта.

### `configs.go`

```go
package repository

import "github.com/arzab/gorock-catalog/rockbun"

type Configs struct {
    Postgres rockbun.Configs // host, port, user, password, pool...
}
```

### `interface.go`

```go
package repository

import (
    "context"
    "errors"

    "github.com/arzab/gorock-kit/rockservices"
)

const Name = "repository"

// ErrNotFound — собственная ошибка пакета, delivery проверяет её через errors.Is.
var ErrNotFound = errors.New("repository: not found")

// Service хранит любую bun-модель. Про internal/models он не знает:
// вызывающий передаёт указатель на свою структуру.
type Service interface {
    rockservices.Service

    Ping(ctx context.Context) error
    Insert(ctx context.Context, model any) error
    GetByID(ctx context.Context, model any, id string) error
    Update(ctx context.Context, model any) error
}

// Get — геттер: имя и тип вместе, перепутать нельзя.
func Get() Service { return rockservices.Get[Service](Name) }
```

### `service.go`

```go
package repository

import (
    "context"
    "database/sql"
    "errors"

    "github.com/arzab/gorock-catalog/rockbun"
)

type service struct {
    db rockbun.Service // дочерний сервис: repository им владеет
}

// NewService — только конфиг и дочерние сервисы. Никаких соединений.
func NewService(cfg Configs) Service {
    return &service{db: rockbun.NewService(cfg.Postgres)}
}

func (s *service) Name() string                    { return Name }
func (s *service) Init(ctx context.Context) error { return s.db.Init(ctx) }
func (s *service) Stop() error                    { return s.db.Stop() }

func (s *service) Ping(ctx context.Context) error { return s.db.Ping(ctx) }

func (s *service) Insert(ctx context.Context, model any) error {
    _, err := s.db.DB().NewInsert().Model(model).Exec(ctx)
    return err
}

func (s *service) GetByID(ctx context.Context, model any, id string) error {
    err := s.db.DB().NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
    if errors.Is(err, sql.ErrNoRows) {
        return ErrNotFound
    }
    return err
}

func (s *service) Update(ctx context.Context, model any) error {
    res, err := s.db.DB().NewUpdate().Model(model).WherePK().Exec(ctx)
    if err != nil {
        return err
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return ErrNotFound
    }
    return nil
}
```

Сервис экспортирует **функционал** — сохранить, найти, обновить, — а не `SaveTask`, `GetUserById`. Завтра нужно хранить `Order` — repository не меняется.

Использование из любого слоя: `repository.Get().Insert(ctx, task)`.

---

## Зависимости между сервисами

**Сервис владеет дочерним.** В примере выше `rockbun.Service` — дочерний сервис `repository`: создаётся в конструкторе родителя, поднимается в его `Init`, закрывается в его `Stop`. Каталогу виден только `repository`. Свои дочерние сервисы кладите в папку родителя — структура директорий отражает дерево зависимостей:

```
pkg/services/
└── repository/
    ├── configs.go
    ├── interface.go
    ├── service.go
    └── migrations/          — дочерний сервис внутри родительского
```

**Общая зависимость** — два сервиса пользуются одним пулом. Тогда пул — отдельный сервис в каталоге, и конструктор берёт его по имени. Можно брать только сервисы, **добавленные раньше**, поэтому зависимости добавляются первыми:

```go
func Register(c *rockservices.Catalog, cfg Configs) {
    rockservices.Add(c, postgres.NewService, cfg.Postgres)     // сначала зависимость
    rockservices.Add(c, repository.NewService, cfg.Repository) // потом тот, кто её использует
}

// repository/service.go
func NewService(cfg Configs) Service {
    return &service{db: postgres.Get()} // создан, но ещё не поднят — пользоваться начиная с Init
}
```

---

## Независимость через контракты

Сервис описывает, что он ожидает, а не что конкретно получает. Если методу нужно поведение объекта — описывает интерфейс внутри своего пакета:

```go
type Uploadable interface {
    ObjectKey() string
    Content() io.Reader
}

func (s *service) Upload(ctx context.Context, u Uploadable) error { ... }
```

Конкретная структура из `internal/models` реализует этот интерфейс — Go проверяет это неявно при передаче. Сервис при этом не знает о `models` ничего.

Упрощение через прямое использование `internal/models` допустимо, если переносимость сервиса не нужна, — на усмотрение автора.

---

## Lifecycle

| Метод | Кто вызывает | Что делает |
|-------|-------|------------|
| `NewService(cfg)` | `rockservices.Add` в `services.Register` | Только сохраняет конфиг и создаёт дочерние сервисы |
| `Init(ctx) error` | `rockservices.Init` в `cmd/main`, до apps | Открывает соединения, проверяет доступность, поднимает дочерние |
| `Stop() error` | `rockservices.Stop` в `cmd/main`, после apps | Закрывает ресурсы, останавливает дочерние |

Если `Init` упал — уже поднятые сервисы закрываются в обратном порядке, и ни один app не стартует. Если `Stop` должен закрыть несколько ресурсов — закройте все и верните `errors.Join(...)`.

---

## Тестирование

Бизнес-код берёт сервисы через `rockservices.Get`, а тот читает каталог по умолчанию. В тесте соберите каталог с подделками:

```go
c := rockservices.New()
rockservices.Add(c, func(struct{}) repository.Service { return &fakeRepo{} }, struct{}{})
prev := rockservices.SetDefault(c)
t.Cleanup(func() { rockservices.SetDefault(prev) })
```

---

## Правила

- **`Service` / `service` / `Name`** — интерфейс, реализация и имя в каталоге всегда называются так.
- **Конструктор возвращает интерфейс** — `func NewService(cfg Configs) Service`.
- **`NewService` — только конфиг и дочерние сервисы.** Никаких соединений и горутин.
- **Геттер `Get()` в пакете сервиса** — имя и тип вместе.
- **Один сервис — одна ответственность.** Не смешивать несвязанную логику.
- **Дочерние сервисы — в папке родителя.** Общие зависимости — отдельными сервисами, добавленными раньше.
