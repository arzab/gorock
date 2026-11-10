# rockhttp

<!-- Сгенерировано docs/scripts/gen-kit-pages.py из rockhttp/README_RU.md — правьте README в репозитории модуля. -->

::: tip Место в GOROCK · App
HTTP App на `net/http`: в `cmd/apps/http.go` вы привязываете цепочки к методам и путям (`Endpoint{Method, Path, Handler}`), он заполняет параметры и переводит итог в HTTP. Экспериментальный. → [Как писать App](/guide/apps)
:::

```sh
go get github.com/arzab/gorock-catalog/rockhttp@latest
```

[Исходный код и README на GitHub](https://github.com/arzab/gorock-catalog/tree/main/rockhttp)

> **Экспериментальный модуль:** API может меняться.

HTTP-транспорт для [rockhandlers](/gorock-kit/rockhandlers) на `net/http` (шаблоны Go 1.22+), без зависимостей.

```go
server := rockhttp.New(cfg.HTTP,
    rockhttp.Endpoint{Method: "POST", Path: "/tasks", Handler: tasks.Create, Status: http.StatusCreated},
    rockhttp.Endpoint{Method: "GET", Path: "/tasks/{id}", Handler: tasks.Get},
    rockhttp.Endpoint{Method: "GET", Path: "/files/{id}", Handler: files.Download}, // тип ответа *rockhttp.File
).Use(Log) // middleware rockhandlers вокруг всех эндпоинтов
engine.MustRegister("http", server)
```

`Handler` — цепочка `rockhandlers.Chain(обработчики...)`: обработчики ничего не знают про HTTP. Пустая цепочка (заготовка) отвечает `501 Not Implemented`.

## Params

Поля заполняются по тегам:
- `path:"shop"` — параметр пути;
- `query:"dry_run"` — параметр запроса; для слайса параметр повторяют;
- `header:"…"` — заголовок;
- `form:"title"` — поле формы, urlencoded или multipart;
- `form:"file"` с `*multipart.FileHeader` или `[]*multipart.FileHeader` — загрузка файлов;
- JSON-тело — по `json`-тегам.

Теги во встроенных структурах тоже биндятся. Например, `auth.BearerAuth` с `header:"Authorization"`.

Порядок: JSON, form, query, header, path — следующий источник перезаписывает предыдущий. Неверный ввод → 400 с сообщением без имён Go-типов: `invalid JSON body`, `invalid query`, `invalid path`, `invalid header`, `invalid form` (поле — в `fields`, например `{"qty": "must be a number"}`) или `request body is larger than N bytes`.

## Ответ

| Тип ответа | HTTP-ответ |
|---|---|
| структура с полями | JSON; статус из `Endpoint.Status`, по умолчанию 200 |
| пустая структура | 204 (или `Endpoint.Status`) |
| `*rockhttp.File` | поток файла; Range и `If-Modified-Since`, если `Content` — `io.ReadSeeker`; `Download: true` → attachment |
| `*rockhttp.Redirect` | редирект, по умолчанию 302 |
| ошибка | `ErrorBody{code, status, kind, message, fields}`; сообщения Internal скрыты; цепочка без обработчиков → 501 |

Вид ошибки → HTTP-статус:

| Kind | Статус |
|---|---|
| Invalid | 400 |
| Unauthorized | 401 |
| Forbidden | 403 |
| NotFound | 404 |
| Conflict | 409 |
| Unavailable | 503 |
| Timeout | 504 |
| Canceled | 499, без тела |
| Internal | 500 |

Это соответствие отдаёт функция `StatusOf`.

То, что не выражается типизированным ответом, — через контекст:
- `rockhttp.SetCookie(c, cookie)`;
- `rockhttp.SetHeader(c, k, v)`;
- `rockhttp.Request(c)` — сырой запрос, запасной выход.

Вне HTTP-вызова они ничего не делают или возвращают nil, поэтому шаги остаются тестируемыми.

## Конфиг

```yaml
port: "8080"            # обязателен
prefix: /api/v1
shutdown_timeout: 30s
read_timeout: 15s
write_timeout: 0s       # 0 = без лимита (долгие скачивания)
max_body_bytes: 10485760
internal_message: internal error
```

В коде:
- `OnError(r, route, err)` — вызывается, только если запись ответа сломалась после успешной цепочки. Ошибки запросов, включая ошибки разбора, логируйте в middleware rockhandlers через `Use`: биндинг выполняется внутри цепочки.
- `Middleware` — уровень net/http (CORS, gzip…).

После `Init` метод `App.Handler()` отдаёт роутер для `httptest`.
