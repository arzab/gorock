# Как писать Service

Сервис — обёртка над технологией или логикой, которой пользуется бизнес: база, кэш, брокер, внешний API, логгер, разбор файлов. Зачем он нужен и чем отличается от остальных слоёв — на странице [Архитектура](/architecture/#_1-services). Здесь — как его написать, зарегистрировать и использовать.

Пример — сервис заказов поверх PostgreSQL и логгер.

```
pkg/services/
├── services.go            имена, Configs, Register, геттеры
├── orders/
│   ├── configs.go         настройки сервиса
│   ├── interface.go       что сервис умеет
│   └── service.go         как он это делает
└── logger/
    └── service.go
```

---

## 1. Что сервис умеет — `interface.go`

Начните с интерфейса: он отвечает на вопрос, какой функционал нужен проекту. Интерфейс встраивает `rockservices.Service` — это `Init` и `Stop`, жизненный цикл сервиса.

```go
// pkg/services/orders/interface.go
package orders

// ErrNotFound — ошибка пакета; обработчики проверяют её через errors.Is.
var ErrNotFound = errors.New("orders: not found")

type Service interface {
    rockservices.Service

    Insert(ctx context.Context, order any) error
    GetByID(ctx context.Context, order any, id string) error
}
```

Методы называйте по тому, что нужно проекту. `UsersRepository` с `FindByEmail` — нормально, если это то, что вам нужно. Форма сервиса свободная. Правило одно: сервис знает, **как** что-то сделать, но не решает, **когда и зачем** — это решает обработчик.

## 2. Настройки — `configs.go`

```go
// pkg/services/orders/configs.go
package orders

type Configs struct {
    Postgres rockbun.Configs // готовые настройки клиента PostgreSQL из каталога
}
```

Секция попадёт в общий файл конфига как `services.orders.postgres`. Правила полей — в [Как писать Config](/guide/configs).

## 3. Как он это делает — `service.go`

```go
// pkg/services/orders/service.go
package orders

type service struct {
    db rockbun.Service // дочерний сервис: orders им владеет
}

// NewService — только сохраняет настройки и создаёт дочерние сервисы. Никаких соединений.
func NewService(cfg Configs) Service {
    return &service{db: rockbun.NewService(cfg.Postgres)}
}

func (s *service) Init(ctx context.Context) error { return s.db.Init(ctx) } // открыть
func (s *service) Stop() error                    { return s.db.Stop() }    // закрыть

func (s *service) Insert(ctx context.Context, order any) error {
    _, err := s.db.DB().NewInsert().Model(order).Exec(ctx)
    return err
}

func (s *service) GetByID(ctx context.Context, order any, id string) error {
    err := s.db.DB().NewSelect().Model(order).Where("id = ?", id).Scan(ctx)
    if errors.Is(err, sql.ErrNoRows) {
        return ErrNotFound
    }
    return err
}
```

- **Конструктор возвращает интерфейс** и ничего не открывает. Соединения — в `Init`: каталог вызовет его при старте, до того как проект начнёт принимать запросы.
- **`Stop` закрывает всё, что открыл `Init`.** Если ресурсов несколько, закройте все и верните `errors.Join(...)`.
- **Дочерний сервис** (здесь — клиент PostgreSQL [rockbun](/catalog/rockbun)) живёт внутри родителя. Родитель открывает и закрывает его сам, а каталогу виден только родитель.
- **Сервис не импортирует `internal/`.** Модели приходят как `any` (ORM сам читает теги), или сервис объявляет у себя интерфейс, который модель выполняет.

## 4. Регистрация и геттеры — `services.go`

Все сервисы проекта регистрируются в одном файле. Здесь же — их имена и геттеры.

```go
// pkg/services/services.go
package services

// Имена задаются при регистрации и живут здесь.
const (
    LoggerName = "logger"
    OrdersName = "orders"
)

type Configs struct {
    Logger logger.Configs `config:",omitempty"`
    Orders orders.Configs
}

// Register добавляет сервисы в каталог. Порядок добавления = порядок Init:
// логгер первым, чтобы остальные могли писать в лог уже при запуске.
func Register(c *rockservices.Catalog, cfg Configs) {
    rockservices.Add(c, LoggerName, logger.NewService, cfg.Logger)
    rockservices.Add(c, OrdersName, orders.NewService, cfg.Orders)
}

// Orders — геттер для всех слоёв.
func Orders() orders.Service { return rockservices.Get[orders.Service](OrdersName) }
```

Использование — откуда угодно, обычно из обработчика:

```go
err := services.Orders().Insert(c, order)
```

`cmd/main` вызывает `services.Register`, затем открывает каталог до запуска App и закрывает после → [Main](/guide/main).

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

## Зависимости между сервисами

**Сервис владеет зависимостью** — как `orders` своим клиентом PostgreSQL. Создайте её в конструкторе, открывайте и закрывайте в своих `Init` и `Stop`.

**Зависимость общая** — например, один пул на несколько сервисов. Тогда она — отдельный сервис в каталоге. Зарегистрируйте её раньше и передайте в конструктор через замыкание:

```go
rockservices.Add(c, PostgresName, postgres.NewService, cfg.Postgres)
rockservices.Add(c, OrdersName, func(cfg orders.Configs) orders.Service {
    return orders.NewService(cfg, Postgres()) // создан, но ещё не открыт — пользоваться с Init
}, cfg.Orders)
```

Пакет сервиса не знает, под каким именем лежит его зависимость. Всё связывание — в `services.go`.

**Два экземпляра одного сервиса** — две регистрации под разными именами:

```go
rockservices.Add(c, OrdersDBName, postgres.NewService, cfg.OrdersDB)
rockservices.Add(c, AnalyticsDBName, postgres.NewService, cfg.AnalyticsDB)
```

---

## Тестирование

Обработчики берут сервисы через геттеры, а геттеры читают каталог по умолчанию. В тесте соберите каталог с подделкой и сделайте его каталогом по умолчанию:

```go
type fakeOrders struct{ inserted []any }

func (f *fakeOrders) Init(context.Context) error { return nil }
func (f *fakeOrders) Stop() error                { return nil }
func (f *fakeOrders) Insert(_ context.Context, o any) error {
    f.inserted = append(f.inserted, o)
    return nil
}
func (f *fakeOrders) GetByID(context.Context, any, string) error { return ordersvc.ErrNotFound }

func TestCreate(t *testing.T) {
    c := rockservices.New()
    rockservices.Add(c, services.OrdersName, func(struct{}) ordersvc.Service { return &fakeOrders{} }, struct{}{})
    prev := rockservices.SetDefault(c)
    t.Cleanup(func() { rockservices.SetDefault(prev) })
    // ... вызвать обработчик
}
```

---

## Чего не делать

- **Не открывать соединения в конструкторе.** Только в `Init` — иначе их некому закрыть, если старт упал.
- **Не принимать бизнес-решения в сервисе.** Списывать ли деньги и что делать при отказе, решает обработчик.
- **Не создавать слой use case поверх сервисов.** Обработчик вызывает сервисы напрямую.
- **Не передавать сервисы через конструкторы обработчиков.** Для этого есть каталог и геттеры.
- **Не хранить имя сервиса в самом сервисе.** Имя — свойство регистрации.

→ Дальше: [Как писать Handler](/guide/handlers) · API: [rockservices](/gorock-kit/rockservices)
