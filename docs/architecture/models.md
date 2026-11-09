# Models

**Realm** · `internal/models/`

Сущности предметной области. Общий язык на котором говорит слой delivery.

```
internal/models/
├── task.go              — plain struct сущности
├── user.go
└── order.go
```

или с разделением на properties/entities:

```
internal/models/
├── properties/
│   └── user.go          — plain данные без поведения
└── entities/
    └── user.go          — embed properties + поведение (ORM, validator, методы)
```

---

## Что такое models

Models — сущности предметной области. Они могут содержать бизнес-логику — но только ту которая принадлежит самой сущности, а не оркестрацию вызовов между слоями.

Например, `User.Create()` может хэшировать пароль перед сохранением — это поведение самой сущности, оно живёт здесь. А решение вызвать `User.Create()` и потом отправить письмо — это уже оркестрация, она живёт в delivery.

Располагается в `internal/` — только для использования внутри проекта. Принадлежат бизнесу, не инфраструктуре.

---

## Реализация контрактов Services

Модели могут реализовывать интерфейсы (контракты) из `pkg/services`. Поведение сущности закладывается прямо в модель — сервис использует интерфейс и не знает о конкретном типе.

Пример: сервис принимает `Saveable` с методом `SaveValidation`. Логика хэширования пароля живёт в `models.User` — не в сервисе:

```go
// pkg/services/repository/interface.go
type Saveable interface {
    GetId() string
    SaveValidation() error
}

// internal/models/user.go
type User struct {
    ID           string
    Name         string
    Email        string
    PasswordHash string
    password     string
}

func (u *User) GetId() string { return u.ID }

func (u *User) SaveValidation() error {
    hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    u.PasswordHash = string(hash)
    return nil
}
```

`User` реализует `Saveable` неявно — Go проверяет это при передаче. Сервис вызывает `SaveValidation()` не зная о `User`.

---

## Кто использует models

В первую очередь — `internal/delivery`. Именно там сосредоточена бизнес-логика которая работает с сущностями.

Остальные слои в идеале не зависят от models напрямую:

- `pkg/services` описывают собственные интерфейсы внутри пакета — это обеспечивает переносимость. Модели реализуют эти интерфейсы неявно.
- `cmd/apps` работает с delivery-сущностями, не с models напрямую.

---

## Два варианта организации

### Плоские structs

Подходит для большинства проектов. Все сущности лежат рядом:

```go
// internal/models/task.go
package models

type Task struct {
    ID           string
    RunId        string
    BucketName   string
    ObjectName   string
    TrainingType string
}
```

### Properties + Entities

Оправдано когда сущностям нужно поведение: реализация интерфейсов сервисов, библиотек (ORM, validator), методы, вычисляемые поля.

```go
// internal/models/properties/user.go
type UserProperties struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    Email        string `json:"email"`
    PasswordHash string `json:"-"`
}

// internal/models/entities/user.go
type User struct {
    bun.BaseModel `bun:"table:users"`
    properties.UserProperties

    Password string `json:"password,omitempty" bun:"-"` // приходит от клиента, не хранится
}

// BeforeCreate — поведение самой сущности: хэширует пароль и забывает его.
func (u *User) BeforeCreate() error {
    hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    u.PasswordHash, u.Password = string(hash), ""
    return nil
}
```

Поведение сущности обусловлено ей самой — `User` знает, как подготовить себя к сохранению. Delivery лишь решает, когда вызвать `BeforeCreate()` и затем `services.Repository().Insert(ctx, user)`.

Теги `bun` в entities — это реализация «интерфейса» ORM: репозиторий на [rockbun](/catalog/rockbun) принимает `any` и сам читает из модели имя таблицы и колонки. Сервис по-прежнему не импортирует `internal/models`.

Разделение позволяет реализовывать интерфейсы внешних библиотек в entities, не загрязняя plain-данные в properties. При смене библиотеки меняются только entities.

---

## Правила

- **Models — сущности предметной области**, а не вспомогательные типы.
- **Логика сущности — в models, оркестрация — в delivery.**
- **Структура свободная.** Плоские structs или properties/entities — выбор разработчика исходя из сложности проекта.
- **Основной потребитель — delivery.** Другие слои избегают прямой зависимости на models где это возможно.
