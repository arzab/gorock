# Services

**Toolkit** · `pkg/services/`

Обёртки над внешними технологиями. В services вы интерпретируете работу с конкретной системой: `repository` (БД), `cache` (Redis), `storage` (S3), `mailer` — что угодно, что нужно проекту.

Сервис не знает о бизнес-логике — не импортирует `internal/models`, описывает собственные интерфейсы. Написали один раз — переносите между проектами без изменений.

```
pkg/services/
├── services.go              — переменные + геттеры + Configs + Register
└── repository/
    ├── configs.go           — Configs
    ├── interface.go         — Service interface + собственные типы и ошибки
    └── service.go           — реализация + NewService(cfg Configs) Service
```

---

## Что такое service

Сервис — обёртка над технологией. Живёт в `pkg/services/`, является singleton'ом, доступен через геттер из любого слоя.

Ключевое свойство: **сервис независим от проекта**. Он не импортирует `internal/models`, не знает о других сервисах напрямую. Вместо этого описывает контракты через интерфейсы внутри себя или принимает `any` там, где технология умеет работать с любой структурой (как ORM). Если модели из `internal/` реализуют эти интерфейсы — сервис использует их неявно, не зная о конкретных типах.

---

## Именование

Интерфейс всегда называется `Service`, структура-реализация — `service` (строчная).

```go
type Service interface { ... }  // публичный интерфейс
type service struct { ... }     // приватная реализация
```

Это исключает дублирование имени пакета при обращении снаружи:

```go
repository.Service   // не repository.Repository
file_storage.Service // не file_storage.FileStorage
```

---

## `pkg/services/services.go`

Единственная точка входа в слой. Создаёт сервисы и передаёт их жизненный цикл движку.

```go
package services

import (
    "github.com/arzab/gorock-kit/rockengine"

    "github.com/yourname/taskapi/pkg/services/repository"
)

type Configs struct {
    Repository repository.Configs
}

var repo repository.Service

// Repository — геттер для всех слоёв.
func Repository() repository.Service { return repo }

// Register создаёт сервисы и регистрирует их как ресурсы движка:
// они инициализируются раньше всех apps и останавливаются после них.
func Register(e *rockengine.Engine, cfg Configs) {
    repo = repository.NewService(cfg.Repository)
    e.MustRegisterResource("repository", repo)
}
```

Своих `Init` и `Stop` у слоя больше нет — порядок, ошибки и таймауты берёт на себя [rockengine](/gorock-kit/rockengine). Геттер возвращает сервис сразу после `Register`, а пользоваться им можно с момента, когда движок вызвал его `Init`, — то есть в любом `Init` или `Exec` приложения.

---

## Реализация сервиса

Пример — репозиторий поверх PostgreSQL. Работу с базой даёт готовый [rockbun](/gorock-kit/rockbun), repository описывает функционал проекта.

### `configs.go`

```go
package repository

import "github.com/arzab/gorock-kit/rockbun"

type Configs struct {
    Postgres rockbun.Configs // готовый конфиг из gorock-kit: host, port, user, password, pool...
}
```

### `interface.go`

```go
package repository

import (
    "context"
    "errors"
)

// ErrNotFound — собственная ошибка пакета, delivery проверяет её через errors.Is.
var ErrNotFound = errors.New("repository: not found")

// Service хранит любую bun-модель. Про internal/models он не знает:
// вызывающий передаёт указатель на свою структуру.
type Service interface {
    // Init / Stop делают сервис ресурсом движка (rockengine.Resource).
    Init(ctx context.Context) error
    Stop() error

    Ping(ctx context.Context) error
    Insert(ctx context.Context, model any) error
    GetByID(ctx context.Context, model any, id string) error
    Update(ctx context.Context, model any) error
}
```

### `service.go`

```go
package repository

import (
    "context"
    "database/sql"
    "errors"

    "github.com/arzab/gorock-kit/rockbun"
)

type service struct {
    db rockbun.Service // дочерний сервис из gorock-kit
}

// NewService — только конфиг и дочерние сервисы. Никаких соединений.
func NewService(cfg Configs) Service {
    return &service{db: rockbun.NewService(cfg.Postgres)}
}

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

---

## Вложенные сервисы

Сервис может содержать внутри себя другие сервисы — так строится дерево зависимостей. В примере выше `rockbun.Service` — дочерний сервис `repository`: создаётся в конструкторе родителя, инициализируется в его `Init`, останавливается в его `Stop`. Движку виден только `repository`.

Свои дочерние сервисы кладите в папку родителя — структура директорий отражает дерево зависимостей:

```
pkg/services/
└── repository/
    ├── configs.go
    ├── interface.go
    ├── service.go
    └── migrations/          — дочерний сервис внутри родительского
        ├── configs.go
        ├── interface.go
        └── service.go
```

Снаружи видно только `repository.Service`. Поменяли PostgreSQL на другую базу — ни один вызов снаружи не меняется.

---

## Независимость через контракты

Сервис описывает, что он ожидает, а не что конкретно получает. Если методу нужно поведение объекта — описывает интерфейс внутри своего пакета:

```go
// Внутри пакета сервиса описан интерфейс аргумента
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
| `NewService(cfg)` | `services.Register` | Только сохраняет конфиг и создаёт дочерние сервисы |
| `Init(ctx) error` | движок, до всех apps | Открывает соединения, проверяет доступность, инициализирует дочерние |
| `Stop() error` | движок, после всех apps | Закрывает ресурсы, останавливает дочерние |

Если `Init` упал — движок закроет уже поднятые сервисы, и приложение не стартует. Если `Stop` нужно закрыть несколько ресурсов — закройте все и верните `errors.Join(...)`.

---

## Правила

- **`Service` / `service`** — интерфейс и реализация всегда называются именно так.
- **Конструктор возвращает интерфейс** — `func NewService(cfg Configs) Service`.
- **`NewService` — только конфиг и дочерние сервисы.** Никаких соединений и горутин.
- **`Init(ctx) error` / `Stop() error`** — сервис должен быть ресурсом движка.
- **Один сервис — одна ответственность.** Не смешивать несвязанную логику.
- **Дочерние сервисы — в папке родителя.** Структура директорий отражает дерево зависимостей.
