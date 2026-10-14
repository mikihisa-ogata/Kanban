<template>
  <!-- ログイン後の画面。上部のヘッダーは置かず、サイドバー（SpaceShell）にスペースの切り替え・作成・通知・ユーザーメニューをまとめる -->
  <div class="min-h-screen bg-canvas text-fg">
    <slot />
    <GlobalCreate />
  </div>
</template>

<script setup>
import GlobalCreate from '~/components/GlobalCreate.vue';
</script>
