# Как писать Config

Config — переменные и константы проекта и ссылки на переменные окружения в одном файле, разложенные по частям проекта. Загружает и проверяет его [rockconfig](/gorock-kit/rockconfig). Зачем так — на странице [Архитектура](/architecture/#_5-config).

---

## Каждая часть объявляет свою секцию

У каждого сервиса и App есть структура `Configs`. Слой собирает их, Main собирает слои, а файл повторяет эту структуру один к одному.

```go
// pkg/services/orders/configs.go
type Configs struct {
    Postgres rockbun.Configs
}

// pkg/services/services.go
type Configs struct {
    Logger logger.Configs `config:",omitempty"`
    Orders orders.Configs
}

// cmd/apps/apps.go
type Configs struct {
    HTTP rockhttp.Config
}

// cmd/main/main.go
type Configs struct {
    Services services.Configs
    Apps     apps.Configs
}
```

```yaml
# configs/configs.yaml
services:
  logger:
    level: info
    format: json
  orders:
    postgres:
      host: ${DB_HOST:-localhost}
      user: ${DB_USER:-postgres}
      password: ${DB_PASSWORD}
      database: ${DB_NAME:-orders}

apps:
  http:
    port: ${PORT:-8080}
    prefix: /api/v1
```

Загрузка — одна строка в Main:

```go
cfg, err := rockconfig.InitFromFile[Configs]("./configs/configs.yaml")
```

Поддерживаются YAML и JSON.

---

## Правила полей

**Имена ключей** — `snake_case` от имени поля, теги не нужны: `Postgres` → `postgres`, `MaxBodyBytes` → `max_body_bytes`.

**Поле обязательно, пока не сказано иначе.** Обязательное — значит ключ должен быть в файле. `0` и `false` подходят, пустая строка — нет.

| Тег | Что значит |
|---|---|
| — | обязательное поле |
| `config:",omitempty"` | необязательное; без ключа — нулевое значение |
| `config:",default=30s"` | необязательное со значением по умолчанию |
| `config:",env=PORT"` | значение берётся из переменной окружения |
| `config:"-"` | не читается из файла, задаётся в коде (функции, сложные типы) |
| `config:"name"` | своё имя ключа |

Опции комбинируются: `config:"port,default=8080"`. `default=` идёт последним.

**Длительности — строкой с единицей:** `5s`, `1m30s`. Голое число — ошибка, иначе `timeout: 30` означало бы 30 наносекунд.

**Секция с `omitempty`, которой нет в файле,** не проверяется. Значения по умолчанию внутри неё всё равно применяются.

---

## Строгость

Все проблемы приходят одной ошибкой при старте:

```
rockconfig: ./configs/configs.yaml: missing required fields:
  apps.http.port
  services.orders.postgres.password (empty)
```

- **Неизвестный ключ — ошибка:** опечатка `requst_timeout` не превратится тихо в значение по умолчанию. Если файл читают несколько программ — `rockconfig.AllowUnknownKeys()`.
- **Неустановленная переменная окружения** без значения по умолчанию — тоже ошибка с её именем.

---

## Секреты и окружения

- **Секреты — только ссылками:** `${DB_PASSWORD}` — ошибка, если переменная не задана; `${DB_HOST:-localhost}` — со значением по умолчанию. `$` без скобок — обычный символ.
- **Ссылки раскрываются** только в значениях, не в ключах и не в комментариях.
- **Окружения — файлы одной структуры:** `configs/configs.yaml` для разработки, `configs/prod.yaml` для продакшена. Main принимает путь аргументом: `./app ./configs/prod.yaml`.

---

## Чего не делать

- **Не читать `os.Getenv` в коде.** Значение — поле `Configs` и ключ в файле.
- **Не хранить настройки в глобальных переменных.**
- **Не делать поле с значением по умолчанию в коде без `omitempty` или `default=`** — иначе его придётся указывать в каждом файле.
- **Не забывать файлы окружений:** новое поле добавляется во все `configs/*.yaml`.

→ Дальше: [Быстрый старт](/guide/getting-started) · API: [rockconfig](/gorock-kit/rockconfig)
