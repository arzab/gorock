# Для ИИ-агентов

Чтобы Claude Code, Codex, Cursor и другие агенты писали код по правилам GOROCK и на актуальном API gorock-kit, а не придумывали своё, есть два инструмента:

- **Плагин для Claude Code** со skill `gorock:service` — создаёт проект, добавляет endpoints, сервисы, события, задачи и apps по рецептам и сам проверяет сборку.
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
POST /notes сохраняет заметку в PostgreSQL, GET /notes/:id отдаёт её (404, если нет).
```

Явный вызов — `/gorock:service`.

Что внутри skill:

| | |
|---|---|
| Порядок работы | разобраться в проекте → выбрать рецепт → подключить в одном месте → `gofmt`, `go build`, `go vet`, запуск |
| Рецепты | endpoint, репозиторий PostgreSQL, кэш Redis, события, задачи по расписанию, Telegram-бот, свой App, поле конфига, разовая команда |
| Справочник API | сигнатуры и ключи конфига всех модулей gorock-kit |
| Каркас проекта | `scripts/scaffold.sh` создаёт рабочий сервис со всеми слоями и `AGENTS.md` |

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

Что в `AGENTS.md`: команды сборки и запуска, куда класть код, контракты `App`, `Service` и маршрутов rockhandlers, правила сервисов, delivery, ошибок, логирования, событий и конфига, и список устаревшего API, которого не должно быть в коде.

---

## Как это проверено

Отдельная сессия Claude Code с плагином получила задачу из примера выше, без упоминания skill. Она сама подключила `gorock:service`, создала проект скриптом и добавила endpoints и репозиторий по рецептам. Сервис собрался без правок и отработал на PostgreSQL: создание, чтение, 404, 400, чистая остановка.

Плагин лежит в [репозитории gorock](https://github.com/arzab/gorock/tree/main/plugin). Версия плагина совпадает с версией gorock-kit, под которую написаны рецепты.
