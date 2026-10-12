<template>
  <!-- ヘッダー右上のアバター。押すとユーザーの情報とログアウトのメニューを開く -->
  <div ref="root" class="relative">
    <button
      type="button"
      class="flex items-center bg-transparent border-0 p-0.5 rounded-full cursor-pointer hover:ring-4 hover:ring-blue-100 transition-all duration-150"
      :title="`${user.DisplayName}（${user.Email}）`"
      aria-haspopup="menu"
      :aria-expanded="open"
      @click="open = !open"
    >
      <UserAvatar :user="user" />
    </button>
    <div
      v-if="open"
      class="absolute right-0 mt-2 w-64 bg-white border border-gray-200 rounded-lg shadow-lg py-2 z-50"
      role="menu"
    >
      <div class="flex items-center gap-3 px-4 pb-3 mb-1 border-b border-gray-200">
        <UserAvatar :user="user" size="lg" />
        <div class="min-w-0">
          <p class="m-0 text-sm font-semibold text-gray-900 truncate">{{ user.DisplayName }}</p>
          <p class="m-0 text-xs text-gray-500 truncate">{{ user.Email }}</p>
        </div>
      </div>
      <button
        type="button"
        class="w-full text-left bg-transparent border-0 px-4 py-2 text-sm text-gray-700 cursor-pointer hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed"
        role="menuitem"
        :disabled="loggingOut"
        @click="logout"
      >
        ログアウト
      </button>
      <p v-if="error" class="m-0 px-4 pt-1 text-xs text-red-600">{{ error }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue';
import UserAvatar from '~/components/UserAvatar.vue';
import { authApi } from '~/api/auth';
import { useCurrentUser } from '~/composables/useCurrentUser';

defineProps({
  // ログイン中のユーザー
  user: {
    type: Object,
    required: true
  }
});

const { user: currentUser } = useCurrentUser();
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
