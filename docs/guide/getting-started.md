# Быстрый старт

Соберём проект заказов из [Архитектуры](/architecture/): создать заказ и получить его по HTTP, заказы хранятся в PostgreSQL.

Нужны Go 1.26+ и Docker (для PostgreSQL).

---

## 1. Заготовка проекта

**С Claude Code** — плагин создаёт рабочий проект по шаблону:

```sh
claude plugin marketplace add arzab/gorock
claude plugin install gorock@gorock
# затем попросите: «создай GOROCK-проект github.com/acme/orders»
```

**Вручную:**

```sh
mkdir orders && cd orders
go mod init github.com/acme/orders

go get github.com/arzab/gorock-kit/rockengine@latest \
       github.com/arzab/gorock-kit/rockconfig@latest \
       github.com/arzab/gorock-kit/rockservices@latest \
       github.com/arzab/gorock-kit/rockhandlers@latest \
       github.com/arzab/gorock-catalog/rockhttp@latest \
       github.com/arzab/gorock-catalog/rockbun@latest \
       github.com/arzab/gorock-catalog/rocklog@latest \
       github.com/google/uuid@latest
```

---

## 2. Что будет в проекте

```
orders/
├── pkg/services/
│   ├── services.go                  имена, Configs, Register, геттер Orders()
│   ├── logger/service.go            логгер
│   └── orders/                      заказы в PostgreSQL
├── internal/
│   ├── handlers/
│   │   ├── orders/create.go         создать заказ
│   │   ├── orders/get.go            получить заказ
│   │   ├── steps/auth.go            проверка токена
│   │   └── middleware/log.go        логирование ошибок
│   └── models/order.go
├── cmd/
│   ├── apps/apps.go, http.go        HTTP App и привязка обработчиков
│   └── main/main.go
├── configs/configs.yaml
└── migrations/001_orders.sql
```

Пишите снизу вверх, по страницам руководства — код на них ровно из этого проекта:

1. **Сервисы** — [Как писать Service](/guide/services): `orders` и `logger`, регистрация в `services.go`.
2. **Модель** `internal/models/order.go` — структура с тегами `bun` (таблица `orders`: `id`, `owner`, `item`, `qty`, `created_at`).
3. **Обработчики** — [Как писать Handler](/guide/handlers): `Create`, `Get`, шаг `Bearer`, middleware `Log`.
4. **App** — [Как писать App](/guide/apps): `http.go` и `apps.go`.
5. **Main** — [Main](/guide/main).
6. **Конфиг** — [Как писать Config](/guide/configs).

Миграция:

```sql
-- migrations/001_orders.sql
CREATE TABLE orders (
    id         text PRIMARY KEY,
    owner      text NOT NULL,
    item       text NOT NULL,
    qty        int  NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
```

---

## 3. Запуск

```sh
docker run -d --name orders-db -p 5432:5432 \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=orders \
  -v "$PWD/migrations:/docker-entrypoint-initdb.d:ro" postgres:16-alpine

DB_PASSWORD=postgres go run ./cmd/main
```

```sh
curl -X POST localhost:8080/api/v1/orders \
  -H 'Authorization: Bearer ann' -H 'Content-Type: application/json' \
  -d '{"item":"tea","qty":2}'
# 201 {"order":{"id":"…","owner":"ann","item":"tea","qty":2,"created_at":"…"}}

curl -X POST localhost:8080/api/v1/orders -H 'Content-Type: application/json' -d '{"item":"tea","qty":2}'
# 401 {"code":401,"status":"Unauthorized","kind":"unauthorized","message":"invalid token"}

curl -X POST localhost:8080/api/v1/orders -H 'Authorization: Bearer ann' \
  -H 'Content-Type: application/json' -d '{"item":"","qty":0}'
# 400 {"code":400,…,"kind":"invalid","message":"invalid order","fields":{"item":"required","qty":"must be > 0"}}

curl localhost:8080/api/v1/orders/nope
# 404 {"code":404,…,"kind":"not_found","message":"order not found"}
```

## Что проверить

- **Без `DB_PASSWORD`** проект не стартует и называет переменную: `services.orders.postgres.password: env var DB_PASSWORD is not set`.
- **С выключенной базой** не стартует ни один App, а ошибка называет сервис: `init services: service "orders": init: …`.
- **По `Ctrl+C`** HTTP-сервер дообрабатывает текущие запросы, затем закрывается соединение с базой.
- **Тест обработчика** проходит без базы и сервера: `go test ./...` — см. [тестирование обработчиков](/guide/handlers#тестирование).
