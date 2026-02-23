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
      { text: 'CLI', link: '/cli/' },
    ],

    sidebar: {
      '/architecture/': [
        {
          text: 'Введение',
          items: [
            { text: 'Что такое GOROCK', link: '/architecture/' },
            { text: 'Концепты', link: '/architecture/concepts' },
          ]
        },
        {
          text: 'Engine',
          items: [
            { text: 'Cmd', link: '/architecture/engine' },
            { text: 'Main', link: '/architecture/main' },
            { text: 'Apps', link: '/architecture/apps' },
          ]
        },
        {
          text: 'Realm',
          items: [
            { text: 'Internal', link: '/architecture/realm' },
            { text: 'Delivery', link: '/architecture/delivery' },
            { text: 'Models', link: '/architecture/models' },
          ]
        },
        {
          text: 'Toolkit',
          items: [
            { text: 'Pkg', link: '/architecture/toolkit' },
            { text: 'Services', link: '/architecture/services' },
            { text: 'Infra', link: '/architecture/infra' },
            { text: 'Libs', link: '/architecture/libs' },
          ]
        },
        {
          text: 'Конфигурация',
          items: [
            { text: 'Configs', link: '/architecture/configs' },
          ]
        },
      ],

      '/gorock-kit/': [
        {
          text: 'gorock-kit',
          items: [
            { text: 'Введение', link: '/gorock-kit/' },
          ]
        }
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

    socialLinks: [
      { icon: 'github', link: 'https://github.com/arzab/gorock' }
    ]
  }
})
