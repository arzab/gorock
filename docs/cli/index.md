# CLI

::: warning В разработке
CLI `gorock` пока не выпущен. Всё, что он будет делать, можно сделать вручную — см. [Быстрый старт](/architecture/getting-started).
:::

CLI будет генерировать шаблонный код по правилам архитектуры — слои, сервисы и endpoints одной командой, — чтобы вы тратили время на логику, а не на создание файлов.

Например, генерация сервиса создаст `pkg/services/<name>/` с файлами `configs.go`, `interface.go`, `service.go`, а генерация HTTP endpoint — `internal/delivery/endpoints/<name>/` с `endpoint.go`, `handlers.go`, `params.go`, `response.go`.

---

## Пока CLI нет

Шаблоны всех файлов есть в документации:

- сервис — [Services](/architecture/services#реализация-сервиса)
- endpoint — [Delivery](/architecture/delivery#http-endpoints)
- свой App — [Apps](/architecture/apps#как-написать-свои-app)
- весь проект целиком — [examples/taskapi](https://github.com/arzab/gorock/tree/main/examples/taskapi)
