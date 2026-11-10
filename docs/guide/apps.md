# Как писать App

App — технология, через которую к проекту обращаются: HTTP-сервер, consumer очереди, планировщик, бот. Он заполняет параметры обработчика, вызывает его и переводит итог на язык своей технологии. Зачем так — на странице [Архитектура](/architecture/#_3-app).

Обычно App берётся готовым из [каталога](/catalog/), а вы только говорите ему, какой обработчик где висит. Свой App пишется, когда готового нет.

```
cmd/apps/
├── apps.go          какие App запущены + их Configs
├── http.go          какой обработчик на каком методе и пути
└── <свой_app>/      свой App, если готового нет
```

---

## Готовый App: HTTP

### Привязка обработчиков — `http.go`

```go
// cmd/apps/http.go
func httpEndpoints() []rockhttp.Endpoint {
    return []rockhttp.Endpoint{
        {Method: http.MethodPost, Path: "/orders", Handler: orders.Create, Status: http.StatusCreated},
        {Method: http.MethodGet, Path: "/orders/{id}", Handler: orders.Get},
    }
}
```

Метод, путь и статус успеха — понятия HTTP, поэтому они здесь, а не в обработчике. Один и тот же обработчик можно привязать к нескольким App.

`Status` необязателен: по умолчанию успех — `200`, а пустой результат — `204`. Укажите `201`, если следуете REST-соглашению «создание → 201». Общий префикс (`/api/v1`) задаётся в конфиге App.

### Регистрация — `apps.go`

```go
// cmd/apps/apps.go
type Configs struct {
    HTTP rockhttp.Config
}

func Register(e *rockengine.Engine, cfg Configs) {
    http := rockhttp.New(cfg.HTTP, httpEndpoints()...).Use(middleware.Log)
    e.MustRegister("http", http, rockengine.RestartPolicy{MaxRetries: 3, Delay: time.Second, ResetAfter: time.Minute})
}
```

- **`Use(...)`** вешает общую обвязку обработчиков на все маршруты App.
- **`RestartPolicy`** — что делать, если App упал: сколько раз перезапускать и с какой паузой.
- **Порядок регистрации важен:** App останавливаются в обратном порядке. Регистрируйте HTTP последним — он остановится первым, пока остальные, от которых он зависит, ещё работают.

Что делает [rockhttp](/catalog/rockhttp):
- **на входе** заполняет параметры по тегам `json`, `header`, `path`, `query`, `form`, включая встроенные структуры, и ограничивает размер тела;
- **на выходе** отдаёт результат как JSON, а ошибки переводит в статусы: 400, 401, 403, 404, 409, 503, 504, 500 со скрытым текстом;
- **при остановке** дообрабатывает начатые запросы.

Для файлов, редиректов и cookie есть особые типы результата и функции — смотрите страницу модуля.

---

## Несколько App с одними обработчиками

Тот же `orders.Create` можно повесить на очередь. Это второй App со своей привязкой, а обработчик не меняется:

```go
func Register(e *rockengine.Engine, cfg Configs) {
    e.MustRegister("orders-queue", queue.New(cfg.Queue,
        queue.Consumer{Queue: "orders.create", Handler: orders.Create}))
    e.MustRegister("http", rockhttp.New(cfg.HTTP, httpEndpoints()...).Use(middleware.Log))
}
```

`queue` здесь — App вашей очереди: готовый из каталога или свой.

---

## Свой App

Свой App нужен, когда готового для вашей технологии нет. Цепочку, проверку параметров, перехват паник и middleware уже делает слой обработчиков. App пишет только свою технологию:

1. **принять обращение;**
2. **заполнить параметры** — внутри вызова `route.Serve(ctx, fill)`;
3. **перевести итог** по `rockhandlers.KindOf(err)` на язык технологии;
4. **аккуратно остановиться.**

Контракт App — три метода, их вызывает движок:

```go
type App interface {
    Init(ctx context.Context) error // подготовиться: соединение, регистрация обработчиков
    Exec(ctx context.Context) error // работать; отмена ctx — «заканчивай текущее и выходи»
    Stop() []error                  // разблокировать Exec и дождаться текущей работы
}
```

Ядро обработки одного обращения:

```go
_, err := route.Serve(ctx, func(params any) error {
    return json.Unmarshal(msg.Body, params) // простая ошибка → «неверные данные»
})
switch rockhandlers.KindOf(err) {
case rockhandlers.KindNone:
    msg.Ack()
case rockhandlers.KindUnavailable, rockhandlers.KindInternal, rockhandlers.KindCanceled, rockhandlers.KindTimeout:
    msg.Requeue()
default:
    msg.Reject() // плохое сообщение — повтор не поможет
}
```

Подробное руководство с чек-листом — [Как написать свой App](/gorock-kit/rockhandlers-transport).

---

## Фоновая работа

Работа без обращений извне — отправка накопленных событий, чистка старых данных — тоже App: у неё есть `Exec`, который работает до отмены контекста, и `Stop`. Обработчики ей не обязательны.

---

## Чего не делать

- **Не класть бизнес-логику в App.** Он только переводит запрос и итог.
- **Не задавать пути и очереди внутри обработчика.** Привязка — в `cmd/apps/`.
- **Не повторять в своём App валидацию, recover и логирование ошибок обработчиков.** Это уже есть в слое обработчиков.
- **Не делать `Stop`, который ждёт `Exec`.** `Stop` вызывается и тогда, когда `Exec` не запускался.

→ Дальше: [Main](/guide/main) · API: [rockengine](/gorock-kit/rockengine), [rockhttp](/catalog/rockhttp)
