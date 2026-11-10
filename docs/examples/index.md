# Примеры

Рабочие проекты, которые можно запустить и разобрать.

---

## taskapi — HTTP-сервис задач на PostgreSQL

[examples/taskapi](https://github.com/arzab/gorock/tree/main/examples/taskapi) — проект на gorock-kit `v0.5.0` и gorock-catalog `v0.5.0`: создать задачу и получить её по HTTP. Устроен так же, как проект из [Быстрого старта](/guide/getting-started), и проверен на PostgreSQL.

| Что показывает | Где |
|---|---|
| Ключи сервисов, регистрация, логгер — первым сервисом | `pkg/services/services.go`, `pkg/services/logger/` |
| Репозиторий поверх rockbun, не знающий о моделях | `pkg/services/repository/` |
| Обработчики: параметры, проверка, результат, ошибки-виды | `internal/handlers/tasks/` |
| Тесты обработчиков без PostgreSQL и HTTP — подделка в каталоге, `rockhandlers.Run` | `internal/handlers/tasks/*_test.go` |
| Общая обвязка: логирование отклонённых и упавших запросов | `internal/handlers/middleware/log.go` |
| HTTP App rockhttp и привязка обработчиков к методам и путям | `cmd/apps/` |
| Сборка: конфиг → сервисы → App → остановка в обратном порядке | `cmd/main/main.go` |
| Единый конфиг с `${ENV:-default}` | `configs/configs.yaml` |

```sh
git clone https://github.com/arzab/gorock && cd gorock/examples/taskapi

go test ./...   # без PostgreSQL и HTTP

docker run -d --name tasks-db -p 5432:5432 \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=tasks \
  -v "$PWD/migrations:/docker-entrypoint-initdb.d:ro" postgres:16-alpine

DB_PASSWORD=postgres go run ./cmd/main
```

```sh
curl -X POST localhost:8080/api/v1/tasks -H 'Content-Type: application/json' -d '{"title":"write docs"}'
# 201 {"task":{"id":"…","title":"write docs","done":false,"created_at":"…"}}

curl localhost:8080/api/v1/tasks/<id>
# 200 {"id":"…","title":"write docs","done":false,"created_at":"…"}

curl -X POST localhost:8080/api/v1/tasks -H 'Content-Type: application/json' -d '{"title":""}'
# 400 {"code":400,"status":"Bad Request","kind":"invalid","message":"invalid task","fields":{"title":"required"}}

curl localhost:8080/api/v1/tasks/nope
# 404 {"code":404,"status":"Not Found","kind":"not_found","message":"task not found"}
```

Если порт 5432 занят, запустите базу на другом, например `-p 55432:5432`, и передайте `DB_PORT=55432`.

События и задачи по расписанию (rockbus, rockcron) вернутся в пример, когда эти модули каталога перейдут на новое ядро.
