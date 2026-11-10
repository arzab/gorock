<div align="center">

# GOROCK

**Шаблон Go-проекта**

Go минималистичен. Бэкенд шаблонен. GOROCK соединяет эти две идеи — даёт стандарт там где он нужен, и оставляет свободу там где она важна.

[![Docs](https://img.shields.io/badge/docs-arzab.github.io/gorock-blue)](https://arzab.github.io/gorock/)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

</div>

---

## Идея

Любой бэкенд делает одно и то же: к нему обращаются, он решает, что делать, ходит в базы и внешние API и отвечает. GOROCK — шаблон такого проекта.

```
[клиент, брокер, расписание] ⇄ App ⇄ Handlers ⇄ Services ⇄ [база, API, файлы, логи]

        Main — одинаковая сборка · Config — настройки и константы проекта
```

- **Services** — инструменты: обёртки над технологиями и логикой, в одном каталоге, по имени.
- **Handlers** — бизнес-логика: параметры на входе, результат или ошибка с видом на выходе, без привязки к технологии.
- **App** — технология обращения: HTTP, очередь, cron, бот; один обработчик — в нескольких App.
- **Main** и **Config** — сборка в правильном порядке и строгие настройки.

Открываешь любой GOROCK-проект — сразу знаешь, где искать код.

---

## Документация

**[arzab.github.io/gorock](https://arzab.github.io/gorock/)**

| Раздел | |
|--------|-|
| [Архитектура](https://arzab.github.io/gorock/architecture/) | Слои, зачем каждый и как они работают вместе |
| [Быстрый старт](https://arzab.github.io/gorock/guide/getting-started) | Проект заказов на PostgreSQL за 10 минут |
| [Руководство](https://arzab.github.io/gorock/guide/services) | Как писать Service, Handler, App, Main, Config |
| [gorock-kit](https://arzab.github.io/gorock/gorock-kit/) | Ядро: rockservices, rockhandlers, rockengine, rockconfig |
| [Каталог](https://arzab.github.io/gorock/catalog/) | HTTP (rockhttp), PostgreSQL, Redis, логгер, события, расписание, Telegram |
| [Примеры](https://arzab.github.io/gorock/examples/) | [examples/taskapi](examples/taskapi) — HTTP-сервис задач на PostgreSQL, v0.5.0 |

---

## Для ИИ-агентов

Плагин для Claude Code — skill `gorock:service` создаёт проект и добавляет код по правилам GOROCK на актуальном API gorock-kit:

```sh
claude plugin marketplace add arzab/gorock
claude plugin install gorock@gorock
```

Для Codex, Cursor и других агентов — [`AGENTS.md`](plugin/skills/service/templates/AGENTS.md) в корень проекта. Подробнее — [Для ИИ-агентов](https://arzab.github.io/gorock/ai/).

---

## Экосистема

| Репозиторий | Описание |
|-------------|----------|
| [arzab/gorock](https://github.com/arzab/gorock) | Архитектурный стандарт и документация |
| [arzab/gorock-kit](https://github.com/arzab/gorock-kit) | Ядро: rockservices, rockhandlers, rockengine, rockconfig · v0.5.0, Go 1.26+ |
| [arzab/gorock-catalog](https://github.com/arzab/gorock-catalog) | Каталог: rockhttp и модули под конкретные технологии · v0.5.0 |
| arzab/gorock-cli | Генератор слоёв и сущностей *(в разработке)* |
