# Services

**Toolkit** · `pkg/services/`

Шаблонные обёртки над внешними технологиями. В services вы интерпретируете работу с конкретной системой: `repository` (БД), `rabbit` (очередь), `cache` (Redis), `storage` (S3), `elastic` — что угодно что нужно проекту.

Сервис не знает о бизнес-логике — не импортирует `internal/models`, описывает собственные интерфейсы. Написали один раз — переносите между проектами без изменений.

```
pkg/services/
├── main.go                  — переменные + геттеры + Configs + Init + Stop
└── repository/
    ├── configs.go           — Configs
    ├── interface.go         — Service interface + внутренние типы результатов
    └── service.go           — реализация + NewService(configs Configs) Service
```

---

## Что такое service

Сервис — шаблонная обёртка над технологией. Живёт в `pkg/services/`, является singleton'ом, доступен через геттер из любого слоя.

Ключевое свойство: **сервис независим от проекта**. Он не импортирует `internal/models`, не знает о других сервисах напрямую. Вместо этого описывает контракты через интерфейсы внутри себя или через примитивные типы. Если модели из `internal/` реализуют эти интерфейсы — сервис использует их неявно, не зная о конкретных типах.

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

## `pkg/services/main.go`

Единственная точка входа в слой. Содержит всё необходимое для управления сервисами снаружи.

```go
package services

import (
    "project/pkg/services/repository"
    "project/pkg/services/run_manager"
    "fmt"
)

var (
    repo repository.Service
    rm   run_manager.Service
)

func Repository() repository.Service { return repo }
func RunManager() run_manager.Service { return rm }

type Configs struct {
    Repository repository.Configs `json:"repository"`
    RunManager run_manager.Configs `json:"run_manager"`
}

func Init(configs Configs) error {
    repo = repository.NewService(configs.Repository)
    if err := repo.Init(); err != nil {
        return fmt.Errorf("init repository: %w", err)
    }

    rm = run_manager.NewService(configs.RunManager)
    if err := rm.Init(); err != nil {
        return fmt.Errorf("init run_manager: %w", err)
    }

    return nil
}

func Stop() []error {
    var result []error
    result = append(result, repo.Stop()...)
    result = append(result, rm.Stop()...)
    return result
}
```

---

## Реализация сервиса

Пример — репозиторий базы данных.

### `configs.go`

```go
package repository

type Configs struct {
    Host     string `json:"host"`
    Port     string `json:"port"`
    Name     string `json:"name"`
    User     string `json:"user"`
    Password string `json:"password"`
}
```

### `interface.go`

Аргументы и результаты методов — примитивные типы или интерфейсы, описанные внутри пакета. Не конкретные структуры из `internal/models`.

```go
package repository

type Service interface {
    Init() error
    Stop() []error
    Create(Entity) (err error)
	Update(Entity) (err error)
    Get(Entity) error
    List() ([]Entity, error)
}

// Результат описан внутри пакета — не зависит от internal/models
type Entity interface {
    GetId() string 
	CreateValidation() error 
	UpdateValidation() error
	SearchArguments() map[string][]interface{}
	DeleteValidation() error
}
```

### `service.go`

```go

```

---

## Вложенные сервисы

Сервис может содержать внутри себя другие сервисы — так строится древовидная система зависимостей. Дочерний сервис создаётся в конструкторе родителя, инициализируется в его `Init` и останавливается в `Stop`.

Папка дочернего сервиса располагается внутри папки родительского — структура директорий отражает дерево зависимостей.

```
pkg/services/
└── repository/
    ├── configs.go
    ├── interface.go
    ├── service.go
    └── postgres/            # дочерний сервис внутри родительского
        ├── configs.go
        ├── interface.go
        └── service.go
```

```go
// repository/configs.go
type Configs struct {
    Postgres postgres.Configs `json:"postgres"`
}

// repository/interface.go
type Service interface {
    Init() error
    Stop() []error
    CreateUser(name, email string) (string, error)
}

// repository/service.go
type service struct {
    configs  Configs
    postgres postgres.Service  // дочерний сервис
}

func NewService(configs Configs) Service {
    return &service{
        configs:  configs,
        postgres: postgres.NewService(configs.Postgres),  // создаётся в конструкторе
    }
}

func (s *service) Init() error {
    if err := s.postgres.Init(); err != nil {  // инициализируется в Init родителя
        return fmt.Errorf("init postgres: %w", err)
    }
    return nil
}

func (s *service) Stop() []error {
    return s.postgres.Stop()  // останавливается в Stop родителя
}

func (s *service) CreateUser(name, email string) (string, error) {
    return s.postgres.CreateUser(name, email)
}
```

Снаружи видно только `repository.Service` — детали реализации (что внутри postgres) скрыты. При необходимости можно подменить postgres на mysql, не меняя ни один вызов снаружи.

---

## Независимость через контракты

Сервис описывает что он ожидает, а не что конкретно получает. Если метод принимает сложный объект — описывает интерфейс внутри своего пакета:

```go
// Внутри пакета сервиса описан интерфейс аргумента
type Uploadable interface {
    RunId() string
    SourceDir() string
    ArchiveName() string
}

func (s *service) Upload(u Uploadable) error { ... }
```

Конкретная структура из `internal/models` реализует этот интерфейс — Go проверяет это неявно при передаче. Сервис при этом не знает о `models` ничего.

Упрощение через прямое использование `internal/models` допустимо, если переносимость сервиса не нужна — на усмотрение автора.

---

## Lifecycle

| Метод | Когда | Что делает |
|-------|-------|------------|
| `NewService(configs)` | При `Init` слоя services | Только сохраняет конфиг и создаёт дочерние сервисы |
| `Init() error` | После `NewService` | Открывает соединения, создаёт клиентов, инициализирует дочерние |
| `Stop() []error` | При завершении приложения | Закрывает ресурсы, останавливает дочерние |

`Stop` возвращает `[]error` — нужно попытаться закрыть все ресурсы даже если один упал.

---

## Правила

- **`Service` / `service`** — интерфейс и реализация всегда называются именно так.
- **Конструктор возвращает интерфейс** — `func NewService(configs Configs) Service`.
- **`NewService` — только конфиг и дочерние сервисы.** Никаких соединений и горутин.
- **Один сервис — одна ответственность.** Не смешивать несвязанную логику.
- **Дочерние сервисы — в папке родителя.** Структура директорий отражает дерево зависимостей.
