# Для ИИ-агентов

Чтобы Claude Code, Codex, Cursor и другие агенты писали код по правилам GOROCK и на актуальном API gorock-kit, а не придумывали своё, есть два инструмента:

- **Плагин для Claude Code** со skill `gorock:service` — создаёт проект, добавляет сервисы, обработчики, привязку к HTTP, App и поля конфига по рецептам и сам проверяет сборку.
- **`AGENTS.md`** — правила проекта для любого агента. Кладётся в корень вашего проекта.

---

## Claude Code

```sh
claude plugin marketplace add arzab/gorock
claude plugin install gorock@gorock
```

Или внутри сессии: `/plugin install gorock --marketplace arzab/gorock`.

Дальше просто просите, что нужно, — skill подключается сам, когда в проекте есть gorock-kit или вы просите создать сервис на нём:

```
Создай сервис на gorock-kit в ./notes, модуль github.com/acme/notes:
POST /notes сохраняет заметку в PostgreSQL, GET /notes/{id} отдаёт её (404, если нет).
```

Явный вызов — `/gorock:service`.

Что внутри skill:

| | |
|---|---|
| Порядок работы | разобраться в проекте → выбрать рецепт → подключить в одном месте → `gofmt`, `go build`, `go vet`, запуск |
| Рецепты | сервис (PostgreSQL, Redis, внешний API, логгер), обработчик и привязка к HTTP, общий шаг и группы, middleware, свой App, события, задачи по расписанию, поле конфига, разовая команда, тесты |
| Справочник API | сигнатуры ядра gorock-kit и модулей каталога, ключи конфига |
| Каркас проекта | `scripts/scaffold.sh` создаёт рабочий проект со всеми слоями и `AGENTS.md` |

---

## Другие агенты — AGENTS.md

`AGENTS.md` читают Codex, Cursor, Gemini CLI, GitHub Copilot и другие. Положите его в корень проекта:

```sh
curl -o AGENTS.md https://raw.githubusercontent.com/arzab/gorock/main/plugin/skills/service/templates/AGENTS.md
```

Для Claude Code рядом создайте `CLAUDE.md` с одной строкой — он подключит те же правила:

```
@AGENTS.md
```

Проект, созданный через skill, получает оба файла автоматически.

Что в `AGENTS.md`: команды сборки и запуска, слои и куда класть код, контракты `Service`, обработчиков и `App`, правила каждого слоя и конфига, и список устаревшего API, которого не должно быть в коде.

---

## Как это проверено

Каркас, который создаёт `scaffold.sh`, собирается, проходит `go vet` и отвечает на запросы: `GET /api/v1/hello?name=Ann` → `200`, без имени → `400` с записью в логе. Код рецептов и руководства взят из одного проекта заказов, который собирается и проходит тесты.

Плагин лежит в [репозитории gorock](https://github.com/arzab/gorock/tree/main/plugin). Версия плагина совпадает с версией gorock-kit, под которую написаны рецепты.
