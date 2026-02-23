# Configs

---

## Концепт

Configs — это документация которая всегда актуальна. Открыл один файл — увидел все слои, все зависимости, все технологии. Структура configs отражает структуру приложения. Неполный конфиг — приложение не запускается.

---

## Идея

Каждый слой проекта объявляет свою структуру `Configs` в файле `configs.go`. Корневой `Configs` в `cmd/main` агрегирует их все. Конфигурационный файл повторяет эту иерархию — один к одному.

Результат: один файл покрывает конфиги всего приложения. Глядя на него — сразу понятно что принадлежит какому слою.

---

## Как устроен Configs

Каждый пакет объявляет свою структуру `Configs` в `configs.go`. Сгруппируем по слоям.

### Engine — `cmd/`

Каждый App объявляет свой `Configs`. `cmd/configs.go` агрегирует их:

```go
// cmd/apps/http_server/configs.go
type Configs struct {
    Port    string `json:"port"`
    Timeout int    `json:"timeout" config:"default:30"`
}

// cmd/apps/pubsub_consumer/configs.go
type Configs struct {
    SubscriptionId string `json:"subscription_id"`
    ProjectId      string `json:"project_id"`
}

// cmd/configs.go
type Configs struct {
    HttpServer     http_server.Configs     `json:"http_server"`
    PubsubConsumer pubsub_consumer.Configs `json:"pubsub_consumer"`
}
```

### Realm — `internal/`

Как правило не имеет конфигов. Delivery, models и utils не нуждаются в настройке при старте — их поведение определяется бизнес-логикой, а не конфигурационным файлом.

### Toolkit — `pkg/`

Services и infra объявляют конфиги. Libs конфигов не имеют — они не singletons:

```go
// pkg/infra/logs/configs.go
type Configs struct {
    Level string `json:"level" config:"default:info"`
}

// pkg/infra/vars/configs.go
type Configs struct {
    Env string `json:"env" config:"default:dev"`
}

// pkg/infra/configs.go
type Configs struct {
    Logs logs.Configs `json:"logs"`
    Vars vars.Configs `json:"vars"`
}

// pkg/services/repository/configs.go
type Configs struct {
    Host     string `json:"host"`
    Port     string `json:"port"`
    Name     string `json:"name"`
    User     string `json:"user"`
    Password string `json:"password"`
}

// pkg/services/configs.go
type Configs struct {
    Repository repository.Configs  `json:"repository"`
    RunManager run_manager.Configs `json:"run_manager"`
}

// pkg/configs.go
type Configs struct {
    Infra    infra.Configs    `json:"infra"`
    Services services.Configs `json:"services"`
}
```

### infra/vars — глобальные переменные

Бывают значения которые не принадлежат ни одному конкретному сервису — имя окружения, feature flags, shared-параметры. Класть их в `Configs` какого-то одного сервиса неудобно, а читать напрямую из env в произвольных местах — плохая практика.

Для этого есть `pkg/infra/vars` — singleton, который хранит такие переменные и делает их доступными из любого слоя:

```go
// pkg/infra/vars/configs.go
type Configs struct {
    Env string `json:"env" config:"default:dev"`
}

// pkg/infra/vars/vars.go
var (
    Env          string
    FeatureFlags map[string]bool
)

func Init(configs Configs) {
    Env = configs.Env
}
```

Использование из любого слоя:

```go
if vars.Env == "prod" {
    // prod-specific behaviour
}
```

В конфигурационном файле vars лежит внутри секции `infra`:

```json
{
  "infra": {
    "logs": { "level": "info" },
    "vars": { "env": "$APP_ENV" }
  }
}
```

Разница с обычными `Configs`: vars — это не настройка конкретного компонента, а глобальное состояние доступное всем слоям без передачи аргументов.

---

### Корневой Configs

`cmd/main` собирает конфиги всех слоёв:

```go
// cmd/main/main.go
type Configs struct {
    Infra    infra.Configs    `json:"infra"`
    Services services.Configs `json:"services"`
    Apps     apps.Configs     `json:"apps"`
}
```

Конфигурационный файл повторяет эту вложенность:

```json
{
  "infra": {
    "logs": { "level": "info" }
  },
  "services": {
    "repository": {
      "host": "$DB_HOST",
      "port": "5432",
      "name": "$DB_NAME",
      "user": "$DB_USER",
      "password": "$DB_PASSWORD"
    },
    "run_manager": {
      "script_path": "./scripts/run.py"
    }
  },
  "apps": {
    "http_server": {
      "port": "$APP_PORT"
    }
  }
}
```

