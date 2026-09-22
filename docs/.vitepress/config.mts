import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'LazyFire',
  description: 'A keyboard-driven terminal app for Firebase: browse projects, databases and Firestore documents, run queries, and read Cloud Functions logs.',
  base: '/lazyfire/',

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/lazyfire/logo.svg' }],
    ['meta', { name: 'theme-color', content: '#161719' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.googleapis.com' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' }],
    ['link', { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=Recursive:slnt,wght,CASL,CRSV,MONO@-15..0,300..1000,0..1,0..1,0..1&display=swap' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'LazyFire - Firebase in your terminal' }],
    ['meta', { property: 'og:description', content: 'Browse projects, databases and Firestore documents, run queries, and read Cloud Functions logs from your terminal.' }],
    ['meta', { property: 'og:image', content: 'https://marjoballabani.github.io/lazyfire/preview.gif' }],
    ['meta', { property: 'og:url', content: 'https://marjoballabani.github.io/lazyfire/' }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { name: 'twitter:title', content: 'LazyFire - Firebase in your terminal' }],
    ['meta', { name: 'twitter:description', content: 'Browse projects, databases and Firestore documents, run queries, and read Cloud Functions logs from your terminal.' }],
    ['meta', { name: 'twitter:image', content: 'https://marjoballabani.github.io/lazyfire/preview.gif' }],
    ['meta', { name: 'keywords', content: 'firebase, firestore, tui, terminal, cli, cloud functions, go, lazygit, vim' }],
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
          { text: 'Releases', link: 'https://github.com/marjoballabani/lazyfire/releases' },
          { text: 'Changelog', link: 'https://github.com/marjoballabani/lazyfire/blob/main/CHANGELOG.md' }
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
            { text: 'Databases', link: '/guide/databases' },
            { text: 'Collections & Documents', link: '/guide/collections' },
            { text: 'Cloud Functions', link: '/guide/cloud-functions' },
            { text: 'Storage, Auth, Rules & Indexes', link: '/guide/storage-auth-rules' },
            { text: 'Query Builder', link: '/guide/query-builder' },
            { text: 'Visual Select Mode', link: '/guide/select-mode' },
            { text: 'Document Stats', link: '/guide/document-stats' },
            { text: 'Collection Health Scan', link: '/guide/collection-health-scan' },
            { text: 'Filtering & Search', link: '/guide/filtering' }
          ]
        },
        {
          text: 'Advanced',
          items: [
            { text: 'Emulator Mode', link: '/guide/emulator-mode' }
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
