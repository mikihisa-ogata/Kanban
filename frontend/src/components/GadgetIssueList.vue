<template>
  <!-- ダッシュボードのガジェットの課題一覧（自分の担当課題・保存済みフィルターの結果）。JQL に合う課題の先頭の数件と、課題一覧へのリンク -->
  <div>
    <p v-if="loading" class="m-0 text-sm text-muted">読み込み中...</p>
    <p v-else-if="error" class="m-0 text-sm text-danger">{{ error }}</p>
    <template v-else>
      <ul v-if="result.Issues.length" class="m-0 p-0 list-none flex flex-col">
        <li v-for="issue in result.Issues" :key="issue.ID" class="flex items-center gap-2 py-1.5 border-b border-line last:border-b-0 text-sm">
          <IssueTypeIcon :type="issue.IssueType" :size="14" class="shrink-0" />
          <NuxtLink :to="issuePath(issue.Key)" class="shrink-0 font-mono text-xs text-muted no-underline hover:text-accent hover:underline">{{ issue.Key }}</NuxtLink>
          <NuxtLink :to="issuePath(issue.Key)" class="min-w-0 flex-1 truncate text-fg no-underline hover:text-accent" :title="issue.Title">{{ issue.Title }}</NuxtLink>
          <PriorityIcon :priority="issue.Priority" :size="14" class="shrink-0" />
          <span :class="['shrink-0 max-w-28 truncate px-1.5 py-0.5 rounded text-[11px] font-semibold', STATUS_CATEGORY_BADGE_CLASS[issue.StatusCategory] || 'bg-raised text-fg']">
            {{ issue.StatusName }}
          </span>
        </li>
      </ul>
      <p v-else class="m-0 text-sm text-muted">{{ emptyText }}</p>
      <NuxtLink v-if="result.Total > 0" :to="moreLink" class="inline-block mt-2 text-xs font-semibold text-accent no-underline hover:underline">
        {{ result.Total > result.Issues.length ? `全 ${result.Total} 件を課題一覧で見る` : '課題一覧で開く' }}
      </NuxtLink>
    </template>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue';
import IssueTypeIcon from './IssueTypeIcon.vue';
import PriorityIcon from './PriorityIcon.vue';
import { todosApi } from '../api/todos';
import { issuePath } from '../utils/spaceRoutes';
import { STATUS_CATEGORY_BADGE_CLASS } from '../utils/statuses';

const props = defineProps({
  jql: {
    type: String,
    required: true
  },
  // 課題一覧で開くリンク（フィルターのリンクなど）
  moreLink: {
    type: [String, Object],
    required: true
  },
  emptyText: {
    type: String,
    default: '課題はありません'
  }
});

// ガジェットに出す件数
const LIMIT = 10;

const loading = ref(true);
const error = ref('');
const result = ref({ Issues: [], Total: 0 });

const load = async () => {
  loading.value = true;
  error.value = '';
  try {
    result.value = await todosApi.searchJql(props.jql, 1, LIMIT);
  } catch (err) {
    error.value = '課題の取得に失敗しました: ' + err.message;
    console.error('Failed to fetch gadget issues:', err);
  } finally {
    loading.value = false;
  }
};
watch(() => props.jql, load, { immediate: true });

defineExpose({ reload: load });
</script>
