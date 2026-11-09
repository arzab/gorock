# Итог

Вы прошли все три слоя. Вот как выглядит готовый проект.

---

## Структура проекта

```
project/
├── cmd/
│   ├── main/
│   │   └── main.go             — Configs, загрузка конфига, регистрация в движке, Run
│   └── apps/
│       ├── apps.go             — Configs всех apps + Register
│       └── queue_consumer/     — свой App, если готового нет
│
├── internal/
│   ├── delivery/
│   │   ├── steps/              — переиспользуемые шаги (auth, тенант…)
│   │   ├── endpoints/          — маршруты rockhandlers + список HTTP endpoints
│   │   ├── events/             — обработчики событий (rockbus)
│   │   └── jobs/               — задачи по расписанию (rockcron)
│   ├── models/                 — сущности предметной области
│   └── utils/                  — вспомогательные функции
│
├── pkg/
│   ├── services/               — обёртки над внешними системами (каталог rockservices)
│   ├── infra/                  — логгер, трейсер, vars
│   └── libs/                   — ваш переиспользуемый код
│
├── configs/
│   ├── configs.yaml
│   ├── dev.yaml
│   └── prod.yaml
│
├── migrations/                 — допустимо
├── scripts/                    — допустимо
└── deployments/                — допустимо
```

Весь **код приложения** живёт внутри трёх слоёв. Остальные папки — на усмотрение разработчика.

---

## Как всё связано

При входящем запросе:

```
Запрос
  ↓
App (Engine)           принимает запрос, передаёт в Delivery
  ↓
Delivery (Realm)       выполняет бизнес-логику, вызывает Services
  ↓
Services (Toolkit)     работает с технологией — БД, кэш, API
  ↓
Ответ
  ↑
App                    получает результат, переводит на язык протокола
```

При запуске и остановке:

```
infra.Init → [движок] services Init → apps Init → apps Exec
                                                     ↓
                                               [сигнал ОС]
                                                     ↓
                     services Stop ← apps Stop (по одному, в обратном порядке)
```

---

## Каждый слой знает только своё

| Слой | Знает | Не знает |
|------|-------|----------|
| **Toolkit** | как работать с технологией | что делает бизнес |
| **Realm** | что делать при запросе | как работает транспорт |
| **Engine** | когда передать управление | что делает бизнес |

---

## Где что взять

| Слой | Ядро ([gorock-kit](/gorock-kit/)) | [Каталог](/catalog/) |
|---|---|---|
| Engine | [rockengine](/gorock-kit/rockengine) | [rockhttp](/catalog/rockhttp), [rockfiber](/catalog/rockfiber), [rockbus](/catalog/rockbus), [rockcron](/catalog/rockcron), [rocktelebot](/catalog/rocktelebot) |
| Configs | [rockconfig](/gorock-kit/rockconfig) | |
| Realm | [rockhandlers](/gorock-kit/rockhandlers) | |
| Toolkit | [rockservices](/gorock-kit/rockservices) | [rockbun](/catalog/rockbun), [rockredis](/catalog/rockredis), [rocklog](/catalog/rocklog) |

---

## Куда положить код

Когда не знаешь, куда положить код, — ответь на вопросы:

1. **Это обёртка над внешней технологией?** → `pkg/services`
2. **Это инструмент самого проекта** (логгер, трейсер)? → `pkg/infra`
3. **Это бизнес-логика — что делать при запросе или событии?** → `internal/delivery`
4. **Это структура данных проекта?** → `internal/models`
5. **Это запуск, транспорт или оркестрация?** → `cmd/`
