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

## Кто использует models

В первую очередь — `internal/delivery`. Именно там сосредоточена бизнес-логика которая работает с сущностями.

Остальные слои в идеале не зависят от models напрямую:

- `pkg/services` описывают собственные типы внутри пакета — это обеспечивает переносимость и исключает import cycles. Если переносимость не нужна, сервис может использовать models напрямую — на усмотрение разработчика.
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
    ID           string
    Name         string
    Email        string
    PasswordHash string
}

// internal/models/entities/user.go
type User struct {
    properties.UserProperties
}

// реализует интерфейс pkg/services/crud
func (u *User) Create() error {
    hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    u.PasswordHash = string(hash)
    return nil
}

func (u *User) TableName() string { return "users" }
func (u *User) Validate() error   { ... }
```

Поведение сущности обусловлено ей самой — `User` знает как создать себя корректно. Delivery лишь решает когда вызвать `Create()`.

Разделение позволяет реализовывать интерфейсы внешних библиотек в entities, не загрязняя plain-данные в properties. При смене библиотеки меняются только entities.

---

## Правила

- **Models — сущности предметной области**, а не вспомогательные типы.
- **Логика сущности — в models, оркестрация — в delivery.**
- **Структура свободная.** Плоские structs или properties/entities — выбор разработчика исходя из сложности проекта.
- **Основной потребитель — delivery.** Другие слои избегают прямой зависимости на models где это возможно.
