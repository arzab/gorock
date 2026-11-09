import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'GOROCK',
  description: 'Стандарт структуры Go-проекта',
  lang: 'ru-RU',
  base: '/gorock/',

  themeConfig: {
    nav: [
      { text: 'Архитектура', link: '/architecture/' },
      { text: 'gorock-kit', link: '/gorock-kit/' },
      { text: 'Каталог', link: '/catalog/' },
      { text: 'Примеры', link: '/examples/' },
      { text: 'Для ИИ', link: '/ai/' },
      { text: 'CLI', link: '/cli/' },
    ],

    sidebar: {
      '/architecture/': [
        {
          text: 'Введение',
          items: [
            { text: 'Что такое GOROCK', link: '/architecture/' },
            { text: 'Концепты', link: '/architecture/concepts' },
            { text: 'Быстрый старт', link: '/architecture/getting-started' },
          ]
        },
        {
          text: 'Toolkit',
          items: [
            { text: 'Обзор', link: '/architecture/toolkit' },
            { text: 'Services', link: '/architecture/services' },
            { text: 'Infra', link: '/architecture/infra' },
            { text: 'Libs', link: '/architecture/libs' },
          ]
        },
        {
          text: 'Realm',
          items: [
            { text: 'Обзор', link: '/architecture/realm' },
            { text: 'Models', link: '/architecture/models' },
            { text: 'Delivery', link: '/architecture/delivery' },
            { text: 'Utils', link: '/architecture/utils' },
          ]
        },
        {
          text: 'Engine',
          items: [
            { text: 'Обзор', link: '/architecture/engine' },
            { text: 'Main', link: '/architecture/main' },
            { text: 'Apps', link: '/architecture/apps' },
          ]
        },
        {
          text: 'Конфигурация',
          items: [
            { text: 'Configs', link: '/architecture/configs' },
          ]
        },
        {
          text: 'Итог',
          items: [
            { text: 'Итог', link: '/architecture/summary' },
          ]
        },
      ],

      '/gorock-kit/': [
        {
          text: 'gorock-kit — ядро',
          items: [
            { text: 'Введение', link: '/gorock-kit/' },
            { text: 'rockengine — apps', link: '/gorock-kit/rockengine' },
            { text: 'rockconfig — конфиг', link: '/gorock-kit/rockconfig' },
            { text: 'rockservices — сервисы', link: '/gorock-kit/rockservices' },
            { text: 'rockhandlers — обработка запросов', link: '/gorock-kit/rockhandlers' },
          ]
        },
      ],
      '/catalog/': [
        {
          text: 'Каталог',
          items: [
            { text: 'Обзор', link: '/catalog/' },
          ]
        },
        {
          text: 'Apps',
          items: [
            { text: 'rockhttp — HTTP', link: '/catalog/rockhttp' },
            { text: 'rockfiber — HTTP (Fiber)', link: '/catalog/rockfiber' },
            { text: 'rockbus — события', link: '/catalog/rockbus' },
            { text: 'rockcron — расписание', link: '/catalog/rockcron' },
            { text: 'rocktelebot — Telegram', link: '/catalog/rocktelebot' },
          ]
        },
        {
          text: 'Services и Infra',
          items: [
            { text: 'rockbun — PostgreSQL', link: '/catalog/rockbun' },
            { text: 'rockredis — Redis', link: '/catalog/rockredis' },
            { text: 'rocklog — логгер', link: '/catalog/rocklog' },
          ]
        },
      ],
      '/examples/': [
        {
          text: 'Примеры',
          items: [
            { text: 'Все примеры', link: '/examples/' },
          ]
        },
      ],

      '/ai/': [
        {
          text: 'Для ИИ-агентов',
          items: [
            { text: 'Плагин и AGENTS.md', link: '/ai/' },
          ]
        },
      ],

      '/cli/': [
        {
          text: 'CLI',
          items: [
            { text: 'Введение', link: '/cli/' },
          ]
        }
      ],
    },

    outline: { level: [2, 3], label: 'На этой странице' },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/arzab/gorock' }
    ]
  }
})