---

## gorock-kit: InitFromFile

Для загрузки конфига используется готовая функция из `gorock-kit`:

```go
mainConfigs, err := configs.InitFromFile[Configs](configsPath)
```

Функция принимает путь к файлу и шаблонную структуру. Поддерживает JSON и YAML.

Под капотом происходит последовательно:

1. Читает файл
2. Подставляет env-переменные — `$VAR` и `${VAR}` заменяются значениями до парсинга
3. Анмаршаллит в структуру `T`
4. Валидирует все поля — по умолчанию ни одно поле не должно быть пустым

Если валидация не прошла — возвращает ошибку с перечислением незаполненных полей. Приложение не запустится с неполным конфигом.

---

## Env-переменные

Подставляются до парсинга — работает для любых значений в файле:

```json
{
  "services": {
    "repository": {
      "host": "$DB_HOST",
      "password": "$DB_PASSWORD"
    }
  }
}
```

Если переменная не задана — подставляется пустая строка, валидация это поймает.

Секреты — только через env-переменные. Остальное можно вшить прямо в файл.

---

## Теги

Поведение валидации можно изменить через теги на полях структуры:

| Тег | Поведение |
|-----|-----------|
| _(без тега)_ | Поле обязательно, не может быть пустым |
| `config:"ignore"` | Поле полностью пропускается при валидации |
| `config:"omitempty"` | Поле не проверяется если nil или zero value |
| `config:"default:{value}"` | Подставляет значение по умолчанию если поле пустое |

```go
type ServerConfigs struct {
    Port      string       `json:"port"    config:"default:8080"`
    Timeout   int          `json:"timeout" config:"default:30"`
    App       fiber.Config `json:"app"     config:"ignore"`
    DebugMode bool         `json:"debug"   config:"omitempty"`
}
```

---

## Несколько окружений

Конфигурационный файл один, но окружений может быть несколько. Каждое окружение — отдельный файл:

```
configs/
├── configs.json   — дефолт если путь не указан
├── dev.json
├── staging.json
└── prod.json
```

Приложение не знает в каком окружении запускается — оно просто читает файл по переданному пути. Это позволяет запускать один и тот же бинарь в любом окружении без пересборки.

### Через позиционный аргумент

Простейший вариант — путь к файлу первым аргументом:

```go
// cmd/main/main.go
configsPath := "./configs/configs.json"
if len(os.Args) >= 2 {
    configsPath = os.Args[1]
}

mainConfigs, err := configs.InitFromFile[Configs](configsPath)
```

```bash
./app                           # → ./configs/configs.json
./app ./configs/dev.json        # → dev.json
./app ./configs/prod.json       # → prod.json
```

### Через именованный параметр

Более явный вариант — флаг с именем:

```go
// cmd/main/main.go
configsPath := flag.String("config", "./configs/configs.json", "path to config file")
flag.Parse()

mainConfigs, err := configs.InitFromFile[Configs](*configsPath)
```

```bash
./app --config ./configs/dev.json
./app --config ./configs/prod.json
```

Именованный параметр удобнее когда приложение принимает несколько аргументов — не нужно помнить порядок.

### Содержимое файлов

В `dev.json` значения вшиты прямо в файл. В `prod.json` — секреты через env, остальное вшито:

```json
// dev.json
{
  "services": {
    "repository": {
      "host": "localhost",
      "port": "5432",
      "name": "mydb_dev",
      "user": "postgres",
      "password": "postgres"
    }
  }
}
```

```json
// prod.json
{
  "services": {
    "repository": {
      "host": "$DB_HOST",
      "port": "5432",
      "name": "$DB_NAME",
      "user": "$DB_USER",
      "password": "$DB_PASSWORD"
    }
  }
}
```

---

## Правила

- **Каждый пакет объявляет свой `Configs`** — не передавать конфиг родительского слоя вниз.
- **Секреты — только через env-переменные.** Не хардкодить пароли и токены в файл.
- **Конфиг неизменяем после загрузки.** Это настройка при старте, не состояние.
- **Дефолты через тег `config:"default:{value}"`** — для сложной логики дефолтов используй `Init()`.
