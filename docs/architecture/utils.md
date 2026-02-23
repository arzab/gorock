# Utils

---

## Что такое utils

Вспомогательные stateless функции общего назначения — конвертации, форматирование, утилитарные операции. Не хранят состояние, не реализуют бизнес-логику.

Располагается в `internal/utils/` — используется только внутри проекта.

---

## Отличие от infra

`pkg/infra` — stateful глобальные объекты с инициализацией (логгер, трейсер).

`internal/utils` — stateless функции, которые просто принимают аргументы и возвращают результат. Никакого глобального состояния.

---

## Пример

```go
// internal/utils/convert.go
package utils

import "encoding/json"

func ToMap(v interface{}) (map[string]interface{}, error) {
    data, err := json.Marshal(v)
    if err != nil {
        return nil, err
    }
    var m map[string]interface{}
    if err := json.Unmarshal(data, &m); err != nil {
        return nil, err
    }
    return m, nil
}
```

---

## Правила

- **Только stateless функции.** Если нужно состояние — это не utils.
- **Общего назначения.** Если функция знает о конкретных сущностях проекта — она принадлежит delivery или models, не utils.
- **`internal/utils`**, не `pkg/utils` — утилиты специфичны для проекта и не предназначены для переноса.
