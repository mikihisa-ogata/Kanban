<template>
  <!-- ログイン画面・サインアップ画面。ヘッダーを出さず、フォームを中央に置く -->
  <div class="min-h-screen bg-canvas flex flex-col items-center justify-center px-4 py-12">
    <h1 class="text-3xl font-bold text-fg mb-6">Kanban</h1>
    <div class="w-full max-w-sm bg-surface border border-line rounded-xl shadow-xl px-8 py-8">
      <slot />
    </div>
  </div>
</template>
