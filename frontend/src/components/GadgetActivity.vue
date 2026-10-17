<template>
  <!-- ダッシュボードの最近のアクティビティ。すべての課題の変更履歴を新しい順に出す（同じ操作で変わった項目はまとめる）。
       ほかのガジェットと高さが揃うよう、長いときは中でスクロールする -->
  <div>
    <p v-if="loading" class="m-0 text-sm text-muted">読み込み中...</p>
    <p v-else-if="error" class="m-0 text-sm text-danger">{{ error }}</p>
    <ul v-else-if="groups.length" class="max-h-[26rem] overflow-y-auto m-0 p-0 pr-1 list-none flex flex-col gap-3">
      <li v-for="group in groups" :key="group.key" class="flex gap-2">
        <UserAvatar v-if="group.user" :user="group.user" size="sm" class="shrink-0 mt-0.5" />
        <span v-else class="shrink-0 w-6 h-6 mt-0.5 rounded-full bg-raised" aria-hidden="true" />
        <div class="min-w-0 flex-1">
          <p class="m-0 text-sm text-fg">
            <span class="font-semibold">{{ group.user ? group.user.DisplayName : '（変更者なし）' }}</span>
            <span class="text-muted">が</span>
            <NuxtLink :to="issuePath(group.issueKey)" class="font-mono text-xs text-accent no-underline hover:underline">{{ group.issueKey }}</NuxtLink>
            <span class="ml-1">{{ group.issueTitle }}</span>
            <span class="text-muted">{{ group.created ? ' を作成しました' : ' を変更しました' }}</span>
          </p>
          <ul v-if="group.changes.length" class="mt-0.5 mb-0 p-0 list-none flex flex-col gap-0.5">
            <li v-for="entry in group.changes" :key="entry.ID" class="flex flex-wrap items-baseline gap-x-1.5 text-xs">
              <span class="shrink-0 font-semibold text-muted">{{ historyFieldLabel(entry.Field) }}</span>
              <span class="min-w-0 max-w-40 truncate text-muted line-through">{{ formatHistoryValue(entry.Field, entry.OldValue) }}</span>
              <span class="text-muted" aria-label="から">→</span>
              <span class="min-w-0 max-w-48 truncate text-fg">{{ formatHistoryValue(entry.Field, entry.NewValue) }}</span>
            </li>
          </ul>
          <time :datetime="group.time" :title="formatDateTime(group.time)" class="text-xs text-muted">{{ formatRelativeTime(group.time, now) }}</time>
        </div>
      </li>
    </ul>
    <p v-else class="m-0 text-sm text-muted">アクティビティはまだありません</p>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import UserAvatar from './UserAvatar.vue';
import { dashboardApi } from '../api/dashboard';
import { issuePath } from '../utils/spaceRoutes';
import { formatDateTime, formatRelativeTime } from '../utils/datetime';
import { historyFieldLabel, formatHistoryValue } from '../utils/history';

// 読む変更履歴の件数（まとめると表示はこれより少なくなる）
const LIMIT = 20;

const loading = ref(true);
const error = ref('');
const entries = ref([]);
const now = ref(Date.now());

// 同じ課題・同じ日時・同じ変更者の変更（同じ操作）を 1 つにまとめる。まとめた中の項目は記録順にする
const groups = computed(() => {
  const list = [];
  for (const entry of entries.value) {
    const created = entry.Field === 'created';
    const last = list[list.length - 1];
    if (!created && last && !last.created && last.todoId === entry.TodoID && last.time === entry.CreatedAt && last.userId === entry.UserID) {
      last.changes.unshift(entry);
      continue;
    }
    if (created && last && last.todoId === entry.TodoID && last.time === entry.CreatedAt && last.userId === entry.UserID) {
      // 作成と同時に記録した項目（あれば）は作成にまとめる
      last.created = true;
      last.changes = [];
      continue;
    }
    list.push({
      key: `history-${entry.ID}`, todoId: entry.TodoID, issueKey: entry.Key, issueTitle: entry.Title,
      time: entry.CreatedAt, userId: entry.UserID, user: entry.User, created, changes: created ? [] : [entry]
    });
  }
  return list;
});

const load = async () => {
  loading.value = true;
  error.value = '';
  try {
    entries.value = await dashboardApi.fetchActivity(LIMIT);
    now.value = Date.now();
  } catch (err) {
    error.value = 'アクティビティの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch activity:', err);
  } finally {
    loading.value = false;
  }
};
onMounted(load);

defineExpose({ reload: load });
</script>
