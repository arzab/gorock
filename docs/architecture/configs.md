# Configs

---

## Концепт

Configs — это документация, которая всегда актуальна. Открыл один файл — увидел все слои, все зависимости, все технологии. Структура configs отражает структуру приложения. Неполный конфиг — приложение не запускается.

---

## Идея

Каждый слой проекта объявляет свою структуру `Configs`. Корневой `Configs` в `cmd/main` агрегирует их все. Файл конфига повторяет эту иерархию один к одному.

Результат: один файл покрывает конфиги всего приложения. Глядя на него, сразу понятно, что какому слою принадлежит.

Загружает файл [rockconfig](/gorock-kit/rockconfig) из gorock-kit. Поддерживаются YAML и JSON; примеры ниже — на YAML.

---

## Как устроен Configs

### Engine — `cmd/`

`cmd/apps` агрегирует конфиги приложений. Готовые App из gorock-kit приносят свой конфиг:

```go
// cmd/apps/apps.go
type Configs struct {
    HTTP rockfiber.Config                         // port, endpoints_path_prefix, request_timeout...
    Bus  rockbus.Config  `config:",omitempty"`     // queue_size
    Cron rockcron.Config `config:",omitempty"`     // location
}
```

### Realm — `internal/`

Как правило, конфигов не имеет. Delivery, models и utils не нуждаются в настройке при старте — их поведение определяется бизнес-логикой, а не файлом.

### Toolkit — `pkg/`

Services и infra объявляют конфиги. Libs конфигов не имеют — они не singletons.

```go
// pkg/infra/logs/logs.go
type Configs struct {
    Level  string `config:",default=info"`
    Format string `config:",default=json"`
}

// pkg/infra/infra.go
type Configs struct {
    Logs logs.Configs `config:",omitempty"`
}

// pkg/services/repository/configs.go
type Configs struct {
    Postgres rockbun.Configs // host, port, user, password, database, пул...
}

// pkg/services/services.go
type Configs struct {
    Repository repository.Configs
}
```

### Корневой Configs

```go
// cmd/main/main.go
type Configs struct {
    Infra    infra.Configs
    Services services.Configs
    Apps     apps.Configs
}
```

Файл повторяет эту вложенность:

```yaml
infra:
  logs:
    level: info
    format: json

services:
  repository:
    postgres:
      host: ${DB_HOST:-localhost}
      user: ${DB_USER:-postgres}
      password: ${DB_PASSWORD}
      database: ${DB_NAME:-tasks}

apps:
  http:
    port: ${PORT:-8080}
    endpoints_path_prefix: /api/v1
    use_trace_id: true
    request_timeout: 5s
```

Загрузка — одна строка:

```go
cfg, err := rockconfig.InitFromFile[Configs]("./configs/configs.yaml")
```

---

## Имена ключей

Теги `json`/`yaml` не нужны. Имя поля переводится в `snake_case`:

| Поле Go | Ключ в файле |
|---|---|
| `Services` | `services` |
| `RequestTimeout` | `request_timeout` |
| `HTTPServer` | `http_server` |
| `UserID` | `user_id` |

Своё имя — первым параметром тега: `config:"public_url"`.

---

## Обязательные поля и теги

Поле **обязательно**, если у него нет `omitempty`, `default=` или `-`. Обязательное поле должно **присутствовать** в файле — значение может быть любым, включая `0` и `false`. Обязательная строка не может быть пустой.

| Тег | Поведение |
|-----|-----------|
| _(без тега)_ | обязательное, ключ должен быть в файле |
| `config:",omitempty"` | необязательное |
| `config:",default=8080"` | значение по умолчанию, если ключа нет или он пустой |
| `config:",env=APP_SECRET"` | взять из переменной окружения, она важнее файла |
| `config:"-"` | поле не читается из файла — задаётся в коде (функции, сложные типы) |
| `config:"name"` | своё имя ключа |

Опции комбинируются: `config:"port,default=8080"`. `default=` должен быть последней опцией.

```go
type ServerConfigs struct {
    Port    string        `config:",default=8080"`
    Timeout time.Duration `config:",default=30s"`
    Debug   bool          `config:",omitempty"`
    App     fiber.Config  `config:"-"`
}
```

**Вложенные секции.** Если у секции `omitempty` и её нет в файле — её поля не проверяются. Если секция есть — её поля проверяются как обычно. Значения по умолчанию и `env=` внутри секции применяются, даже когда самой секции в файле нет.

