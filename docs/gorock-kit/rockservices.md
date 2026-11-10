# rockservices

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockservices/README_RU.md — правьте README в репозитории модуля. -->

::: tip Место в GOROCK · Services
Каталог сервисов: `pkg/services/services.go` объявляет типизированные ключи и регистрирует сервисы, Main открывает их до App и закрывает после, любой код получает сервис по ключу: `services.Orders.Get()`. → [Как писать Service](/guide/services)
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

Имени у сервиса нет: его имя — это **ключ**, под которым он зарегистрирован. Поэтому один и тот же сервис можно зарегистрировать дважды под двумя ключами.

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

Ключи и регистрация живут в одном файле — `pkg/services/services.go`. Ключ типизирован: тип сервиса записан один раз, в объявлении ключа, а `Add` сверяет с ним тип конструктора при компиляции.

```go
// pkg/services/services.go
var (
    Logger = rockservices.NewKey[logger.Service]("logger")
    Orders = rockservices.NewKey[orders.Service]("orders")
)

type Configs struct {
    Logger logger.Configs
    Orders orders.Configs
}

func Register(c *rockservices.Catalog, cfg Configs) {
    rockservices.Add(c, Logger, logger.NewService, cfg.Logger) // порядок Add = порядок Init
    rockservices.Add(c, Orders, orders.NewService, cfg.Orders)
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
return engine.Run() // сервисы закроются после всех apps
```

```go
// где угодно — обычно в обработчиках
err := services.Orders.Get().Insert(ctx, order) // типизировано, без приведения
```

**Забытая регистрация ловится при старте.** Каталог по умолчанию в `Init` проверяет, что каждый объявленный ключ добавлен, и называет пропущенные — проект не стартует, а не падает на первом запросе. Каталоги из `New()` эту проверку не делают: в тестах туда кладут только нужные подделки.

## Зависимости между сервисами

- **Сервис владеет дочерним** (репозиторий сам создаёт свой пул): ничего делать не нужно. Каталог видит только родителя, а тот сам открывает и закрывает дочерний.
- **Общая зависимость** — тоже сервис в каталоге. Регистрируйте её раньше и передавайте в конструктор **значение, которое вернул `Add`**:

```go
func Register(c *rockservices.Catalog, cfg Configs) {
    pg := rockservices.Add(c, Postgres, postgres.NewService, cfg.Postgres)
    rockservices.Add(c, Orders, func(cfg orders.Configs) orders.Service {
        return orders.NewService(cfg, pg) // создан, но ещё не открыт — пользоваться с Init
    }, cfg.Orders)
}
```

Не вызывайте `key.Get()` внутри конструктора: он читает каталог по умолчанию, а не тот, в который идёт регистрация. С `Register(rockservices.New(), cfg)` сервис получил бы чужую зависимость или панику. Пакет сервиса не знает, под каким ключом лежит его зависимость: всё связывание — в `services.go`.

**Два экземпляра одного сервиса** — два ключа и две регистрации:

```go
rockservices.Add(c, OrdersDB, postgres.NewService, cfg.OrdersDB)
rockservices.Add(c, AnalyticsDB, postgres.NewService, cfg.AnalyticsDB)
```

**Логгер — тоже сервис.** Регистрируйте его первым, чтобы остальные могли писать в лог уже в своём `Init`.

## Тестирование

Бизнес-код вызывает `key.Get()`, а он читает каталог по умолчанию. В тесте соберите каталог с подделками и сделайте его каталогом по умолчанию:

```go
func TestCreateOrder(t *testing.T) {
    c := rockservices.New()
    rockservices.Add(c, services.Orders, func(struct{}) orders.Service { return &fakeOrders{} }, struct{}{})
    prev := rockservices.SetDefault(c)
    t.Cleanup(func() { rockservices.SetDefault(prev) })

    // ... вызвать тестируемый обработчик
}
```

`SetDefault` — только для тестов; в рабочем коде `Default()` заполняет `services.Register`. Каталог по умолчанию один на процесс, поэтому тесты, которые его подменяют, **нельзя запускать с `t.Parallel()`**.

## Правила

- Сервисы хранятся **по ключу**. `key.Get()` читает каталог по умолчанию, `key.From(c)` — указанный. Сервиса нет в каталоге — паника с именем (с каталогом по умолчанию после успешного `Init` такого не бывает).
- `Init` каталога по умолчанию сначала проверяет, что все объявленные ключи добавлены, и называет пропущенные. Каталоги из `New()` не проверяют.
- `Add` сразу вызывает конструктор. Паникует на nil-ключе, пустом или повторяющемся имени (в том числе у двух разных ключей), nil-сервисе, панике конструктора или вызове после `Init` — ошибки подключения видны при старте.
- `Init` — в порядке добавления, один раз: повторный вызов возвращает `ErrAlreadyStarted`. Если один упал, уже открытые закрываются в обратном порядке, ошибка называет сервис.
- `Stop` — в обратном порядке, собирает все ошибки, безопасен при повторном вызове, ограничен таймаутом (30s, `WithStopTimeout`), который называет зависший сервис.
- Паники в `Init`/`Stop` возвращаются как ошибки.
- `rockservices.New()` — изолированный каталог (`key.From(c)`); `SetDefault` делает его каталогом по умолчанию в тестах.
