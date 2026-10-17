// https://nuxt.com/docs/api/configuration/nuxt-config
import tailwindcss from '@tailwindcss/vite'
import { THEME_INIT_SCRIPT } from './src/utils/theme'

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
      ],
      // 選んだテーマを、画面を描く前に <html> に付ける（読み込み直後のちらつきを防ぐ）
      script: [
        { innerHTML: THEME_INIT_SCRIPT, tagPosition: 'head' }
      ]
    }
  },

  vite: {
    plugins: [
      tailwindcss(),
    ],
    // サーバー描画側の Vite が開く HMR の WebSocket（既定 24678 番）を、同じネットワークの他の端末から
    // 接続できないよう localhost だけで待ち受ける。ポートは Nuxt があとから決めるので、その後（enforce: 'post'）に指定する。
    // （ブラウザ側の HMR は 3000 番の dev サーバーに相乗りするので設定は要らない）
    $server: {
      plugins: [
        {
          name: 'kanban:server-hmr-localhost',
          enforce: 'post',
          config(config) {
            if (config.server && typeof config.server.hmr === 'object') {
              config.server.hmr.host = 'localhost'
            }
          }
        }
      ]
    }
  },

  nitro: {
    prerender: {
      crawlLinks: true
    }
  }
})
