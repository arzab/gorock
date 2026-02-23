<div align="center">

# GOROCK

**Прагматичная Go-архитектура**

Go минималистичен. Бэкенд шаблонен. GOROCK соединяет эти две идеи — даёт стандарт там где он нужен, и оставляет свободу там где она важна.

[![Docs](https://img.shields.io/badge/docs-arzab.github.io/gorock-blue)](https://arzab.github.io/gorock/)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

</div>

---

## Идея

Три слоя. Каждый знает своё место.

```
project/
├── cmd/        — Engine    — когда и как запустить
├── internal/   — Realm     — что происходит внутри
└── pkg/        — Toolkit   — инструменты без бизнес-логики
```

Открываешь любой GOROCK-проект — сразу знаешь где искать код.
Работает для HTTP-сервера, consumer'а, worker'а, scheduler'а, CLI.

---

## Документация

**[arzab.github.io/gorock](https://arzab.github.io/gorock/)**

| Раздел | |
|--------|-|
| [Что такое GOROCK](https://arzab.github.io/gorock/architecture/) | Идея, три слоя, структура проекта |
| [Концепты](https://arzab.github.io/gorock/architecture/concepts) | Пять принципов архитектуры |
| [Engine](https://arzab.github.io/gorock/architecture/engine) | Запуск, lifecycle, Apps |
| [Realm](https://arzab.github.io/gorock/architecture/realm) | Бизнес-логика, Delivery, Models |
| [Toolkit](https://arzab.github.io/gorock/architecture/toolkit) | Services, Infra, Libs |
| [Конфигурация](https://arzab.github.io/gorock/architecture/configs) | Конфиги как документация |

---

## Экосистема

| Репозиторий | Описание |
|-------------|----------|
| [arzab/gorock](https://github.com/arzab/gorock) | Архитектурный стандарт и документация |
| arzab/gorock-kit | Готовые реализации для GOROCK-проектов *(скоро)* |
| arzab/gorock-cli | Генератор слоёв и сущностей *(скоро)* |
