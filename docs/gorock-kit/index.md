# gorock-kit

Ядро GOROCK — структурные части, которые нужны каждому сервису, чем бы он ни занимался: как он стартует и останавливается, как настраивается, где живут сервисы и как обрабатывается запрос.

Архитектура GOROCK говорит, **где** лежит код. gorock-kit даёт готовый код для мест, которые в каждом проекте одинаковые. Вам остаётся писать то, что уникально для проекта: бизнес-логику в Realm и обёртки над своими технологиями в Toolkit.

[GitHub](https://github.com/arzab/gorock-kit) · [CHANGELOG](https://github.com/arzab/gorock-kit/blob/main/CHANGELOG.md) · текущая версия **v0.3.0** · требуется **Go 1.26+**

---

## Модули и слои

| Слой GOROCK | Модуль | Что даёт |
|---|---|---|
| **Engine** · `cmd/main` | [rockengine](./rockengine) | Жизненный цикл apps: запуск в порядке регистрации, остановка по одному в обратном порядке, рестарты, паники, таймаут |
| **Configs** · `cmd/main` | [rockconfig](./rockconfig) | Один файл YAML/JSON → типизированный `Configs`, `${ENV}`, дефолты, проверка обязательных полей и опечаток |
| **Toolkit · Services** · `pkg/services` | [rockservices](./rockservices) | Каталог сервисов: создать, хранить по имени, выдать, поднять до apps, остановить после |
| **Realm · Delivery** · `internal/delivery` | [rockhandlers](./rockhandlers) | Обработка запроса без привязки к транспорту: цепочки шагов, `Next`, Locals, middleware, валидация, виды ошибок |

Модули не импортируют друг друга, ставите только нужные. `rockconfig` зависит от `gopkg.in/yaml.v3`, остальные — только от стандартной библиотеки.

Всё, что привязано к конкретной технологии — HTTP-сервер, PostgreSQL, Redis, логгер, шина событий, планировщик, Telegram-бот, — живёт в [каталоге](/catalog/).

---

## Установка

```sh
go get github.com/arzab/gorock-kit/rockengine@latest
go get github.com/arzab/gorock-kit/rockconfig@latest
go get github.com/arzab/gorock-kit/rockservices@latest
go get github.com/arzab/gorock-kit/rockhandlers@latest
```

Все модули выпускаются одной версией. Теги — по модулю (`rockengine/v0.3.0`), как того требует Go для репозитория из нескольких модулей.

---

## Контракты, на которых всё держится

Модули не зависят друг от друга, но совместимы по сигнатурам.

**App** — то, что работает: HTTP-сервер, consumer, планировщик, бот. Им управляет [rockengine](./rockengine).

```go
type App interface {
    Init(ctx context.Context) error // подготовка
    Exec(ctx context.Context) error // работа; при отмене ctx — завершить текущее и вернуться
    Stop() []error                  // разблокировать Exec, дождаться текущей работы
}
```

**Service** — то, чем пользуются: репозиторий, кэш, хранилище. Их хранит и поднимает [rockservices](./rockservices).

```go
type Service interface {
    Name() string                   // ключ в каталоге
    Init(ctx context.Context) error // открыть соединения
    Stop() error                    // закрыть
}
```

**Route** — то, что обрабатывает запрос: цепочка шагов [rockhandlers](./rockhandlers). App-транспорт (например, [rockhttp](/catalog/rockhttp)) заполняет параметры из своего протокола и переводит результат или ошибку обратно.

```go
var CreateTask = rockhandlers.Chain("tasks.create", auth.Bearer, create)
result, err := route.Serve(ctx, fill) // так его вызывает транспорт
```

---

## Как это выглядит в проекте

```
cmd/main:  rockconfig → rockservices.Init → rockengine.Run (apps) → rockservices.Stop
```

```go
// cmd/main/main.go
func Exec() error {
    cfg, err := rockconfig.InitFromFile[Configs]("./configs/configs.yaml")
    if err != nil {
        return fmt.Errorf("init configs: %w", err)
    }

    ctx := context.Background()
    services.Register(rockservices.Default(), cfg.Services) // pkg/services/services.go
    if err := rockservices.Init(ctx); err != nil {
        return fmt.Errorf("init services: %w", err) // ни один app не стартует
    }
    defer rockservices.Stop() // после остановки всех apps

    engine := rockengine.NewEngine()
    apps.Register(engine, cfg.Apps) // cmd/apps: rockhttp и другие
    return engine.Run()             // до SIGINT/SIGTERM или падения app
}
```

Разбор по шагам — в [Быстром старте](/architecture/getting-started).
