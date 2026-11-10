---
layout: home

hero:
  name: GOROCK
  text: Шаблон Go-проекта
  tagline: Сервисы, обработчики, App, сборка и настройки — у каждого слоя одна задача. Бизнес-логика пишется один раз и работает в HTTP, очереди и по расписанию.
  actions:
    - theme: brand
      text: Архитектура
      link: /architecture/
    - theme: alt
      text: Быстрый старт
      link: /guide/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/arzab/gorock

features:
  - icon: 🧭
    title: Один шаблон для любого проекта
    details: HTTP-API, consumer очереди, бот, задачи по расписанию — устройство одно. Пришёл в новый GOROCK-проект — уже знаешь, где искать.
    link: /architecture/
    linkText: Архитектура

  - icon: 🔁
    title: Бизнес не знает транспорт
    details: Обработчики устроены как во Fiber — параметры и ответ приходят аргументами, `c.Next()` ведёт по цепочке. HTTP, очередь и cron вызывают один и тот же обработчик, а тест — просто вызов функции.
    link: /guide/handlers
    linkText: Как писать Handler

  - icon: 🧰
    title: Сервисы в одном каталоге
    details: База, кэш, внешние API, логгер — сервисы в одном каталоге. Открываются до запуска App, закрываются после, берутся по имени откуда угодно.
    link: /guide/services
    linkText: Как писать Service

  - icon: 📦
    title: Ядро и каталог
    details: gorock-kit делает одинаковую для всех часть — движок App, строгий конфиг, каталог сервисов, обработчики. Каталог даёт готовые HTTP, PostgreSQL, Redis и другие модули.
    link: /gorock-kit/
    linkText: Модули

  - icon: 🤖
    title: Для ИИ-агентов
    details: Плагин Claude Code и AGENTS.md учат агента писать код по правилам GOROCK на актуальном API.
    link: /ai/
    linkText: Подробнее
---
