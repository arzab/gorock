---
layout: home

hero:
  name: GOROCK
  text: Стандарт структуры Go-проекта
  tagline: Три слоя. Чёткие правила. Одна архитектура — для любого Go-приложения.
  actions:
    - theme: brand
      text: Быстрый старт
      link: /architecture/getting-started
    - theme: alt
      text: Архитектура
      link: /architecture/
    - theme: alt
      text: GitHub
      link: https://github.com/arzab/gorock

features:
  - icon: 🗂️
    title: Понятная структура
    details: Открываешь любой GOROCK-проект — сразу знаешь, где что лежит. Никакого «а куда это положить».

  - icon: 🧱
    title: Три слоя
    details: Engine запускает, Realm содержит бизнес-логику, Toolkit — переиспользуемый инструментарий. Каждый слой знает своё место.

  - icon: ⚙️
    title: Для любого приложения
    details: HTTP-сервер, consumer, worker, планировщик, бот — структура не меняется. GOROCK не привязан к транспорту.

  - icon: 📦
    title: gorock-kit
    details: Готовый код для того, что в каждом проекте одинаковое — движок с graceful shutdown, загрузка конфигов с env, логгер, HTTP-сервер на Fiber, шина событий, планировщик, PostgreSQL и Redis.
    link: /gorock-kit/
    linkText: Модули

  - icon: 🧪
    title: Рабочий пример
    details: Эталонный проект на gorock-kit — HTTP, PostgreSQL, события и расписание. Код в документации взят из него.
    link: /examples/
    linkText: Примеры

  - icon: ⚡
    title: CLI — в разработке
    details: Генерация слоёв, сервисов и endpoints одной командой. Пока — шаблоны в документации.
    link: /cli/
    linkText: Подробнее
---
