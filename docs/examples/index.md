# Примеры

Рабочие проекты, которые можно запустить и разобрать.

---

::: warning Примеры ещё на модели gorock-kit v0.2.0
`taskapi` написан на gorock-kit `v0.2.0`: сервисы — ресурсы движка, HTTP — на Fiber (`rockfiber`), endpoints — цепочки `fiber.Handler`. В текущем ядре этого больше нет: сервисы — [rockservices](/gorock-kit/rockservices), обработчики — [rockhandlers](/gorock-kit/rockhandlers), HTTP — [rockhttp](/catalog/rockhttp). Проект на текущей модели — в [Быстром старте](/guide/getting-started); пример будет переведён на неё.
:::

## taskapi — HTTP-сервис на PostgreSQL

[examples/taskapi](https://github.com/arzab/gorock/tree/main/examples/taskapi) — GOROCK-проект на gorock-kit v0.2.0.

| Что показывает | Где |
|---|---|
| Запуск через движок: ресурсы → apps, graceful shutdown | `cmd/main/main.go` |
| Готовые App: HTTP, шина событий, планировщик | `cmd/apps/apps.go` |
| Самоописывающиеся endpoints, цепочка handlers, params/response | `internal/delivery/endpoints/` |
| Бизнес-ошибки: 400, 404 и скрытые 500 | `internal/delivery/endpoints/get_task/` |
| События и trace_id, который доходит из HTTP-запроса в лог события | `internal/delivery/events/` |
| Задачи по расписанию | `internal/delivery/jobs/` |
| Репозиторий поверх rockbun, не знающий о моделях | `pkg/services/repository/` |
| Настройка логгера из конфига | `pkg/infra/logs/` |
| Единый конфиг с `${ENV:-default}` | `configs/configs.yaml` |

```sh
git clone https://github.com/arzab/gorock && cd gorock/examples/taskapi

docker run -d --name tasks-db -p 5432:5432 \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=tasks \
  -v "$PWD/migrations:/docker-entrypoint-initdb.d:ro" postgres:16-alpine

DB_PASSWORD=postgres go run ./cmd/main
```
