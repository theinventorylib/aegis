import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
    site: 'https://theinventorylib.github.io',
    base: '/aegis',
    integrations: [
        starlight({
            title: 'Aegis',
            description: 'A lightweight, modular authentication framework for Go.',
            logo: {
                src: './src/assets/logo.png',
            },
            social: {
                github: 'https://github.com/theinventorylib/aegis',
            },
            sidebar: [
                {
                    label: 'Introduction',
                    items: [
                        { label: 'Getting Started', link: '/guide/getting-started/' },
                        { label: 'Architecture', link: '/guide/architecture/' },
                        { label: 'Core Concepts', link: '/guide/concepts/' },
                    ],
                },
                {
                    label: 'Plugins',
                    items: [
                        { label: 'Overview', link: '/plugins/' },
                        { label: 'Admin', link: '/plugins/admin/' },
                        { label: 'Bearer Token', link: '/plugins/bearer/' },
                        { label: 'Email', link: '/plugins/email/' },
                        { label: 'JWT', link: '/plugins/jwt/' },
                        { label: 'OAuth', link: '/plugins/oauth/' },
                        { label: 'OpenAPI', link: '/plugins/openapi/' },
                        { label: 'Organizations', link: '/plugins/organizations/' },
                        { label: 'SMS', link: '/plugins/sms/' },
                    ],
                },
                {
                    label: 'API',
                    items: [
                        { label: 'Go Reference', link: 'https://pkg.go.dev/github.com/theinventorylib/aegis', attrs: { target: '_blank' } }
                    ]
                }
            ],
            customCss: [
                // Path to custom CSS if we need it later
            ],
        }),
    ],
});
