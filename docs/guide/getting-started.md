# Быстрый старт

Соберём HTTP-сервис задач на PostgreSQL: создать задачу и получить её. Готовый результат — пример [examples/taskapi](https://github.com/arzab/gorock/tree/main/examples/taskapi): код на этой странице и в руководстве взят из него.

Нужны Go 1.26+ и Docker (для PostgreSQL).

---

## 1. Заготовка проекта

**С Claude Code** — плагин создаёт рабочий проект по шаблону:

```sh
claude plugin marketplace add arzab/gorock
claude plugin install gorock@gorock
# затем попросите: «создай GOROCK-проект github.com/acme/tasks»
```

**Вручную:**

```sh
mkdir tasks && cd tasks
go mod init github.com/acme/tasks

go get github.com/arzab/gorock-kit/rockengine@v0.6.0 \
       github.com/arzab/gorock-kit/rockconfig@v0.6.0 \
       github.com/arzab/gorock-kit/rockservices@v0.6.0 \
       github.com/arzab/gorock-kit/rockhandlers@v0.6.0 \
       github.com/arzab/gorock-catalog/rockhttp@v0.6.0 \
       github.com/arzab/gorock-catalog/rockbun@v0.6.0 \
       github.com/arzab/gorock-catalog/rocklog@v0.6.0 \
       github.com/google/uuid@latest
```

---

## 2. Что будет в проекте

```
tasks/
├── pkg/services/
│   ├── services.go                  каталог: названия, Configs, Init, Stop, геттеры
│   ├── logger/service.go            логгер
│   └── repository/                  задачи в PostgreSQL
├── internal/
│   ├── handlers/
│   │   ├── tasks/create.go          создать задачу (+ тест)
│   │   ├── tasks/get.go             получить задачу (+ тест)
│   │   └── middleware/log.go        логирование ошибок
│   └── models/task.go
├── cmd/
│   ├── apps/apps.go, http.go        HTTP App и привязка цепочек
│   └── main/main.go
├── configs/configs.yaml
└── migrations/001_tasks.sql
```

Пишите снизу вверх, по страницам руководства:

1. **Сервисы** — [Как писать Service](/guide/services): `repository` и `logger`, каталог в `services.go`.
2. **Модель** `internal/models/task.go` — структура с тегами `bun` (таблица `tasks`: `id`, `title`, `done`, `created_at`).
3. **Обработчики** — [Как писать Handler](/guide/handlers): `tasks.Create`, `tasks.Get`, middleware `Log`.
4. **App** — [Как писать App](/guide/apps): `http.go` и `apps.go`.
5. **Main** — [Main](/guide/main).
6. **Конфиг** — [Как писать Config](/guide/configs).

```sql
-- migrations/001_tasks.sql
CREATE TABLE IF NOT EXISTS tasks (
    id         TEXT PRIMARY KEY,
    title      TEXT        NOT NULL,
    done       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## 3. Запуск

```sh
go test ./...   # обработчики — без PostgreSQL и HTTP

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

## Что проверить

- **Без `DB_PASSWORD`** проект не стартует и называет переменную.
- **С выключенной базой** не стартует ни один App, а ошибка называет сервис: `init services: service "repository": init: …`.
- **По `Ctrl+C`** HTTP-сервер дообрабатывает текущие запросы, затем закрывается соединение с базой.
