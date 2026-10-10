<template>
  <div class="min-h-screen bg-gradient-to-br from-blue-500 to-blue-800">
    <!-- スクロールしてもハンバーガーメニューを押せるよう、ヘッダーは画面上部に固定する（高さはサイドバーの位置と合わせる） -->
    <header class="sticky top-0 z-40 h-17 bg-white bg-opacity-95 backdrop-blur-md border-b border-gray-200 px-8 flex items-center gap-4">
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
      <!-- Jira と同じく、どのページからでも課題を作成できる -->
      <button
        type="button"
        class="ml-2 px-4 py-1.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700 transition-colors duration-150"
        @click="createIssueOpen = true"
      >
        + 作成
      </button>
    </header>
    <slot />
    <GlobalCreate />
  </div>
</template>

<script setup>
import GlobalCreate from '~/components/GlobalCreate.vue';

// サイドバー（スペース一覧）の開閉状態。SpaceShell と共有する
const sidebarOpen = useState('sidebarOpen', () => false);
// 課題の作成モーダルの開閉状態。GlobalCreate と共有する
const createIssueOpen = useState('createIssueOpen', () => false);
</script>
