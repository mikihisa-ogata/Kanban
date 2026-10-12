<template>
  <!-- ヘッダーのベル。未読の数を出し、押すと通知一覧を開く。通知は表示時と 30 秒ごとに取り直す -->
  <div ref="root" class="relative">
    <button
      type="button"
      class="relative flex items-center bg-transparent border-0 p-1.5 rounded-full text-blue-900 cursor-pointer hover:bg-blue-50 transition-colors duration-150"
      :title="unread ? `通知（未読 ${unread} 件）` : '通知'"
      aria-haspopup="dialog"
      :aria-expanded="open"
      @click="toggle"
    >
      <svg class="w-6 h-6 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9" />
        <path d="M13.73 21a2 2 0 0 1-3.46 0" />
      </svg>
      <span
        v-if="unread"
        class="absolute -top-0.5 -right-0.5 min-w-5 h-5 px-1 rounded-full bg-red-600 text-white text-[11px] font-bold leading-5 text-center"
        aria-hidden="true"
      >
        {{ unread > 99 ? '99+' : unread }}
      </span>
    </button>

    <div
      v-if="open"
      class="absolute right-0 mt-2 w-96 max-w-[calc(100vw-2rem)] bg-white border border-gray-200 rounded-lg shadow-lg z-50"
      role="dialog"
      aria-label="通知"
    >
      <div class="flex items-center gap-2 px-4 py-3 border-b border-gray-200">
        <h2 class="m-0 text-base font-bold text-gray-900">通知</h2>
        <label class="ml-auto flex items-center gap-1.5 text-xs text-gray-600 cursor-pointer select-none">
          <input v-model="unreadOnly" type="checkbox" class="cursor-pointer" />
          未読のみ
        </label>
        <button
          type="button"
          :disabled="!unread || busy"
          class="bg-transparent border-0 p-0 text-xs font-semibold text-blue-600 cursor-pointer hover:underline disabled:text-gray-400 disabled:no-underline disabled:cursor-not-allowed"
          @click="markAllRead"
        >
          すべて既読にする
        </button>
      </div>

      <p v-if="error" class="m-0 px-4 py-2 text-xs text-red-600">{{ error }}</p>
      <ul v-if="visible.length" class="m-0 p-0 list-none max-h-[28rem] overflow-y-auto">
        <li v-for="notification in visible" :key="notification.ID">
          <button
            type="button"
            :class="[
              'w-full flex items-start gap-3 px-4 py-3 border-0 border-b border-gray-100 text-left cursor-pointer',
              notification.Read ? 'bg-white hover:bg-gray-50' : 'bg-blue-50 hover:bg-blue-100'
            ]"
            @click="openNotification(notification)"
          >
            <UserAvatar v-if="notification.Actor" :user="notification.Actor" />
            <span v-else class="shrink-0 w-8 h-8 rounded-full bg-gray-200" aria-hidden="true" />
            <span class="min-w-0 flex-1">
              <span class="block text-sm text-gray-900">
                <span class="font-semibold">{{ notification.Actor ? notification.Actor.DisplayName : '誰か' }}</span>
                が
                <span class="font-mono font-semibold">{{ notification.IssueKey }}</span>
                {{ KIND_TEXTS[notification.Kind] ?? 'を変更しました' }}
              </span>
              <span class="block mt-0.5 text-xs text-gray-600 truncate">{{ notification.IssueTitle }}</span>
              <span class="block mt-0.5 text-xs text-gray-500 line-clamp-2 break-words">{{ describe(notification) }}</span>
              <time
                class="block mt-1 text-xs text-gray-400"
                :datetime="notification.CreatedAt"
                :title="formatDateTime(notification.CreatedAt)"
              >
                {{ formatRelativeTime(notification.CreatedAt, now) }}
              </time>
            </span>
            <span v-if="!notification.Read" class="shrink-0 mt-1.5 w-2 h-2 rounded-full bg-blue-600" aria-label="未読" />
          </button>
        </li>
      </ul>
      <p v-else-if="!error" class="m-0 px-4 py-8 text-sm text-gray-500 text-center">
        {{ unreadOnly ? '未読の通知はありません' : '通知はありません' }}
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue';
import UserAvatar from './UserAvatar.vue';
import { notificationsApi } from '../api/notifications';
import { historyFieldLabel } from '../utils/history';
import { formatDateTime, formatRelativeTime } from '../utils/datetime';
import { issuePath } from '../utils/spaceRoutes';

