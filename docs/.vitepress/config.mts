import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'LazyFire',
  description: 'A TUI Firebase browser for the terminal',
  base: '/lazyfire/',

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }],
    ['meta', { name: 'theme-color', content: '#ff6f00' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'LazyFire' }],
    ['meta', { property: 'og:description', content: 'A TUI Firebase browser for the terminal' }],
  ],

  themeConfig: {
    logo: '/logo.svg',

    nav: [
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Reference', link: '/reference/keybindings' },
      {
        text: 'Links',
        items: [
          { text: 'GitHub', link: 'https://github.com/marjoballabani/lazyfire' },
          { text: 'Releases', link: 'https://github.com/marjoballabani/lazyfire/releases' }
        ]
      }
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'Introduction',
          items: [
            { text: 'Getting Started', link: '/guide/getting-started' },
            { text: 'Installation', link: '/guide/installation' },
            { text: 'Configuration', link: '/guide/configuration' }
          ]
        },
        {
          text: 'Features',
          items: [
            { text: 'Navigation', link: '/guide/navigation' },
            { text: 'Collections & Documents', link: '/guide/collections' },
            { text: 'Cloud Functions', link: '/guide/cloud-functions' },
            { text: 'Query Builder', link: '/guide/query-builder' },
            { text: 'Visual Select Mode', link: '/guide/select-mode' },
            { text: 'Document Stats', link: '/guide/document-stats' },
            { text: 'Filtering & Search', link: '/guide/filtering' }
          ]
        }
      ],
      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'Keybindings', link: '/reference/keybindings' },
            { text: 'Themes', link: '/reference/themes' },
            { text: 'CLI Options', link: '/reference/cli-options' }
          ]
        }
      ]
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/marjoballabani/lazyfire' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024-present Marjo Ballabani'
    },

    search: {
      provider: 'local'
    },

    editLink: {
      pattern: 'https://github.com/marjoballabani/lazyfire/edit/main/docs/:path',
      text: 'Edit this page on GitHub'
    }
  }
})