**Неизвестные ключи — ошибка.** Опечатка `request_timout` не превратится молча в значение по умолчанию. Если один файл читают несколько программ — `rockconfig.InitFromFile[Configs](path, rockconfig.AllowUnknownKeys())`.

**Длительности — только строкой с единицей:** `5s`, `1m30s`. Голое число отклоняется — иначе `timeout: 30` означало бы 30 наносекунд.

Все проблемы приходят одной ошибкой:

```
rockconfig: ./configs/configs.yaml: missing required fields:
  apps.http.port
  services.repository.postgres.password (empty)
unknown keys (typo? use rockconfig.AllowUnknownKeys() to ignore):
  apps.http.request_timout
```

---

## Env-переменные

Строковые значения могут ссылаться на переменные окружения:

```yaml
password: ${DB_PASSWORD}             # ошибка, если переменная не задана
host: ${DB_HOST:-localhost}          # значение по умолчанию
port: ${PORT:-8080}                  # строка приводится к типу поля
dsn: "postgres://${DB_HOST}/app"     # можно встраивать
hash: "$2a$10$..."                   # одиночный $ остаётся как есть
```

Подстановка выполняется **после** разбора файла, поэтому значение переменной не может сломать структуру YAML. Незаданная переменная без `:-default` — ошибка с именем переменной и путём к полю:

```
services.repository.postgres.password: env var DB_PASSWORD is not set (use ${DB_PASSWORD:-default} for an optional value)
```

Секреты — только через env-переменные. Остальное можно вшить прямо в файл.

---

## Глобальные переменные — infra/vars

Бывают значения, которые не принадлежат ни одному сервису, — имя окружения, feature flags. Класть их в конфиг какого-то сервиса неудобно, а читать env в произвольных местах — плохая практика. Для них есть `pkg/infra/vars`:

```go
// pkg/infra/vars/vars.go
type Configs struct {
    Env string `config:",default=dev"`
}

var cfg Configs

func Init(c Configs) { cfg = c }
func Get() Configs   { return cfg }
```

```yaml
infra:
  vars:
    env: ${APP_ENV:-dev}
```

Подробнее → [Infra](/architecture/infra).

---

## Несколько окружений

Файл конфига один, окружений может быть несколько. Каждое окружение — отдельный файл:

```
configs/
├── configs.yaml   — по умолчанию, если путь не указан
├── dev.yaml
└── prod.yaml
```

Приложение не знает, в каком окружении запускается, — оно читает файл по переданному пути. Один и тот же бинарь работает в любом окружении без пересборки.

### Через позиционный аргумент

```go
path := "./configs/configs.yaml"
if len(os.Args) > 1 {
    path = os.Args[1]
}
cfg, err := rockconfig.InitFromFile[Configs](path)
```

```bash
./app                         # → ./configs/configs.yaml
./app ./configs/prod.yaml     # → prod.yaml
```

### Через флаг

```go
path := flag.String("config", "./configs/configs.yaml", "path to config file")
flag.Parse()
cfg, err := rockconfig.InitFromFile[Configs](*path)
```

```bash
./app --config ./configs/prod.yaml
```

Флаг удобнее, когда у приложения несколько аргументов, — не нужно помнить порядок.

### Содержимое файлов

В `dev.yaml` значения вшиты прямо в файл, в `prod.yaml` секреты приходят из env:

```yaml
# dev.yaml
services:
  repository:
    postgres:
      host: localhost
      user: postgres
      password: postgres
      database: tasks_dev
```

```yaml
# prod.yaml
services:
  repository:
    postgres:
      host: ${DB_HOST}
      user: ${DB_USER}
      password: ${DB_PASSWORD}
      database: tasks
      ssl_mode: verify-full
```

---

## Правила

- **Каждый пакет объявляет свой `Configs`** — не передавать конфиг родительского слоя вниз.
- **Секреты — только через env-переменные.** Не хардкодить пароли и токены в файл.
- **Конфиг неизменяем после загрузки.** Это настройка при старте, а не состояние.
- **Дефолты — через `config:",default=..."`**; если у поля есть дефолт в коде, пометьте его `omitempty`, иначе пользователю придётся писать его в файл.
- **Поля, которые нельзя задать в файле** (функции, сторонние структуры), — `config:"-"`, их заполняют в коде.
