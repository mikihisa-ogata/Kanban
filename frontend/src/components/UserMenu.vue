<template>
  <!-- ヘッダー右上のアバター。押すとユーザーの情報・テーマの切り替え・ログアウトのメニューを開く -->
  <div ref="root" class="relative">
    <button
      type="button"
      class="flex items-center bg-transparent border-0 p-0.5 rounded-full cursor-pointer hover:ring-4 hover:ring-accent/25 transition-all duration-150"
      :title="`${user.DisplayName}（${user.Email}）`"
      aria-haspopup="menu"
      :aria-expanded="open"
      @click="open = !open"
    >
      <UserAvatar :user="user" />
    </button>
    <div
      v-if="open"
      class="absolute right-0 mt-2 w-64 bg-surface border border-line rounded-lg shadow-lg py-2 z-50"
      role="menu"
    >
      <div class="flex items-center gap-3 px-4 pb-3 mb-1 border-b border-line">
        <UserAvatar :user="user" size="lg" />
        <div class="min-w-0">
          <p class="m-0 text-sm font-semibold text-fg truncate">{{ user.DisplayName }}</p>
          <p class="m-0 text-xs text-muted truncate">{{ user.Email }}</p>
        </div>
      </div>
      <!-- テーマ -->
      <div class="px-4 pt-1 pb-2 mb-1 border-b border-line" role="group" aria-labelledby="theme-menu-label">
        <p id="theme-menu-label" class="m-0 mb-1.5 text-xs font-semibold text-muted">テーマ</p>
        <div class="flex p-0.5 rounded-md bg-raised">
          <button
            v-for="option in themeOptions"
            :key="option.value"
            type="button"
            role="menuitemradio"
            :aria-checked="theme === option.value"
            :class="[
              'flex-1 px-1 py-1 border-0 rounded text-xs font-semibold cursor-pointer whitespace-nowrap',
              theme === option.value ? 'bg-surface text-fg shadow-sm' : 'bg-transparent text-muted hover:text-fg'
            ]"
            @click="setTheme(option.value)"
          >
            {{ option.value === 'system' ? 'システム' : option.label }}
          </button>
        </div>
      </div>
      <button
        type="button"
        class="w-full text-left bg-transparent border-0 px-4 py-2 text-sm text-fg cursor-pointer hover:bg-raised disabled:opacity-50 disabled:cursor-not-allowed"
        role="menuitem"
        :disabled="loggingOut"
        @click="logout"
      >
        ログアウト
      </button>
      <p v-if="error" class="m-0 px-4 pt-1 text-xs text-danger">{{ error }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue';
import UserAvatar from '~/components/UserAvatar.vue';
import { authApi } from '~/api/auth';
import { useCurrentUser } from '~/composables/useCurrentUser';
import { useTheme } from '~/composables/useTheme';

defineProps({
  // ログイン中のユーザー
  user: {
    type: Object,
    required: true
  }
});

const { user: currentUser } = useCurrentUser();
const { theme, setTheme, options: themeOptions } = useTheme();
const root = ref(null);
const open = ref(false);
const loggingOut = ref(false);
const error = ref('');

const logout = async () => {
  loggingOut.value = true;
  error.value = '';
  try {
    await authApi.logout();
    open.value = false;
    currentUser.value = null;
    await navigateTo('/login');
  } catch (err) {
    error.value = 'ログアウトに失敗しました: ' + err.message;
    console.error('Failed to log out:', err);
  } finally {
    loggingOut.value = false;
  }
};

// メニューの外を押したら閉じる
const closeOnOutsideClick = (event) => {
  if (open.value && root.value && !root.value.contains(event.target)) {
    open.value = false;
  }
};
// Esc で閉じる
const closeOnEscape = (event) => {
  if (event.key === 'Escape') {
    open.value = false;
  }
};
onMounted(() => {
  document.addEventListener('click', closeOnOutsideClick);
  document.addEventListener('keydown', closeOnEscape);
});
onBeforeUnmount(() => {
  document.removeEventListener('click', closeOnOutsideClick);
  document.removeEventListener('keydown', closeOnEscape);
});
</script>
