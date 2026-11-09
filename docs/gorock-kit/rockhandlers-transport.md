# Как написать транспорт для rockhandlers

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockhandlers/TRANSPORT_RU.md — правьте файл в репозитории модуля. -->

[Исходный файл на GitHub](https://github.com/arzab/gorock-kit/blob/main/rockhandlers/TRANSPORT_RU.md)

Транспорт — это App, который принимает запросы своего протокола (HTTP, очередь, gRPC, CLI, Telegram…) и отдаёт их маршрутам rockhandlers. Эталонная реализация — [rockhttp](/catalog/rockhttp); пример consumer'а очереди — ниже.

## Кто за что отвечает

| rockhandlers (уже сделано) | Транспорт (пишете вы) |
|---|---|
| цепочка: middleware → разбор → `Validate` → шаги | принять запрос/сообщение своего протокола |
| перехват паник везде, включая ваш разбор | заполнить `Params` из запроса (функция `fill`) |
| виды ошибок, `Canceled` / `Timeout` из контекста | превратить `Result` в ответ протокола |
| Locals, middleware, `Timeout` | превратить **вид** ошибки в ответ протокола |
| новый `Ctx` на каждый вызов, `Result` не утекает при ошибке | жизненный цикл App, остановка, параллельность, лимиты |

Не повторяйте в транспорте то, что уже есть в цепочке: свой `recover` вокруг `Serve`, валидацию, логирование ошибок маршрутов. Логирование — это middleware rockhandlers.

## Минимальный цикл

```go
result, err := route.Serve(ctx, func(params any) error {
    return decode(req, params) // заполнить новый P из запроса
})
if err != nil {
    writeError(resp, err)      // по rockhandlers.KindOf(err)
    return
}
writeResult(resp, result)      // result — это R маршрута (указатель на структуру)
```

Всё остальное в этом руководстве — подробности этих трёх строк.

## Шаг 1. Описание маршрутов и проверки на старте

У каждого транспорта своя «точка монтирования»: у HTTP — метод и путь, у очереди — имя очереди, у CLI — команда. Опишите её структурой, в которой маршрут — обычное поле:

```go
type Consumer struct {
    Queue   string             // транспортные метаданные
    Handler rockhandlers.Route // цепочка
}
```

В `Init` App-а:

- **Проверьте описания** (пустые поля, дубликаты) и верните ошибку — движок не запустит App.
- **Добавьте middleware уровня App:** `route := c.Handler.With(a.mws...)`. Первый — самый внешний; каждый `With` оборачивает всё добавленное раньше. Делайте это один раз при регистрации, а не на каждый запрос.
- **Проверьте типы**, если транспорт умеет не всё: `route.ResultType()` и `route.ParamsType()` возвращают `reflect.Type` указателей. Например, CLI, который печатает только текст, может отказать маршруту с `*rockhttp.File` — лучше упасть на старте, чем на запросе.
- **Подготовьте разбор заранее**, если он дорогой: по `ParamsType()` можно один раз построить карту полей и тегов.

`Route` неизменяем, `Serve` можно вызывать параллельно из разных горутин.

## Шаг 2. Разбор запроса (`fill`)

`fill` получает новый `P` — указатель на нулевую структуру — и заполняет его.

**Где он выполняется.** Внутри цепочки: после middleware, перед `Validate`. Поэтому:
- ошибки разбора видят middleware (логирование, метрики) — отдельный канал отчётов не нужен;
- `Timeout` покрывает и разбор;
- паника в `fill` становится `Internal`, а не роняет процесс;
- если контекст уже отменён или middleware не вызвал `Next`, `fill` не выполнится.

**Ошибки.**
- Простая `error` из `fill` становится `Invalid`, **и её текст увидит клиент.** Возвращайте понятный текст без внутренних подробностей: `query "page": "x" is not an integer`.
- Чтобы задать вид и сообщение самому, верните `*rockhandlers.Error`: например, `rockhandlers.Unauthorized("bad signature")` при проверке подписи вебхука.
- Ошибку ввода-вывода тела, если ничего нельзя сказать клиенту, оберните в `*rockhandlers.Error` с видом `KindInternal` — текст будет скрыт.

**Теги.** Используйте те же имена, что у rockhttp, где смысл совпадает, — тогда одна модель `Params` работает в нескольких транспортах:

| Тег | HTTP | Очередь | CLI |
|---|---|---|---|
| `json:"x"` | JSON-тело | JSON-тело сообщения | JSON из stdin / флага |
| `header:"X"` | заголовок | заголовок сообщения (AMQP headers, Kafka headers) | переменная окружения |
| `query:"x"`, `path:"x"`, `form:"x"` | URL, путь, форма | — | флаги / аргументы |

Обязательно поддержите **встроенные структуры**: переиспользуемые шаги (авторизация, тенант) получают данные через встроенную модель вроде `BearerAuth` с тегом `header:"Authorization"`.

**Лимиты.** Ограничивайте размер тела в самом разборе (`http.MaxBytesReader` и аналоги) — шаги уже получают заполненную структуру.

## Шаг 3. Результат

При успехе `result` — это `R` маршрута, указатель на заполненную структуру.

- **По умолчанию** сериализуйте его в формат протокола (JSON, protobuf…).
- **Особые типы результата** — ваш способ выразить то, что умеет только ваш протокол: `*rockhttp.File` (поток файла), `*rockhttp.Redirect`. Объявите их в пакете транспорта и разбирайте через type switch перед сериализацией. Шаг, который их возвращает, честно привязан к транспорту — это нормально.
- **Пустая структура** (`struct{}`) — договорённость «нечего отправлять»: HTTP отвечает 204, очередь просто подтверждает.
- **Статус успеха и прочие метаданные ответа**, которые относятся к точке монтирования, держите в её описании (`Endpoint.Status`), а не в маршруте.
- Если запись ответа сломалась после того, как его начало ушло клиенту, исправить уже ничего нельзя — сообщите через хук транспорта (`OnError` в rockhttp).

## Шаг 4. Ошибки

Переводите не саму ошибку, а её вид — `rockhandlers.KindOf(err)`:

| Kind | HTTP | Очередь | CLI |
|---|---|---|---|
| `KindNone` (успех) | 2xx | ack | 0 |
| `Invalid` | 400 | reject → DLQ | 2 |
| `Unauthorized` / `Forbidden` | 401 / 403 | reject → DLQ | 1 |
| `NotFound` / `Conflict` | 404 / 409 | reject → DLQ | 1 |
| `Unavailable` | 503 | requeue | 1 |
| `Timeout` | 504 | requeue или DLQ | 1 |
| `Canceled` | 499, без тела | requeue | 130 |
| `Internal` | 500 | requeue с лимитом попыток | 1 |

Правила:

- **Текст для клиента — только `rockhandlers.MessageOf(err)`.** Для `Internal` она возвращает пустую строку: покажите общий текст («internal error»). Никогда не отдавайте клиенту `err.Error()` — там причины для логов.
- **`rockhandlers.FieldsOf(err)`** — ошибки по полям; отдайте их, если протокол это позволяет (HTTP — в JSON, gRPC — в details).
- **`Canceled` — не ошибка сервера**: клиент ушёл или приложение останавливается. Не пишите тело ответа, а в очереди верните сообщение (requeue), чтобы оно не потерялось.
- **Неизвестные виды** (будущие версии) обрабатывайте как `Internal`.
- Логировать ошибки маршрутов — работа middleware, не транспорта.

## Шаг 5. Запасные выходы через контекст

Если шагу нужно повлиять на ответ так, как не выражается результатом (cookie, заголовок ответа, приоритет сообщения), или прочитать сырой запрос, дайте функции, работающие через контекст:

```go
type exchangeKey struct{}

type exchange struct {
    req     *http.Request
    cookies []*http.Cookie
}

// Транспорт: перед Serve.
ctx = context.WithValue(ctx, exchangeKey{}, &exchange{req: r})

// Публичное API транспорта: шаг вызывает rockhttp.SetCookie(c, cookie).
func SetCookie(ctx context.Context, c *http.Cookie) {
    if x, ok := ctx.Value(exchangeKey{}).(*exchange); ok {
        x.cookies = append(x.cookies, c)
    }
}
```

`Ctx` шага — это `context.Context`, поэтому такие функции принимают его напрямую. Вне вызова транспорта они ничего не делают (или возвращают nil), поэтому шаги остаются тестируемыми через `rockhandlers.Run`. Применяйте накопленное при записи ответа — и успешного, и ошибки.

## Шаг 6. Контекст, параллельность, остановка

- **Контекст запроса** — из протокола (`r.Context()` у HTTP, отменяется при обрыве соединения) или из контекста `Exec` App-а для очереди. Передайте его в `Serve`: отмена дойдёт до шагов и до БД.
- **Отмена кооперативная.** rockhandlers не бросает шаги в горутинах. Он прерывает цепочку между шагами, а внутри шага всё прерывается само, если шаг передаёт контекст в блокирующие вызовы.
- **Параллельность — забота транспорта:** пул воркеров, prefetch, лимит одновременных запросов. Один `Ctx` не потокобезопасен, но каждый вызов `Serve` получает свой.
- **Остановка.** При отмене контекста `Exec` перестаньте принимать новое и дождитесь сообщений в работе. Что делать с ними, если не успели, решает протокол: HTTP-сервер ждёт в пределах таймаута, очередь возвращает неподтверждённое. Контракт App — в [rockengine](/gorock-kit/rockengine).

## Шаг 7. Тестирование транспорта

Брокер для этого не нужен: соберите маршруты-заглушки и проверьте перевод.

```go
func TestKindMapping(t *testing.T) {
    type P struct{ ID string `json:"id"` }
    type R struct{}
    fail := func(err error) rockhandlers.Route {
        return rockhandlers.Chain("t", func(*rockhandlers.Ctx[*P, *R]) error { return err })
    }
    cases := []struct {
        route rockhandlers.Route
        want  outcome
    }{
        {fail(nil), ack},
        {fail(rockhandlers.Invalid("bad")), reject},
        {fail(rockhandlers.Unavailable("down")), requeue},
        {fail(errors.New("boom")), requeue},
        {rockhandlers.Chain("p", func(*rockhandlers.Ctx[*P, *R]) error { panic("x") }), requeue},
    }
    for _, c := range cases {
        if got := handle(c.route, message{Body: `{"id":"1"}`}); got != c.want {
            t.Fatalf("got %v, want %v", got, c.want)
        }
    }
}
```

Проверьте отдельно:
- ошибку разбора (`Invalid` с понятным текстом);
- скрытие текста `Internal`;
- `Canceled` на заранее отменённом контексте;
- встроенные структуры в `Params`;
- особые типы результата;
- middleware App-а видят и ошибки разбора.

## Пример: consumer очереди

```go
func (a *App) handle(ctx context.Context, msg *queue.Message) {
    route := a.routes[msg.Queue] // подготовлены в Init: Handler.With(a.mws...)
    _, err := route.Serve(ctx, func(params any) error {
        if err := json.Unmarshal(msg.Body, params); err != nil {
            return fmt.Errorf("invalid JSON body: %v", err) // → Invalid
        }
        return bindHeaders(msg.Headers, params) // теги header, включая встроенные структуры
    })
    switch rockhandlers.KindOf(err) {
    case rockhandlers.KindNone:
        msg.Ack()
    case rockhandlers.KindUnavailable, rockhandlers.KindTimeout, rockhandlers.KindCanceled, rockhandlers.KindInternal:
        msg.Requeue()
    default:
        msg.Reject() // плохое сообщение — повтор не поможет
    }
}
```

`Init`, `Exec` и `Stop` этого App-а — на странице [Apps](/architecture/apps) документации GOROCK.

## Чек-лист

- [ ] Описание точки монтирования со своими метаданными и полем `Handler rockhandlers.Route`.
- [ ] `Init`: проверка описаний, `With(mws...)` один раз, проверка `ResultType()`, если нужна.
- [ ] Разбор только внутри `fill`; понятные тексты ошибок; `*Error` для своих видов; встроенные структуры; лимиты размера.
- [ ] Результат: формат по умолчанию, особые типы, пустая структура.
- [ ] Ошибки по `KindOf`; текст клиенту — только `MessageOf`, общий текст для `Internal`; `FieldsOf`; `Canceled` — не ошибка сервера.
- [ ] Запасные выходы через контекст, без эффекта вне транспорта.
- [ ] Контекст запроса передаётся в `Serve`; параллельность и остановка — на стороне App.
- [ ] Тесты перевода видов, ошибок разбора, паник, отмены.
