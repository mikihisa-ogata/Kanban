<template>
  <div class="min-h-screen bg-canvas text-fg">
    <!-- スクロールしてもハンバーガーメニューを押せるよう、ヘッダーは画面上部に固定する（高さはサイドバーの位置と合わせる） -->
    <header class="sticky top-0 z-40 h-17 bg-sidebar/95 backdrop-blur-md border-b border-line px-8 flex items-center gap-4">
      <button
        type="button"
        class="bg-transparent border-0 p-1.5 rounded-lg text-fg cursor-pointer hover:bg-raised transition-all duration-200"
        :title="sidebarOpen ? 'スペース一覧を閉じる' : 'スペース一覧を開く'"
        :aria-expanded="sidebarOpen"
        @click="sidebarOpen = !sidebarOpen"
      >
        <svg class="w-6 h-6 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <path d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
      <h1 class="text-2xl font-bold text-fg">Kanban</h1>
      <!-- Jira と同じく、どのページからでも課題を作成できる -->
      <button
        type="button"
        class="ml-2 px-4 py-1.5 bg-accent text-on-accent border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-accent-hover transition-colors duration-150"
        @click="createIssueOpen = true"
      >
        + 作成
      </button>
      <!-- ログイン中のユーザーはブラウザ側でだけ取得するので、サーバーの描画には含めない -->
      <ClientOnly>
        <div v-if="currentUser" class="ml-auto flex items-center gap-3">
          <NotificationBell />
          <UserMenu :user="currentUser" />
        </div>
      </ClientOnly>
    </header>
    <slot />
    <GlobalCreate />
  </div>
</template>

<script setup>
import GlobalCreate from '~/components/GlobalCreate.vue';
import UserMenu from '~/components/UserMenu.vue';
import NotificationBell from '~/components/NotificationBell.vue';
import { useCurrentUser } from '~/composables/useCurrentUser';

// サイドバー（スペース一覧）の開閉状態。SpaceShell と共有する
const sidebarOpen = useState('sidebarOpen', () => false);
// 課題の作成モーダルの開閉状態。GlobalCreate と共有する
const createIssueOpen = useState('createIssueOpen', () => false);
// ログイン中のユーザー（ヘッダー右上のアバター）
const { user: currentUser } = useCurrentUser();
</script>
