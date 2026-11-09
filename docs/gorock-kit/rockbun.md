# rockbun

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockbun/README_RU.md — правьте README в gorock-kit. -->

::: tip Место в GOROCK · Toolkit · Services
Готовый сервис PostgreSQL. Обычно — дочерний сервис внутри `pkg/services/repository`: repository описывает функционал проекта, rockbun даёт пул и запросы. Реализует `rockengine.Resource`. → [Services](/architecture/services)
:::

```sh
go get github.com/arzab/gorock-kit/rockbun@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-kit/tree/main/rockbun)

Обёртка над [bun](https://bun.uptrace.dev/) ORM для PostgreSQL с интеграцией в `rockconfig` и управлением жизненным циклом (`Ping`, `Close`, `RunInTx`) через интерфейс `Service`.

В отличие от `rockredis`, этот пакет **не оборачивает** отдельные операции — query builder bun достаточно богатый для прямого использования. `rockbun` отвечает за настройку соединения, пула и управление транзакциями.

## Быстрый старт

```go
svc := rockbun.NewService(rockbun.Configs{
    DSN: "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
})

// С rockengine: инициализируется до apps, закрывается после них.
rockengine.RegisterResource("db", svc)

// Без движка:
if err := svc.Init(ctx); err != nil { // открывает пул и делает ping
    log.Fatal(err)
}
defer svc.Stop()

db := svc.DB() // *bun.DB — используй для всех запросов
```

## Загрузка конфига из файла

`rockbun.Configs` совместим с `rockconfig.InitFromFile`. Все поля опциональны.

```yaml
# config.yaml — вариант 1: DSN
dsn: "postgres://user:pass@localhost:5432/mydb?sslmode=disable"

# вариант 2: отдельные поля (используются если dsn пустой)
host: "localhost"
port: 5432
user: "myuser"
password: "secret"
database: "mydb"
ssl_mode: "disable"

# настройки пула
max_open_conns: 25
max_idle_conns: 5
conn_max_lifetime: "1h"
conn_max_idle_time: "30m"
```

```go
cfg, err := rockconfig.InitFromFile[rockbun.Configs]("config.yaml")
if err != nil {
    log.Fatal(err)
}

// QueryHooks задаются только в коде
cfg.QueryHooks = []bun.QueryHook{bundebug.NewQueryHook()}

svc := rockbun.NewService(*cfg)
```

## Конфигурация

```go
type Configs struct {
    DSN string  // "postgres://user:pass@host:port/db?sslmode=disable"
                // имеет приоритет над отдельными полями

    Host     string // по умолчанию "localhost"
    Port     int    // по умолчанию 5432
    User     string
    Password string
    Database string
    SSLMode  string // по умолчанию "disable"; в проде используйте "verify-full"

    MaxOpenConns    int           // по умолчанию 25
    MaxIdleConns    int           // по умолчанию 5
    ConnMaxLifetime time.Duration // по умолчанию 1h
    ConnMaxIdleTime time.Duration // по умолчанию 30m

    QueryHooks []bun.QueryHook `config:"-"` // только в коде
}
```

## Запросы

Полный query builder доступен через `DB()`:

```go
db := svc.DB()

// Select
var users []User
err := db.NewSelect().Model(&users).Where("active = ?", true).Scan(ctx)

// Insert
user := &User{Name: "Alice", Email: "alice@example.com"}
_, err = db.NewInsert().Model(user).Exec(ctx)

// Update
_, err = db.NewUpdate().Model(user).Column("name").Where("id = ?", user.ID).Exec(ctx)

// Delete
_, err = db.NewDelete().Model(user).Where("id = ?", user.ID).Exec(ctx)

// Raw SQL
var count int
err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
```

## Транзакции

```go
err := svc.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
    if _, err := tx.NewInsert().Model(&order).Exec(ctx); err != nil {
        return err
    }
    if _, err := tx.NewUpdate().Model(&stock).
        Set("qty = qty - ?", order.Qty).
        Where("id = ?", stock.ID).
        Exec(ctx); err != nil {
        return err
    }
    return nil
})
```

`nil` в opts — уровень изоляции по умолчанию. Для явного задания:

```go
svc.RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}, fn)
```

## Логирование запросов

Встроенный хук `bundebug` для логирования всех запросов:

```go
import "github.com/uptrace/bun/extra/bundebug"

cfg.QueryHooks = []bun.QueryHook{
    bundebug.NewQueryHook(bundebug.WithVerbose(true)),
}
```

Или реализуй `bun.QueryHook` для кастомного логгера:

```go
type myHook struct{}

func (h *myHook) BeforeQuery(ctx context.Context, e *bun.QueryEvent) context.Context {
    return ctx
}

func (h *myHook) AfterQuery(ctx context.Context, e *bun.QueryEvent) {
    log.Info("query",
        rocklog.Str("query", e.Query),
        rocklog.Dur("duration", time.Since(e.StartTime)),
        rocklog.Err(e.Err),
    )
}

cfg.QueryHooks = []bun.QueryHook{&myHook{}}
```

## Миграции

Система миграций bun работает напрямую с `*bun.DB`:

```go
db := svc.DB()
migrator := migrate.NewMigrator(db, migrations.Migrations)
if err := migrator.Init(ctx); err != nil {
    log.Fatal(err)
}
if err := migrator.Migrate(ctx); err != nil {
    log.Fatal(err)
}
```

Подробнее: [bun migrations guide](https://bun.uptrace.dev/guide/migrations.html).

## Ограничения

- `Init` закрывает пул, если `Ping` не прошёл (`DB()` остаётся `nil`); повторный `Init` заменяет и закрывает предыдущий пул. После `Stop` запросы возвращают `sql: database is closed`.
- User, password и database из отдельных полей экранируются, поэтому пароли с `@`, `:`, `/` или `#` работают. DSN, заданный вручную, экранируйте сами.
- `SSLMode` по умолчанию `disable`. В pgdriver `prefer` / `require` шифруют, но **не** проверяют сертификат сервера (а `prefer` не откатывается на plaintext). Для реальной защиты используйте `verify-full`.
- Только PostgreSQL (через `pgdriver`). Для MySQL или SQLite используй `bun.NewDB` напрямую с нужным драйвером.
- `QueryHooks` применяются при создании через `NewService`; после этого добавить или удалить хуки нельзя.
