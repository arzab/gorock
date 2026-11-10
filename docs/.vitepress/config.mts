import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'GOROCK',
  description: 'Шаблон Go-проекта',
  lang: 'ru-RU',
  base: '/gorock/',

  themeConfig: {
    nav: [
      { text: 'Архитектура', link: '/architecture/' },
      { text: 'Руководство', link: '/guide/getting-started' },
      { text: 'gorock-kit', link: '/gorock-kit/' },
      { text: 'Каталог', link: '/catalog/' },
      { text: 'Для ИИ', link: '/ai/' },
    ],

    sidebar: {
      '/architecture/': [
        {
          text: 'Архитектура',
          items: [
            { text: 'Архитектура', link: '/architecture/' },
            { text: 'Быстрый старт', link: '/guide/getting-started' },
          ]
        },
      ],
      '/guide/': [
        {
          text: 'Руководство',
          items: [
            { text: 'Быстрый старт', link: '/guide/getting-started' },
            { text: 'Как писать Service', link: '/guide/services' },
            { text: 'Как писать Handler', link: '/guide/handlers' },
            { text: 'Как писать App', link: '/guide/apps' },
            { text: 'Main', link: '/guide/main' },
            { text: 'Как писать Config', link: '/guide/configs' },
            { text: 'Models и libs', link: '/guide/models' },
          ]
        },
        {
          text: 'Дальше',
          items: [
            { text: 'Как написать свой App', link: '/gorock-kit/rockhandlers-transport' },
            { text: 'Примеры', link: '/examples/' },
            { text: 'CLI', link: '/cli/' },
          ]
        },
      ],
      '/gorock-kit/': [
        {
          text: 'gorock-kit — ядро',
          items: [
            { text: 'Введение', link: '/gorock-kit/' },
            { text: 'rockengine — движок App', link: '/gorock-kit/rockengine' },
            { text: 'rockconfig — конфиг', link: '/gorock-kit/rockconfig' },
            { text: 'rockservices — сервисы', link: '/gorock-kit/rockservices' },
            { text: 'rockhandlers — обработчики', link: '/gorock-kit/rockhandlers' },
            { text: 'Как написать свой App', link: '/gorock-kit/rockhandlers-transport' },
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