// 通知を取り直す間隔（WebSocket は使わずポーリングする）
const POLL_INTERVAL_MS = 30000;

const root = ref(null);
const open = ref(false);
const unreadOnly = ref(false);
const notifications = ref([]);
const unread = ref(0);
const error = ref('');
const busy = ref(false);
const now = ref(Date.now());

const visible = computed(() => (unreadOnly.value ? notifications.value.filter(n => !n.Read) : notifications.value));

// 通知の種類ごとの文（「山田 太郎 が KAN-1 ○○」の ○○）
const KIND_TEXTS = {
  updated: 'を変更しました',
  commented: 'にコメントしました',
  mentioned: 'であなたをメンションしました'
};

// 変わった項目名（「,」区切り）は表示名にし、コメント・メンションは本文の冒頭を「」で囲む
const describe = (notification) => {
  if (notification.Kind === 'commented' || notification.Kind === 'mentioned') {
    return `「${notification.Detail}」`;
  }
  return notification.Detail.split(',').filter(Boolean).map(historyFieldLabel).join('、');
};

const fetchNotifications = async () => {
  try {
    const result = await notificationsApi.fetchAll();
    notifications.value = result.Notifications;
    unread.value = result.Unread;
    now.value = Date.now();
    error.value = '';
  } catch (err) {
    error.value = '通知の取得に失敗しました';
    console.error('Failed to fetch notifications:', err);
  }
};

const toggle = () => {
  open.value = !open.value;
  if (open.value) {
    fetchNotifications();
  }
};

const markAllRead = async () => {
  busy.value = true;
  try {
    await notificationsApi.markAllRead();
    await fetchNotifications();
  } catch (err) {
    error.value = '既読にできませんでした';
    console.error('Failed to mark all notifications as read:', err);
  } finally {
    busy.value = false;
  }
};

// 押した通知を既読にして課題を開く。既読にできなくても課題は開く
const openNotification = async (notification) => {
  open.value = false;
  if (!notification.Read) {
    notification.Read = true;
    unread.value = Math.max(unread.value - 1, 0);
    notificationsApi.markRead(notification.ID).catch(err => {
      console.error('Failed to mark notification as read:', err);
    });
  }
  await navigateTo(issuePath(notification.IssueKey));
};

// ページを移ったときも取り直す（課題を開いた先の変更を早く反映する）
const route = useRoute();
watch(() => route.fullPath, () => {
  fetchNotifications();
});

// タブが裏にある間は取りに行かず、表に戻ったらすぐ取り直す
let timer = null;
const poll = () => {
  if (!document.hidden) {
    fetchNotifications();
  }
};
const handleVisibilityChange = () => {
  if (!document.hidden) {
    fetchNotifications();
  }
};

// 一覧の外を押すか Esc で閉じる
const closeOnOutsideClick = (event) => {
  if (open.value && root.value && !root.value.contains(event.target)) {
    open.value = false;
  }
};
const closeOnEscape = (event) => {
  if (event.key === 'Escape') {
    open.value = false;
  }
};

onMounted(() => {
  fetchNotifications();
  timer = setInterval(poll, POLL_INTERVAL_MS);
  document.addEventListener('visibilitychange', handleVisibilityChange);
  document.addEventListener('click', closeOnOutsideClick);
  document.addEventListener('keydown', closeOnEscape);
});
onBeforeUnmount(() => {
  clearInterval(timer);
  document.removeEventListener('visibilitychange', handleVisibilityChange);
  document.removeEventListener('click', closeOnOutsideClick);
  document.removeEventListener('keydown', closeOnEscape);
});
</script>
