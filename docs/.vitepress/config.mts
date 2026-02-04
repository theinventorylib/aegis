import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "Aegis",
  description: "A lightweight, modular authentication framework for Go.",
  head: [['link', { rel: 'icon', href: '/logo.png' }]],
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    logo: '/logo.png',
    
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Plugins', link: '/plugins/' },
      { text: 'API', link: 'https://pkg.go.dev/github.com/theinventorylib/aegis' }
    ],

    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Architecture', link: '/guide/architecture' },
          { text: 'Core Concepts', link: '/guide/concepts' }
        ]
      },
      {
        text: 'Plugins',
        items: [
          { text: 'Overview', link: '/plugins/' },
          { text: 'Admin', link: '/plugins/admin' },
          { text: 'Bearer Token', link: '/plugins/bearer' },
          { text: 'Email', link: '/plugins/email' },
          { text: 'JWT', link: '/plugins/jwt' },
          { text: 'OAuth', link: '/plugins/oauth' },
          { text: 'OpenAPI', link: '/plugins/openapi' },
          { text: 'Organizations', link: '/plugins/organizations' },
          { text: 'SMS', link: '/plugins/sms' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/theinventorylib/aegis' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024-present The Inventory'
    },

    search: {
      provider: 'local'
    }
  }
})
