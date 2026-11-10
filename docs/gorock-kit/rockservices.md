# rockservices

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockservices/README_RU.md — правьте README в репозитории модуля. -->

::: tip Место в GOROCK · Services
Каталог сервисов: `pkg/services/services.go` собирает `map` из названий и сервисов, запускает их в нужном порядке до App и закрывает после; любой код получает сервис через геттер, например `services.Repository()`. → [Как писать Service](/guide/services)
:::

```sh
go get github.com/arzab/gorock-kit/rockservices@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-kit/tree/main/rockservices)

Слой сервисов GOROCK-проекта: каталог, в котором сервисы хранятся по названию и запускаются и останавливаются в том порядке, который вы задали. Только стандартная библиотека.

Сервис — обёртка над технологией (база, кэш, брокер, внешний API, логгер) или над логикой (разбор файлов, генерация PDF), которой пользуется бизнес. `cmd/main` поднимает сервисы до App и закрывает после.

## API

```go
type Service interface {
    Init(ctx context.Context) error // открыть соединения
    Stop() error                    // освободить то, что открыл Init
}

func New(services map[string]Service) *Catalog
func (c *Catalog) Get(name string) Service                         // паника, если сервиса нет
func (c *Catalog) Init(ctx context.Context, names ...string) error // в этом порядке
func (c *Catalog) Stop(names ...string) error                      // без имён — в обратном порядке Init
```

## Пакет services проекта

Каталог живёт в `pkg/services/services.go`. Там же — конфиги сервисов, порядок запуска и геттеры:

```go
// pkg/services/services.go
package services

const (
    LoggerName     = "logger"
    RepositoryName = "repository"
)

type Configs struct {
    Logger     logger.Configs
    Repository repository.Configs
}

var catalog *rockservices.Catalog

// Init создаёт сервисы из их конфигов и запускает в нужном порядке.
func Init(ctx context.Context, cfg Configs) error {
    catalog = rockservices.New(map[string]rockservices.Service{
        LoggerName:     logger.NewService(cfg.Logger),
        RepositoryName: repository.NewService(cfg.Repository),
    })
    return catalog.Init(ctx, LoggerName, RepositoryName) // логгер первым
}

// Stop останавливает сервисы в обратном порядке.
func Stop() error { return catalog.Stop() }

// Геттеры — по одному на сервис.
func Repository() repository.Service { return catalog.Get(RepositoryName).(repository.Service) }
```

```go
// cmd/main
if err := services.Init(ctx, cfg.Services); err != nil {
    return err // ни один App не стартует
}
defer services.Stop()

return apps.Run(ctx, cfg.Apps) // rockengine
```

```go
// где угодно — обычно в обработчиках
err := services.Repository().Insert(ctx, task)
```

## Дочерние сервисы

- **Дочерний сервис, который родитель открывает сам** (репозиторий — свой пул соединений): создаётся и открывается внутри родителя, в каталоге его нет.
- **Дочерний сервис, который открывается снаружи** — например, один пул на несколько сервисов: создайте его заранее, положите в каталог и передайте в конструктор родителя.

```go
pg := postgres.NewService(cfg.Postgres)
catalog = rockservices.New(map[string]rockservices.Service{
    PostgresName:   pg,
    RepositoryName: repository.NewService(cfg.Repository, pg),
})
return catalog.Init(ctx, PostgresName, RepositoryName) // сначала зависимость
```

## Тестирование

Для тестов пакет `services` проекта может дать функцию, которая подменяет каталог:

```go
// pkg/services/services.go
func UseCatalog(c *rockservices.Catalog) (restore func()) {
    prev := catalog
    catalog = c
    return func() { catalog = prev }
}

// тест обработчика
t.Cleanup(services.UseCatalog(rockservices.New(map[string]rockservices.Service{
    services.RepositoryName: &fakeRepository{},
})))
```

Каталог один на пакет, поэтому тесты, которые его подменяют, нельзя запускать с `t.Parallel()`.

## Правила

- `New` паникует на пустом имени или nil-сервисе.
- `Get`, `Init` и `Stop` паникуют на неизвестном имени — до того, как что-то запустилось. Ошибка в названии видна при старте.
- `Init` запускает сервисы в переданном порядке. Если один упал, уже запущенные этим вызовом останавливаются в обратном порядке, а ошибка называет сервис. Повторный `Init` того же сервиса — ошибка.
- `Stop()` без имён останавливает все запущенные сервисы в обратном порядке `Init`; `Stop(names...)` — указанные, в указанном порядке. Не запущенные пропускаются, поэтому повторный `Stop` безопасен. Время ограничено (30s, `WithStopTimeout`), ошибка называет зависший сервис.
- Паники в `Init`/`Stop` возвращаются как ошибки.
