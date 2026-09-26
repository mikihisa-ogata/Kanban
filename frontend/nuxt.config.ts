// https://nuxt.com/docs/api/configuration/nuxt-config
import tailwindcss from '@tailwindcss/vite'

export default defineNuxtConfig({
  srcDir: 'src',

  // 起動中の nuxt dev と .nuxt を取り合わないよう、検証用ビルドは別ディレクトリに出力できるようにする
  buildDir: process.env.NUXT_BUILD_DIR || '.nuxt',

  devtools: { enabled: true },

  // bff の CORS で許可しているポートに固定する
  devServer: {
    port: 3000
  },

  css: ['~/style.css'],

  app: {
    head: {
      title: 'Kanban',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' }
      ],
      link: [
        { rel: 'icon', href: '/favicon.ico', sizes: '32x32' },
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        { rel: 'icon', type: 'image/png', sizes: '32x32', href: '/favicon-32.png' },
        { rel: 'icon', type: 'image/png', sizes: '16x16', href: '/favicon-16.png' },
        { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' },
        { rel: 'manifest', href: '/site.webmanifest' }
      ]
    }
  },

  vite: {
    plugins: [
      tailwindcss(),
    ],
  },

  nitro: {
    prerender: {
      crawlLinks: true
    }
  }
})
