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
      { text: 'Примеры', link: '/examples/' },
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
          text: 'gorock-kit',
          items: [
            { text: 'Введение', link: '/gorock-kit/' },
          ]
        },
        {
          text: 'Engine',
          items: [
            { text: 'rockengine', link: '/gorock-kit/rockengine' },
            { text: 'rockconfig', link: '/gorock-kit/rockconfig' },
          ]
        },
        {
          text: 'Apps',
          items: [
            { text: 'rockfiber — HTTP', link: '/gorock-kit/rockfiber' },
            { text: 'rockbus — события', link: '/gorock-kit/rockbus' },
            { text: 'rockcron — расписание', link: '/gorock-kit/rockcron' },
            { text: 'rocktelebot — Telegram', link: '/gorock-kit/rocktelebot' },
          ]
        },
        {
          text: 'Toolkit',
          items: [
            { text: 'rockbun — PostgreSQL', link: '/gorock-kit/rockbun' },
            { text: 'rockredis — Redis', link: '/gorock-kit/rockredis' },
            { text: 'rocklog — логгер', link: '/gorock-kit/rocklog' },
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
