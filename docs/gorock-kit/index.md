# gorock-kit

Ядро GOROCK — часть работы, одинаковая в каждом проекте: каталог сервисов, механизм обработчиков, движок App и строгая загрузка настроек. Вы пишете то, что отличает ваш проект; ядро — то, что повторяется везде.

[GitHub](https://github.com/arzab/gorock-kit) · [CHANGELOG](https://github.com/arzab/gorock-kit/blob/main/CHANGELOG.md) · текущая версия **v0.4.0** · требуется **Go 1.26+**

---

## Модули

| Слой | Модуль | Что даёт |
|---|---|---|
| **Services** | [rockservices](./rockservices) | каталог сервисов: регистрация под именем, получение по имени откуда угодно, открытие до App и закрытие после |
| **Handlers** | [rockhandlers](./rockhandlers) | обработчики: цепочки шагов, параметры и результат, ошибки с видом, общие шаги, middleware — без привязки к технологии |
| **Main** | [rockengine](./rockengine) | движок App: запуск по порядку, перезапуск упавших, остановка по одному в обратном порядке |
| **Config** | [rockconfig](./rockconfig) | один файл → типизированные `Configs`, `${ENV}`, значения по умолчанию, проверка пропусков и опечаток |

Модули не импортируют друг друга, ставите только нужные. `rockconfig` зависит от `gopkg.in/yaml.v3`, остальные — только от стандартной библиотеки.

Готовые App и сервисы под конкретные технологии — HTTP, PostgreSQL, Redis, логгер — в [каталоге](/catalog/).

```sh
go get github.com/arzab/gorock-kit/rockservices@latest
go get github.com/arzab/gorock-kit/rockhandlers@latest
go get github.com/arzab/gorock-kit/rockengine@latest
go get github.com/arzab/gorock-kit/rockconfig@latest
```

Все модули выпускаются одной версией; теги — по модулю (`rockservices/v0.4.0`).

---

## Контракты

Модули не зависят друг от друга, но совместимы по сигнатурам — это контракты слоёв.

**Service** — то, что хранит каталог:

```go
type Service interface {
    Init(ctx context.Context) error // открыть соединения
    Stop() error                    // закрыть
}
```

**Обработчик** — цепочка шагов, которую вызывает App:

```go
var Create = rockhandlers.Chain("orders.create", steps.Bearer, create)
```

**App** — то, чем управляет движок:

```go
type App interface {
    Init(ctx context.Context) error // подготовиться
    Exec(ctx context.Context) error // работать до отмены ctx
    Stop() []error                  // разблокировать Exec и дождаться текущей работы
}
```

Как всё это собирается в проект — [Архитектура](/architecture/) и [Быстрый старт](/guide/getting-started).
