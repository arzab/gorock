# Как написать свой App

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockhandlers/TRANSPORT_RU.md — правьте файл в репозитории модуля. -->

[Исходный файл на GitHub](https://github.com/arzab/gorock-kit/blob/main/rockhandlers/TRANSPORT_RU.md)

Свой App пишется, когда готового для вашей технологии нет: очередь, gRPC, бот, CLI. App принимает обращения своего протокола и запускает цепочки обработчиков. Готовый пример — [rockhttp](/catalog/rockhttp).

## Кто за что отвечает

| rockhandlers (уже сделано) | App (пишете вы) |
|---|---|
| порядок: middleware → разбор → `Validate` → обработчики → `Normalize` | принять обращение |
| перехват паник везде | разложить обращение в параметры (функция `fill`) |
| виды ошибок, отмена и таймаут | превратить ответ и вид ошибки в ответ протокола |
| | запуск, остановка, параллельность, лимиты |

Не повторяйте в App то, что уже есть: свой `recover`, проверку параметров, логирование ошибок обработчиков (это middleware).

## Главный цикл

```go
response, err := pipeline.Run(ctx, func(params rockhandlers.Params) error {
    return decode(req, params) // params — новая пустая структура параметров
})
if err != nil {
    writeError(resp, err) // по rockhandlers.KindOf(err)
    return
}
writeResponse(resp, response)
```

## 1. Привязка цепочек

У каждого App своя «точка входа»: у HTTP — метод и путь, у очереди — имя очереди, у CLI — команда. Опишите её структурой, где цепочка — обычное поле:

```go
type Consumer struct {
    Queue   string
    Handler *rockhandlers.Pipeline
}
```

В `Init` App-а проверьте описания и добавьте свои middleware: `pipeline.Use(a.middlewares...)` — один раз, при регистрации.

## 2. Разбор обращения (`fill`)

`fill` получает новую пустую структуру параметров и заполняет её.

- Выполняется внутри цепочки, после middleware. Поэтому middleware видят ошибки разбора, а таймаут покрывает и разбор.
- Простая ошибка из `fill` превращается в «неверные данные» с сообщением `invalid request`. Подробность остаётся в причине, для логов.
- Чтобы сказать клиенту точнее, верните `rockhandlers.Invalid("invalid JSON body")` или ошибку с полями. Пишите без имён Go-типов и структур.
- Используйте те же теги, что `rockhttp`, где смысл совпадает: `json` — тело, `header` — заголовки (HTTP или сообщения), `path`, `query`, `form`. Тогда одни параметры работают в разных App.

## 3. Ответ

При успехе `response` — заполненная структура ответа. По умолчанию отправьте её в формате протокола. Особые типы ответа для своего протокола (как `rockhttp.File` и `rockhttp.Redirect`) объявите в пакете App и разберите через type switch.

## 4. Ошибки

Переводите вид ошибки, а не саму ошибку:

| Вид | HTTP | Очередь | CLI |
|---|---|---|---|
| `KindNone` (успех) | 2xx | ack | 0 |
| `Invalid`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict` | 400 / 401 / 403 / 404 / 409 | в очередь недоставленных | 1–2 |
| `Unavailable`, `Timeout` | 503 / 504 | повторить позже | 1 |
| `Canceled` | 499, без тела | вернуть в очередь | 130 |
| `Internal` | 500 | повторить с лимитом | 1 |

- Клиенту — только `rockhandlers.MessageOf(err)`; для `Internal` она пустая, покажите общий текст.
- `rockhandlers.FieldsOf(err)` — ошибки по полям, если протокол позволяет их передать.
- `errors.Is(err, rockhandlers.ErrHandlerNotFound)` — цепочка ещё не написана: HTTP отвечает `501`.
- `*rockhandlers.NormalizeError` — ответ не прошёл `Normalize`: это ошибка сервера.

## 5. Контекст и остановка

- Контекст обращения — из протокола (у HTTP — `r.Context()`) или из `Exec` App-а. Передайте его в `Run`.
- Параллельность, лимиты и prefetch — забота App.
- При остановке перестаньте принимать новое и дождитесь начатого.

## 6. Тесты

Брокер не нужен: соберите цепочки-заглушки и проверьте перевод видов ошибок.

```go
type P struct{}
func (*P) Validate() error { return nil }
type R struct{}
func (*R) Normalize() error { return nil }

fail := func(err error) *rockhandlers.Pipeline {
    return rockhandlers.Chain(func(*rockhandlers.Ctx, *P, *R) error { return err })
}
// fail(nil) → ack, fail(rockhandlers.Invalid("bad")) → reject, fail(errors.New("boom")) → requeue
```
