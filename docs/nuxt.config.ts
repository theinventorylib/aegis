import tailwindcss from "@tailwindcss/vite";


// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  extends: ['docus'],
  modules: [
    '@nuxt/icon',
    '@nuxt/eslint',
    '@nuxt/ui',
    '@nuxtjs/sitemap',
    '@nuxtjs/robots',
    'nuxt-og-image'
  ],
  site: {
    url: 'https://theinventorylib.github.io/aegis',
    name: 'Aegis Auth Framework',
    description: 'A lightweight, modular authentication framework for Go with plugin architecture.',
    defaultLocale: 'en'
  },
  app: {
    baseURL: process.env.NODE_ENV === 'production' ? '/aegis/' : '/',
    head: {
      link: [
        { rel: 'icon', type: 'image/png', href: process.env.NODE_ENV === 'production' ? '/aegis/logo.png' : '/logo.png' }
      ]
    }
  },
  css: ['~/assets/css/main.css'],
  robots: {
    robotsTxt: false // We use static public/robots.txt
  },
  sitemap: {
    strictNuxtContentPaths: true
  },
  compatibilityDate: '2026-02-05',
  colorMode: {
    preference: 'dark',
    fallback: 'dark',
    classSuffix: ''
  },
  content: {
    build: {
      markdown: {
        highlight: {
          theme: {
            default: 'github-light',
            dark: 'github-dark'
          },
          langs: [
            'go',
            'typescript',
            'javascript',
            'bash',
            'shell',
            'sql',
            'yaml',
            'json',
            'markdown',
            'vue',
            'html',
            'css'
          ]
        }
      }
    }
  },
  vite: {
    plugins: [tailwindcss()]
  }
})


