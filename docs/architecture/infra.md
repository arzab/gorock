# Infra

**Toolkit** · `pkg/infra/`

Инфраструктурные примитивы которые обслуживают сам проект. Инициализируются первыми — раньше services и apps.

```
pkg/infra/
├── main.go        — Configs + Init + Stop (только если нужно)
├── logs/          — глобальный логгер
├── tracer/        — distributed tracing
├── eventbus/      — внутренний publish/subscribe
└── vars/          — env-переменные и feature flags
```

---

## Что такое infra

Обёртки над системами которые обслуживают **сам проект** — логирование, трассировка, внутренний event bus. В отличие от `pkg/services`, которые использует бизнес-логика, `pkg/infra` нужна всем слоям независимо от того что делает приложение.

Infra не обязана следовать паттерну сервиса. Примитив может не иметь `Configs`, `Init`, `Stop` и интерфейса — достаточно набора глобальных функций. Форма определяется потребностью конкретного примитива.

`main.go` появляется только когда есть что инициализировать. Если все примитивы самодостаточны — он не нужен.

---

## logs

Глобальный логгер. Простейший случай — набор функций без состояния после `Init`.

```go
package logs

func Init(configs Configs) { ... }

func Infof(format string, args ...interface{})  { ... }
func Errorf(format string, args ...interface{}) { ... }
func Warnf(format string, args ...interface{})  { ... }
func Fatalf(format string, args ...interface{}) { ... }

func InfoWithFields(obj interface{}, message string)  { ... }
func ErrorWithFields(obj interface{}, message string) { ... }
```

Все слои импортируют `pkg/infra/logs`, а не конкретную библиотеку напрямую. Это позволяет поменять реализацию в одном месте.

---

## vars

Env-переменные и настройки которые не принадлежат ни одному конкретному сервису — имя окружения, feature flags, shared-значения.

```go
package vars

type Configs struct {
	Env          string `json:"env"`
	FeatureFlags map[string]bool `json:"feature_flags"`
}

var cfg Configs

func Init(configs Configs){
	cfg = configs
}

func Get() {return cfg}

```

Использование:

```go
if vars.Get().Env == "dev" { ... }
```

---

## tracer

Distributed tracing — трассировка запроса через все слои. Конкретная реализация (OpenTelemetry, Jaeger и др.) скрыта за функциями пакета.

---

## eventbus

Внутренний publish/subscribe для event-driven логики в монолите. Любой слой публикует событие без прямой зависимости от обработчика.

```go
package eventbus

func Publish(id string, payload any) { ... }
func Register(events []Event)        { ... }

type Event struct {
    ID      string
    Handler func(payload any) error
    Mode    EventMode
}

type EventMode int
const (
    Instant    EventMode = iota // горутина, fire and forget
    WorkerPool                  // канал + N воркеров, контроль нагрузки
    Queue                       // буферизованный канал, гарантия порядка
)
```

Обработчики описываются в `internal/delivery/events/` и регистрируются через `event_bus_app` — отдельный App в `cmd/apps`, который при `Init` вызывает `eventbus.Register`, а при `Exec` слушает и диспетчеризует события.

В микросервисах eventbus не нужен — event-driven выносится на уровень инфраструктуры (Pub/Sub, Kafka).

---

## Правила

- **Infra инициализируется первой.** Логгер нужен всем уже при старте остальных слоёв.
- **Доступ через функции пакета**, не через геттеры как в services: `logs.Infof(...)`, не `infra.Logs().Infof(...)`.
- **Форма свободная.** Не нужно натягивать паттерн сервиса если примитив в нём не нуждается.
- **Нет привязки к предметной области.** Если примитив знает о конкретных сущностях проекта — он не infra.
