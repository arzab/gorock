# Как писать Service

Сервис — обёртка над технологией или логикой, которой пользуется бизнес: база, кэш, брокер, внешний API, логгер, разбор файлов. Зачем он нужен — на странице [Архитектура](/architecture/#_1-services). Здесь — как его написать, собрать в каталог и использовать.

Весь код ниже — из примера [taskapi](https://github.com/arzab/gorock/tree/main/examples/taskapi).

```
pkg/services/
├── services.go            названия, Configs, Init, Stop, геттеры
├── repository/
│   ├── configs.go         настройки сервиса
│   ├── interface.go       что сервис умеет
│   └── service.go         как он это делает
└── logger/
    └── service.go
```

---

## 1. Что сервис умеет — `interface.go`

Начните с интерфейса: какой функционал нужен проекту. Интерфейс встраивает `rockservices.Service` — это `Init` и `Stop`, жизненный цикл сервиса.

```go
// pkg/services/repository/interface.go
package repository

// ErrNotFound — ошибка пакета; обработчики проверяют её через errors.Is.
var ErrNotFound = errors.New("repository: not found")

type Service interface {
    rockservices.Service

    Insert(ctx context.Context, model any) error
    GetByID(ctx context.Context, model any, id string) error
}
```

Методы называйте по тому, что нужно проекту. `UsersRepository` с `FindByEmail` — нормально, если это то, что вам нужно. Правило одно: сервис знает, **как** что-то сделать, но не решает, **когда и зачем** — это решают обработчики.

## 2. Настройки — `configs.go`

```go
// pkg/services/repository/configs.go
type Configs struct {
    Postgres rockbun.Configs // готовые настройки клиента PostgreSQL из каталога
}
```

## 3. Как он это делает — `service.go`

```go
// pkg/services/repository/service.go
type service struct {
    db rockbun.Service // дочерний сервис: репозиторий открывает его сам
}

// NewService — только сохраняет настройки и создаёт дочерние сервисы. Никаких соединений.
func NewService(cfg Configs) Service {
    return &service{db: rockbun.NewService(cfg.Postgres)}
}

func (s *service) Init(ctx context.Context) error { return s.db.Init(ctx) } // открыть
func (s *service) Stop() error                    { return s.db.Stop() }    // закрыть

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
```

- **Конструктор ничего не открывает.** Соединения открываются в `Init`, при старте, до того как проект начнёт принимать запросы.
- **`Stop` закрывает всё, что открыл `Init`.**
- **Сервис не импортирует `internal/`.** Модели приходят как `any` (ORM сам читает теги), или сервис объявляет у себя интерфейс, который модель выполняет.

## 4. Каталог — `services.go`

Все сервисы проекта собираются в одном файле: названия, конфиги, порядок запуска и геттеры.

```go
// pkg/services/services.go
package services

const (
    LoggerName     = "logger"
    RepositoryName = "repository"
)

type Configs struct {
    Logger     logger.Configs `config:",omitempty"`
    Repository repository.Configs
}

var catalog *rockservices.Catalog

// Init создаёт сервисы из их конфигов и запускает в нужном порядке:
// логгер первым, чтобы остальные могли писать в лог уже при запуске.
func Init(ctx context.Context, cfg Configs) error {
    catalog = rockservices.New(map[string]rockservices.Service{
        LoggerName:     logger.NewService(cfg.Logger),
        RepositoryName: repository.NewService(cfg.Repository),
    })
    return catalog.Init(ctx, LoggerName, RepositoryName)
}

// Stop останавливает сервисы в обратном порядке.
func Stop() error { return catalog.Stop() }

// Repository — геттер для всех слоёв.
func Repository() repository.Service { return catalog.Get(RepositoryName).(repository.Service) }
```

Использование — откуда угодно, обычно из обработчика:

```go
err := services.Repository().Insert(c, task)
```

`cmd/main` вызывает `services.Init` до запуска App и `services.Stop` после → [Main](/guide/main).

Ошибка в названии видна сразу: `Init` паникует на неизвестном имени ещё до того, как что-то запустилось.

---

## Логгер — тоже сервис

Логгер настраивается из конфига, поэтому он сервис со своим `Init`. Пишут в лог через функции [rocklog](/catalog/rocklog): `rocklog.Info(...)`, `rocklog.FromContext(ctx).Error(...)`.

```go
// pkg/services/logger/service.go
type Configs struct {
    Level  string `config:",default=info"` // debug | info | warn | error
    Format string `config:",default=json"` // json | text
}

type Service interface{ rockservices.Service }

type service struct{ cfg Configs }

func NewService(cfg Configs) Service { return &service{cfg: cfg} }

func (s *service) Init(context.Context) error {
    levels := map[string]rocklog.Level{
        "debug": rocklog.LevelDebug, "info": rocklog.LevelInfo,
        "warn": rocklog.LevelWarn, "error": rocklog.LevelError,
    }
    level, ok := levels[s.cfg.Level]
    if !ok {
        return fmt.Errorf("unknown log level %q", s.cfg.Level)
    }
    format := rocklog.FormatJSON
    if s.cfg.Format == "text" {
        format = rocklog.FormatText
    }
    rocklog.Init(rocklog.Config{Level: level, Format: format})
    return nil
}

func (s *service) Stop() error { return nil }
```

---

## Дочерние сервисы

**Дочерний сервис, который родитель открывает сам** — как репозиторий свой клиент PostgreSQL. Создайте его в конструкторе родителя, открывайте и закрывайте в `Init` и `Stop` родителя. В каталоге его нет.

**Дочерний сервис, который открывается снаружи** — например, один пул на несколько сервисов. Создайте его заранее, положите в каталог и передайте в конструкторы тех, кто им пользуется:

```go
pg := postgres.NewService(cfg.Postgres)
catalog = rockservices.New(map[string]rockservices.Service{
    PostgresName:   pg,
    RepositoryName: repository.NewService(cfg.Repository, pg),
})
return catalog.Init(ctx, PostgresName, RepositoryName) // сначала зависимость
```

**Два экземпляра одного сервиса** — два названия в каталоге:

```go
OrdersDBName:    postgres.NewService(cfg.OrdersDB),
AnalyticsDBName: postgres.NewService(cfg.AnalyticsDB),
```

---

## Тестирование

Обработчики берут сервисы через геттеры, а геттеры читают каталог пакета `services`. Для тестов дайте функцию, которая его подменяет:

```go
// pkg/services/services.go
func UseCatalog(c *rockservices.Catalog) (restore func()) {
    prev := catalog
    catalog = c
    return func() { catalog = prev }
}
```

```go
// internal/handlers/tasks/fake_test.go
func useFake(t *testing.T) *fakeRepository {
    fake := &fakeRepository{tasks: map[string]*models.Task{}}
    t.Cleanup(services.UseCatalog(rockservices.New(map[string]rockservices.Service{
        services.RepositoryName: fake,
    })))
    return fake
}
```

Каталог один на пакет, поэтому тесты, которые его подменяют, **нельзя запускать с `t.Parallel()`**. Полные тесты — в [taskapi](https://github.com/arzab/gorock/tree/main/examples/taskapi/internal/handlers/tasks).

---

## Чего не делать

- **Не открывать соединения в конструкторе.** Только в `Init` — иначе их некому закрыть, если старт упал.
- **Не принимать бизнес-решения в сервисе.** Когда и зачем его вызывать, решают обработчики.
- **Не создавать слой use case поверх сервисов.** Обработчик вызывает сервисы напрямую.
- **Не передавать сервисы через конструкторы обработчиков.** Для этого есть каталог и геттеры.

→ Дальше: [Как писать Handler](/guide/handlers) · API: [rockservices](/gorock-kit/rockservices)
