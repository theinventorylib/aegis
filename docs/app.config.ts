export default defineAppConfig({
  docus: {
    title: 'Aegis',
    description: 'A lightweight authentication framework for Go with a modular plugin architecture.',
    image: '/logo.png',
    url: 'https://theinventorylib.github.io/aegis',
    socials: {
      github: 'theinventorylib/aegis',
    },
    aside: {
      level: 0,
      collapsed: false,
      exclude: ['changelog', 'community']
    },
    header: {
      logo: true,
      showLinkIcon: true,
      exclude: [],
      fluid: true,
      search: true,
      colorMode: true,
      links: [
        {
          label: 'Documentation',
          to: '/get-started/introduction',
          icon: 'i-lucide-book-open'
        },
        {
          label: 'Concepts',
          to: '/concepts/core-concepts',
          icon: 'i-lucide-layers'
        },
        {
          label: 'Reference',
          to: '/reference/api-reference',
          icon: 'i-lucide-cpu'
        },
        {
          label: 'Changelog',
          to: '/changelog',
          icon: 'i-lucide-history'
        }
      ]
    },
    footer: {
      credits: {
        icon: 'i-lucide-shield',
        text: 'Aegis Auth Framework',
        href: 'https://github.com/theinventorylib/aegis'
      },
      textLinks: [
        {
          text: 'GitHub',
          href: 'https://github.com/theinventorylib/aegis',
          target: '_blank',
          rel: 'noopener'
        },
        {
          text: 'The Inventory',
          href: 'https://github.com/theinventorylib',
          target: '_blank'
        }
      ],
      iconLinks: [
        {
          href: 'https://github.com/theinventorylib/aegis',
          icon: 'simple-icons-github',
          label: 'GitHub'
        }
      ]
    }
  },
  ui: {
    colors: {
      primary: 'indigo',
      neutral: 'zinc'
    },
    prose: {
      codeIcon: {
        terminal: 'i-ph-terminal-window-duotone'
      }
    }
  },
  seo: {
    titleTemplate: '%s | Aegis Auth',
    meta: [
      { name: 'twitter:card', content: 'summary_large_image' },
      { name: 'twitter:site', content: '@theinventorylib' },
      { property: 'og:type', content: 'website' },
      { property: 'og:locale', content: 'en_US' },
      { name: 'theme-color', content: '#6366f1' },
      { name: 'author', content: 'The Inventory' }
    ]
  }
})
