# rockservices

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockservices/README_RU.md — правьте README в репозитории модуля. -->

::: tip Место в GOROCK · Toolkit · Services
Слой сервисов: `pkg/services/services.go` добавляет сервисы в каталог, `cmd/main` поднимает их до apps и останавливает после, любой слой получает сервис по имени. → [Services](/architecture/services), [Main](/architecture/main)
:::

```sh
go get github.com/arzab/gorock-kit/rockservices@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-kit/tree/main/rockservices)

Слой сервисов GOROCK-проекта: один каталог, который создаёт, хранит, выдаёт, инициализирует и останавливает все сервисы из `pkg/services`. Только стандартная библиотека.

Сервисы не зависят от apps: `cmd/main` запускает их до apps и останавливает после.

## Service

```go
type Service interface {
    Name() string                   // ключ в каталоге; по соглашению — константа Name пакета
    Init(ctx context.Context) error // открыть соединения
    Stop() error                    // освободить то, что открыл Init
}
```

```go
// pkg/services/repository/interface.go
const Name = "repository"

type Service interface {
    rockservices.Service
    Insert(ctx context.Context, model any) error
}

// pkg/services/repository/service.go
func NewService(cfg Configs) Service { return &service{cfg: cfg} } // без соединений
func (s *service) Name() string    { return Name }
```

## Регистрация, запуск, использование

```go
// pkg/services/services.go — единственное место подключения
type Configs struct {
    Postgres   postgres.Configs
    Repository repository.Configs
}

func Register(c *rockservices.Catalog, cfg Configs) {
    rockservices.Add(c, postgres.NewService, cfg.Postgres)     // порядок Add = порядок Init
    rockservices.Add(c, repository.NewService, cfg.Repository)
}
```

```go
// cmd/main
services.Register(rockservices.Default(), cfg.Services)
if err := rockservices.Init(ctx); err != nil {
    return err // ни один app не стартует
}
defer rockservices.Stop()

engine := rockengine.NewEngine()
apps.Register(engine, cfg.Apps)
return engine.Run() // сервисы остановятся после всех apps
```

```go
// где угодно
repo := rockservices.Get[repository.Service](repository.Name)
```

## Зависимости между сервисами

- **Сервис владеет дочерним** (repository сам создаёт свой пул): ничего делать не нужно — каталог видит только родителя, а тот сам поднимает и закрывает дочерний.
- **Общая зависимость**: конструктор берёт сервисы, **добавленные раньше него**. Добавляйте зависимости первыми.

```go
func NewService(cfg Configs) Service {
    return &service{db: rockservices.Get[postgres.Service](postgres.Name)} // создан, но ещё не инициализирован
}
```

Зависимость уже создана, но её `Init` выполнится позже, в порядке добавления — сохраните ссылку и пользуйтесь ей начиная с `Init`.

## Соглашения

**Геттер в пакете сервиса** держит имя и тип вместе, чтобы при вызове их нельзя было перепутать:

```go
// pkg/services/repository/interface.go
const Name = "repository"

func Get() Service { return rockservices.Get[Service](Name) }

// delivery: repository.Get().Insert(ctx, order)
```

**Переиспользуемые компоненты берут имя из конфига.** Собственный сервис проекта использует имя-константу. Компонент, который можно добавить несколько раз (две базы PostgreSQL), читает имя из конфига со значением по умолчанию, чтобы второй экземпляр не конфликтовал:

```go
type Configs struct {
    Name string `config:",default=postgres"`
    // ...
}

func (s *service) Name() string { return s.cfg.Name }
```

**Дополнительные аргументы конструктора** (хуки, клиенты, задаваемые в коде) передаются через замыкание:

```go
rockservices.Add(c, func(cfg postgres.Configs) postgres.Service {
    return postgres.NewService(cfg, queryHooks)
}, cfg.Postgres)
```

## Тестирование

Бизнес-код вызывает `rockservices.Get`, а он читает каталог по умолчанию. В тесте соберите каталог с подделками и сделайте его каталогом по умолчанию:

```go
func TestCreateOrder(t *testing.T) {
    c := rockservices.New()
    rockservices.Add(c, func(struct{}) repository.Service { return &fakeRepo{} }, struct{}{})
    prev := rockservices.SetDefault(c)
    t.Cleanup(func() { rockservices.SetDefault(prev) })

    // ... вызвать тестируемый delivery-код
}
```

`SetDefault` — только для тестов; в рабочем коде `Default()` заполняет `services.Register`.

## Правила

- Сервисы хранятся **по имени**; `Get[T]` лишь возвращает нужный интерфейс. Неизвестное имя или не тот тип — паника с именем.
- `Add` сразу вызывает конструктор. Паникует на nil-сервисе, пустом или повторяющемся `Name()`, панике конструктора или вызове после `Init` — ошибки подключения видны при старте.
- `Init` — в порядке добавления; если один упал, уже поднятые останавливаются в обратном порядке, ошибка называет сервис.
- `Stop` — в обратном порядке, собирает все ошибки, безопасен при повторном вызове, ограничен таймаутом (30s, `WithStopTimeout`), который называет зависший сервис.
- Паники в `Init`/`Stop` возвращаются как ошибки.
- `rockservices.New()` — изолированный каталог (`GetFrom[T](c, name)`); `SetDefault` делает его каталогом по умолчанию в тестах.
