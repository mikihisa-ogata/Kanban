<template>
  <div class="min-h-screen bg-gradient-to-br from-blue-500 to-blue-800">
    <!-- スクロールしてもハンバーガーメニューを押せるよう、ヘッダーは画面上部に固定する（高さはサイドバーの位置と合わせる） -->
    <header class="sticky top-0 z-40 h-17 bg-white bg-opacity-95 backdrop-blur-md px-8 flex items-center gap-4">
      <button
        type="button"
        class="bg-transparent border-0 p-1.5 rounded-lg text-blue-900 cursor-pointer hover:bg-blue-50 transition-all duration-200"
        :title="sidebarOpen ? 'スペース一覧を閉じる' : 'スペース一覧を開く'"
        :aria-expanded="sidebarOpen"
        @click="sidebarOpen = !sidebarOpen"
      >
        <svg class="w-6 h-6 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <path d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
      <h1 class="text-2xl font-bold text-blue-900">Kanban</h1>
    </header>
    <slot />
  </div>
</template>

<script setup>
// サイドバー（スペース一覧）の開閉状態。KanbanBoard と共有する
const sidebarOpen = useState('sidebarOpen', () => false);
</script>
