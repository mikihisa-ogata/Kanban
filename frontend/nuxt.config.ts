// https://nuxt.com/docs/api/configuration/nuxt-config
import tailwindcss from '@tailwindcss/vite'

export default defineNuxtConfig({
  srcDir: 'src',

  devtools: { enabled: true },

  css: ['~/style.css'],

  app: {
    head: {
      title: 'Kanban Board',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' }
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
