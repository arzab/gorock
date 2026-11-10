# Как писать App

App — технология, через которую к проекту обращаются: HTTP-сервер, consumer очереди, планировщик, бот. Он заполняет параметры цепочки, запускает её и переводит итог на язык своей технологии. Зачем так — на странице [Архитектура](/architecture/#_3-app).

Обычно App берётся готовым из [каталога](/catalog/), а вы только говорите ему, какая цепочка где висит. Свой App пишется, когда готового нет.

```
cmd/apps/
├── apps.go          какие App запущены + их Configs + Run
├── http.go          какая цепочка на каком методе и пути
└── <свой_app>/      свой App, если готового нет
```

---

## Готовый App: HTTP

### Какая цепочка где висит — `http.go`

```go
// cmd/apps/http.go
func httpEndpoints() []rockhttp.Endpoint {
    return []rockhttp.Endpoint{
        {Method: http.MethodPost, Path: "/tasks", Handler: tasks.Create, Status: http.StatusCreated},
        {Method: http.MethodGet, Path: "/tasks/{id}", Handler: tasks.Get},
    }
}
```

Метод, путь и статус успеха — понятия HTTP, поэтому они здесь, а не в обработчике. Одну цепочку можно привязать к нескольким App.

`Status` необязателен: по умолчанию успех — `200`, пустой ответ — `204`. Укажите `201`, если следуете REST-соглашению «создание → 201». Общий префикс (`/api/v1`) задаётся в конфиге App.

### Запуск — `apps.go`

```go
// cmd/apps/apps.go
type Configs struct {
    HTTP rockhttp.Config
}

func Run(ctx context.Context, cfg Configs) error {
    engine := rockengine.New(map[string]rockengine.App{
        "http": rockhttp.New(cfg.HTTP, httpEndpoints()...).Use(middleware.Log),
    })
    return engine.Run(ctx, "http") // до SIGINT/SIGTERM или падения App
}
```

- **`Use(...)`** вешает middleware на все цепочки App.
- **Порядок в `Run` важен:** App останавливаются в обратном порядке. Ставьте HTTP последним — он остановится первым, пока остальные, от которых он зависит, ещё работают.
- **Если App упал,** движок останавливает всё, и процесс перезапускает оркестратор. Без оркестратора — `rockengine.New(apps).WithRestarts(3, time.Second)`.

### Как App находит цепочку

В `Init` HTTP-App строит из `httpEndpoints()` обычный `http.ServeMux`. На каждый запрос маршрутизатор находит свою цепочку и вызывает её:

```
POST /tasks → ServeMux → tasks.Create.Run(ctx, разбор запроса)
            → middleware → Validate → обработчики → Normalize
            → ответ → JSON 201, или ошибка → 400 / 404 / 500
```

Что делает [rockhttp](/catalog/rockhttp):
- **на входе** заполняет параметры по тегам `json`, `header`, `path`, `query`, `form` и ограничивает размер тела;
- **на выходе** отдаёт ответ как JSON, ошибки переводит в статусы, пустая цепочка — `501`;
- **при остановке** дообрабатывает начатые запросы.

Для файлов, редиректов и cookie есть особые типы ответа и функции — смотрите страницу модуля.

---

## Несколько App с одними цепочками

Ту же `tasks.Create` можно повесить на очередь — это второй App со своей привязкой:

```go
engine := rockengine.New(map[string]rockengine.App{
    "queue": queue.New(cfg.Queue, queue.Consumer{Queue: "tasks.create", Handler: tasks.Create}),
    "http":  rockhttp.New(cfg.HTTP, httpEndpoints()...).Use(middleware.Log),
})
return engine.Run(ctx, "queue", "http")
```

`queue` здесь — App вашей очереди: готовый из каталога или свой.

---

## Свой App

Порядок цепочки, проверку параметров, перехват паник и middleware уже делает слой обработчиков. Свой App пишет только технологию:

1. **принять обращение;**
2. **найти цепочку** по своей точке входа (очередь, команда, расписание);
3. **запустить её** с функцией, которая раскладывает обращение в параметры;
4. **перевести итог** по `rockhandlers.KindOf(err)` на язык технологии;
5. **аккуратно остановиться.**

```go
response, err := pipeline.Run(ctx, func(params rockhandlers.Params) error {
    return json.Unmarshal(msg.Body, params) // обычная ошибка → «неверные данные»
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

Контракт App — три метода, их вызывает движок:

```go
type App interface {
    Init(ctx context.Context) error // подготовиться: соединение, маршруты
    Exec(ctx context.Context) error // работать; отмена ctx — «заканчивай и выходи»
    Stop() []error                  // разблокировать Exec и дождаться текущей работы
}
```

Подробно, с чек-листом — [Как написать свой App](/gorock-kit/rockhandlers-transport).

---

## Фоновая работа

Работа без обращений извне — отправка накопленных событий, чистка старых данных — тоже App: `Exec` работает до отмены контекста, `Stop` прибирается. Цепочки обработчиков ему не обязательны.

---

## Чего не делать

- **Не класть бизнес-логику в App.** Он только переводит обращение и итог.
- **Не задавать пути и очереди внутри обработчиков.** Привязка — в `cmd/apps/`.
- **Не повторять в своём App проверку параметров, recover и логирование ошибок.** Это уже есть в слое обработчиков.
- **Не делать `Stop`, который ждёт `Exec`.** `Stop` вызывается и тогда, когда `Exec` не запускался.

→ Дальше: [Main](/guide/main) · API: [rockengine](/gorock-kit/rockengine), [rockhttp](/catalog/rockhttp)
