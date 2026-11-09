# Libs

**Toolkit** · `pkg/libs/`

Ваш переиспользуемый код без глобального состояния. Не singletons, не требуют `Init` — импортируешь и используешь.

```
pkg/libs/
├── pagination/       — разбор и формирование страниц
├── money/            — работа с денежными суммами
└── httpclient/       — общий HTTP-клиент для внешних API
```

---

## Что такое libs

Внутренние библиотеки — строительные блоки, на которых строятся слои проекта. В отличие от `pkg/services`, libs не являются singleton'ами и не хранят глобального состояния. Это просто библиотечный код: берёшь, используешь, передаёшь как аргумент.

---

## Libs и gorock-kit

Раньше в `pkg/libs/go-kit` лежал общий код для HTTP-сервера, загрузки конфигов, парсинга параметров и логгера. Теперь это отдельная библиотека [gorock-kit](/gorock-kit/), и её модули подключаются как обычные зависимости через `go get` — копировать их в `pkg/libs` не нужно.

| Было в `pkg/libs/go-kit` | Теперь |
|---|---|
| `configs.InitFromFile` | [rockconfig](/gorock-kit/rockconfig) |
| `http/server`, `http/endpoints` | [rockfiber](/gorock-kit/rockfiber): `rockfiber.New`, `rockfiber.GET/POST/...` |
| `http/params` | `rockfiber.DefaultHandler`, `rockfiber.ParseRequest` |
| `http/responses` | `rockfiber.NewError`, `rockfiber.ErrorResponse` |
| `http/utils` | `rockfiber.GetFromContext`, `rockfiber.HandlerInitInCtx` |
| `logs` (logrus) | [rocklog](/gorock-kit/rocklog) (slog) |

В `pkg/libs` остаётся только то, что специфично для вашей команды или ваших проектов и не является обёрткой над внешней системой.

---

## Отличие от services и infra

| | `pkg/libs` | `pkg/services` | `pkg/infra` |
|---|---|---|---|
| Состояние | Нет | Singleton | Глобальное |
| Init/Stop | Нет | Есть, управляет движок | `Init` при старте |
| Использование | Импортируешь и вызываешь | Через геттер | Через функции пакета |
| Назначение | Строительные блоки | Обёртки над внешними системами | Логирование, трейсинг, vars |

---

## Правила

- **Libs — не singletons.** Никакого глобального состояния, никакого `Init` на уровне слоя.
- **`pkg/libs`** — доступен всем слоям, включая services и infra.
- **Независимость от проекта.** Libs не знают о `internal/models` и бизнес-логике — это переносимый инструментарий.
- **Не дублируйте gorock-kit.** Если готовый модуль решает задачу — подключите его.
