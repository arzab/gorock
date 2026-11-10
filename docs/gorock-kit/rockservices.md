# rockservices

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockservices/README_RU.md — правьте README в репозитории модуля. -->

::: tip Место в GOROCK · Services
Каталог сервисов: `pkg/services/services.go` регистрирует сервисы под именами, Main открывает их до App и закрывает после, любой код получает сервис по имени через геттер. → [Как писать Service](/guide/services)
:::

```sh
go get github.com/arzab/gorock-kit/rockservices@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-kit/tree/main/rockservices)

Слой сервисов GOROCK-проекта: один каталог, который создаёт, хранит, выдаёт, открывает и закрывает все сервисы из `pkg/services`. Только стандартная библиотека.

Сервис — обёртка над технологией (база, кэш, брокер, внешний API, логгер) или над логикой (разбор файлов, генерация PDF), которой пользуется бизнес. Сервисы не зависят от apps: `cmd/main` открывает их до запуска apps и закрывает после.

## Service

```go
type Service interface {
    Init(ctx context.Context) error // открыть соединения
    Stop() error                    // освободить то, что открыл Init
}
```

Имени у сервиса нет: имя даётся при регистрации. Поэтому один и тот же сервис можно зарегистрировать дважды под разными именами.

```go
// pkg/services/orders/interface.go
type Service interface {
    rockservices.Service
    Insert(ctx context.Context, order any) error
}

// pkg/services/orders/service.go
func NewService(cfg Configs) Service { return &service{cfg: cfg} } // без соединений
```

## Регистрация, запуск, использование

Имена, регистрация и геттеры живут в одном файле — `pkg/services/services.go`:

```go
// pkg/services/services.go
const (
    Logger = "logger"
    Orders = "orders"
)

type Configs struct {
    Logger logger.Configs
    Orders orders.Configs
}

func Register(c *rockservices.Catalog, cfg Configs) {
    rockservices.Add(c, Logger, logger.NewService, cfg.Logger) // порядок Add = порядок Init
    rockservices.Add(c, Orders, orders.NewService, cfg.Orders)
}

// Геттеры: имя и тип написаны один раз.
func OrdersRepository() orders.Service { return rockservices.Get[orders.Service](Orders) }
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
return engine.Run() // сервисы закроются после всех apps
```

```go
// где угодно — обычно в обработчиках
err := services.OrdersRepository().Insert(ctx, order)
```

## Зависимости между сервисами

- **Сервис владеет дочерним** (репозиторий сам создаёт свой пул): ничего делать не нужно. Каталог видит только родителя, а тот сам открывает и закрывает дочерний.
- **Общая зависимость** — тоже сервис в каталоге. Регистрируйте её раньше и передавайте в конструктор через замыкание:

```go
func Register(c *rockservices.Catalog, cfg Configs) {
    rockservices.Add(c, Postgres, postgres.NewService, cfg.Postgres)
    rockservices.Add(c, Orders, func(cfg orders.Configs) orders.Service {
        return orders.NewService(cfg, PostgresDB()) // создан, но ещё не открыт — пользоваться с Init
    }, cfg.Orders)
}
```

Пакет сервиса не знает, под каким именем лежит его зависимость. Всё связывание — в `services.go`.

**Два экземпляра одного сервиса** — две регистрации под разными именами:

```go
rockservices.Add(c, OrdersDB, postgres.NewService, cfg.OrdersDB)
rockservices.Add(c, AnalyticsDB, postgres.NewService, cfg.AnalyticsDB)
```

**Логгер — тоже сервис.** Регистрируйте его первым, чтобы остальные могли писать в лог уже в своём `Init`.

## Тестирование

Бизнес-код вызывает геттеры, а они читают каталог по умолчанию. В тесте соберите каталог с подделками и сделайте его каталогом по умолчанию:

```go
func TestCreateOrder(t *testing.T) {
    c := rockservices.New()
    rockservices.Add(c, services.Orders, func(struct{}) orders.Service { return &fakeOrders{} }, struct{}{})
    prev := rockservices.SetDefault(c)
    t.Cleanup(func() { rockservices.SetDefault(prev) })

    // ... вызвать тестируемый обработчик
}
```

`SetDefault` — только для тестов; в рабочем коде `Default()` заполняет `services.Register`.

## Правила

- Сервисы хранятся **по имени из регистрации**; `Get[T]` лишь возвращает нужный интерфейс. Неизвестное имя или не тот тип — паника с именем.
- `Add` сразу вызывает конструктор. Паникует на пустом или повторяющемся имени, nil-сервисе, панике конструктора или вызове после `Init` — ошибки подключения видны при старте.
- `Init` — в порядке добавления. Если один упал, уже открытые закрываются в обратном порядке, ошибка называет сервис.
- `Stop` — в обратном порядке, собирает все ошибки, безопасен при повторном вызове, ограничен таймаутом (30s, `WithStopTimeout`), который называет зависший сервис.
- Паники в `Init`/`Stop` возвращаются как ошибки.
- `rockservices.New()` — изолированный каталог (`GetFrom[T](c, name)`); `SetDefault` делает его каталогом по умолчанию в тестах.
