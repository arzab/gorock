# gorock-kit

Готовые реализации для GOROCK-проектов: движок жизненного цикла, загрузка конфигов, логгер, HTTP-сервер, шина событий, планировщик, Telegram-бот, PostgreSQL и Redis.

Архитектура GOROCK говорит, **где** лежит код. gorock-kit даёт готовый код для мест, которые в каждом проекте одинаковые: как запустить и остановить приложение, как принять HTTP-запрос, как загрузить конфиг. Вам остаётся писать только то, что уникально для проекта, — бизнес-логику в Realm.

[GitHub](https://github.com/arzab/gorock-kit) · [CHANGELOG](https://github.com/arzab/gorock-kit/blob/main/CHANGELOG.md) · текущая версия **v0.1.0**

---

## Модули и слои

Каждый модуль — отдельный Go-модуль. Ставите только нужные, модули не импортируют друг друга.

| Слой GOROCK | Модуль | Что даёт |
|---|---|---|
| **Engine** · `cmd/main` | [rockengine](./rockengine) | Запуск и остановка: ресурсы → apps, остановка в обратном порядке, рестарты, паники, таймаут |
| **Configs** · `cmd/main` | [rockconfig](./rockconfig) | Один файл YAML/JSON → типизированный `Configs`, `${ENV}`, дефолты, проверка обязательных полей и опечаток |
| **Infra** · `pkg/infra` | [rocklog](./rocklog) | Структурированный логгер на `log/slog`, логгер в контексте запроса |
| **Engine · Apps** · `cmd/apps` | [rockfiber](./rockfiber) | HTTP App на Fiber: самоописывающиеся endpoints, парсинг и валидация параметров, стандартные ошибки, trace-id |
| | [rockbus](./rockbus) | Шина событий внутри процесса: очередь на топик, порядок внутри топика |
| | [rockcron](./rockcron) | Планировщик: cron-выражения и интервалы |
| | [rocktelebot](./rocktelebot) | Telegram-бот: команды, middleware, клавиатуры |
| **Toolkit · Services** · `pkg/services` | [rockbun](./rockbun) | PostgreSQL через bun: пул, транзакции |
| | [rockredis](./rockredis) | Redis: строки, хэши, списки, множества, pub/sub |

---

## Установка

```sh
go get github.com/arzab/gorock-kit/rockengine@latest
go get github.com/arzab/gorock-kit/rockconfig@latest
go get github.com/arzab/gorock-kit/rockfiber@latest
# ... только нужные модули
```

Все модули выпускаются одной версией. Теги — по модулю (`rockengine/v0.1.0`), как того требует Go для репозитория из нескольких модулей.

---

## Два контракта, на которых всё держится

Модули не зависят друг от друга, но совместимы по сигнатурам. Всё, что регистрируется в движке, — одно из двух.

**App** — то, что работает: HTTP-сервер, consumer, планировщик, бот.

```go
type App interface {
    Init(ctx context.Context) error // подготовка
    Exec(ctx context.Context) error // работа; при отмене ctx — завершить текущее и вернуться
    Stop() []error                  // разблокировать Exec, дождаться текущей работы
}
```

**Resource** — то, чем пользуются: пул БД, клиент кэша.

```go
type Resource interface {
    Init(ctx context.Context) error // открыть соединения
    Stop() error                    // закрыть
}
```

`rockfiber`, `rockbus`, `rockcron`, `rocktelebot` — это App. `rockbun.Service` и `rockredis.Service` — Resource. Ваши собственные сервисы и приложения пишутся по тем же контрактам — подробнее в [Services](/architecture/services) и [Apps](/architecture/apps).

---

## Как это выглядит в проекте

```go
// cmd/main/main.go
func Exec() error {
    cfg, err := rockconfig.InitFromFile[Configs]("./configs/configs.yaml")
    if err != nil {
        return fmt.Errorf("init configs: %w", err)
    }
    if err := infra.Init(cfg.Infra); err != nil { // настраивает rocklog
        return fmt.Errorf("init infra: %w", err)
    }

    engine := rockengine.NewEngine()
    services.Register(engine, cfg.Services) // rockbun, rockredis — ресурсы
    apps.Register(engine, cfg.Apps)         // rockfiber, rockbus, rockcron — apps

    return engine.Run()
}
```

Полный рабочий проект — [examples/taskapi](https://github.com/arzab/gorock/tree/main/examples/taskapi), разбор по шагам — в [Быстром старте](/architecture/getting-started).
