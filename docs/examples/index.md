# Примеры

Рабочие проекты, которые можно запустить и разобрать. Код в документации взят из них.

---

## taskapi — HTTP-сервис на PostgreSQL

[examples/taskapi](https://github.com/arzab/gorock/tree/main/examples/taskapi) — эталонный GOROCK-проект на gorock-kit. На нём построен [Быстрый старт](/architecture/getting-started).

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

---

## fullapp — все модули gorock-kit

[gorock-kit/examples/fullapp](https://github.com/arzab/gorock-kit/tree/main/examples/fullapp) — компактный сервис в одном файле, где подключены все модули: PostgreSQL и Redis как ресурсы, шина, планировщик, HTTP. Удобен, чтобы быстро увидеть API, но слои GOROCK в нём не разнесены по папкам.
