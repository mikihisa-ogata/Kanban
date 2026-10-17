<template>
  <!-- ダッシュボードのガジェット 1 つ。見出しに名前（フィルターを選べる種類はフィルター名も）と、更新・並べ替え・削除の操作。
       フィルターを選べる種類は、見出しの下の選択欄でフィルターを変えられる -->
  <section class="flex flex-col min-w-0 bg-surface border border-line rounded-lg" :aria-labelledby="`gadget-${gadget.ID}-title`">
    <header class="flex items-center gap-1 px-4 pt-3 pb-2">
      <h2 :id="`gadget-${gadget.ID}-title`" class="m-0 min-w-0 flex-1 truncate text-sm font-semibold text-fg">
        {{ definition.label }}
        <span v-if="definition.usesFilter" class="font-normal text-muted">: {{ filterName }}</span>
      </h2>
      <button type="button" :class="iconButtonClass" title="読み直す" aria-label="読み直す" @click="reload">
        <svg class="w-3.5 h-3.5 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M21 12a9 9 0 1 1-3-6.7L21 8" /><path d="M21 3v5h-5" />
        </svg>
      </button>
      <button type="button" :class="iconButtonClass" :disabled="first" title="前へ移動" aria-label="前へ移動" @click="$emit('move', -1)">
        <svg class="w-3.5 h-3.5 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 19V5M5 12l7-7 7 7" /></svg>
      </button>
      <button type="button" :class="iconButtonClass" :disabled="last" title="後ろへ移動" aria-label="後ろへ移動" @click="$emit('move', 1)">
        <svg class="w-3.5 h-3.5 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 5v14M19 12l-7 7-7-7" /></svg>
      </button>
      <button type="button" :class="[iconButtonClass, 'hover:text-danger']" title="ダッシュボードから外す" aria-label="ダッシュボードから外す" @click="$emit('remove')">
        <svg class="w-3.5 h-3.5 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18" /></svg>
      </button>
    </header>

    <div v-if="definition.usesFilter" class="px-4 pb-2">
      <label :for="`gadget-${gadget.ID}-filter`" class="sr-only">フィルター</label>
      <select
        :id="`gadget-${gadget.ID}-filter`"
        :value="gadget.FilterID || 0"
        class="w-full max-w-72 px-2 py-1 border border-line rounded-md bg-surface text-xs text-fg focus:outline-none focus:border-accent"
        @change="$emit('change-filter', Number($event.target.value))"
      >
        <option :value="0">{{ gadget.Type === 'filterResults' ? 'フィルターを選ぶ' : 'すべての課題' }}</option>
        <option v-for="filter in filters" :key="filter.ID" :value="filter.ID">{{ filter.Name }}</option>
      </select>
    </div>

    <div class="flex-1 min-h-0 px-4 pb-4">
      <GadgetIssueList
        v-if="gadget.Type === 'assignedToMe'"
        ref="content"
        :jql="ASSIGNED_TO_ME_JQL"
        :more-link="filterLink(DEFAULT_FILTERS.find(f => f.id === 'my-open'))"
        empty-text="担当している未完了の課題はありません"
      />
      <GadgetActivity v-else-if="gadget.Type === 'activityStream'" ref="content" />
      <template v-else-if="gadget.Type === 'filterResults'">
        <GadgetIssueList v-if="selectedFilter" ref="content" :jql="selectedFilter.JQL" :more-link="filterLink(selectedFilter)" empty-text="フィルターに合う課題はありません" />
        <p v-else class="m-0 text-sm text-muted">
          {{ filters.length ? '表示するフィルターを選んでください。' : '保存したフィルターはまだありません。課題一覧の「名前を付けて保存」から作れます。' }}
        </p>
      </template>
      <template v-else>
        <p v-if="chartLoading" class="m-0 text-sm text-muted">読み込み中...</p>
        <p v-else-if="chartError" class="m-0 text-sm text-danger">{{ chartError }}</p>
        <template v-else-if="gadget.Type === 'statusPie'">
          <StatusPieChart v-if="statusCounts.length" :counts="statusCounts" />
          <p v-else class="m-0 text-sm text-muted">課題はありません</p>
        </template>
        <CreatedResolvedChart v-else-if="gadget.Type === 'createdVsResolved' && trend" :trend="trend" />
      </template>
    </div>
  </section>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import GadgetIssueList from './GadgetIssueList.vue';
import GadgetActivity from './GadgetActivity.vue';
import StatusPieChart from './StatusPieChart.vue';
import CreatedResolvedChart from './CreatedResolvedChart.vue';
import { dashboardApi } from '../api/dashboard';
import { gadgetType, ASSIGNED_TO_ME_JQL, TREND_DAYS } from '../utils/dashboard';
import { DEFAULT_FILTERS, filterLink } from '../utils/filters';

const props = defineProps({
  // { ID, Type, FilterID }
  gadget: {
    type: Object,
    required: true
  },
  // ログイン中のユーザーの保存済みフィルター
  filters: {
    type: Array,
    required: true
  },
  // 並びの先頭・末尾か（移動ボタンを押せなくする）
  first: {
    type: Boolean,
    default: false
  },
  last: {
    type: Boolean,
    default: false
  }
});

// move: 前（-1）・後ろ（1）へ移動、remove: ダッシュボードから外す、change-filter: フィルターを選んだ（0 は未選択）
defineEmits(['move', 'remove', 'change-filter']);

const iconButtonClass = 'shrink-0 w-6 h-6 flex items-center justify-center p-0 border-0 rounded bg-transparent text-muted cursor-pointer hover:bg-raised hover:text-fg disabled:opacity-40 disabled:cursor-not-allowed';

const definition = computed(() => gadgetType(props.gadget.Type));
const selectedFilter = computed(() => props.filters.find(f => f.ID === props.gadget.FilterID) ?? null);
const filterName = computed(() => {
  if (selectedFilter.value) return selectedFilter.value.Name;
  return props.gadget.Type === 'filterResults' ? '未選択' : 'すべての課題';
});
// 円グラフ・作成 vs 解決で絞り込む JQL（ORDER BY は件数に関係ないが、そのまま渡してよい）
const chartJql = computed(() => selectedFilter.value?.JQL ?? '');

const content = ref(null);
const chartLoading = ref(false);
const chartError = ref('');
const statusCounts = ref([]);
const trend = ref(null);

const loadChart = async () => {
  if (props.gadget.Type !== 'statusPie' && props.gadget.Type !== 'createdVsResolved') return;
  chartLoading.value = true;
  chartError.value = '';
  try {
    if (props.gadget.Type === 'statusPie') {
      statusCounts.value = await dashboardApi.fetchStatusCounts(chartJql.value);
    } else {
      trend.value = await dashboardApi.fetchCreatedVsResolved(chartJql.value, TREND_DAYS);
    }
  } catch (err) {
    chartError.value = 'グラフの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch gadget chart:', err);
  } finally {
    chartLoading.value = false;
  }
};
watch(chartJql, loadChart, { immediate: true });

const reload = () => {
  if (content.value && content.value.reload) {
    content.value.reload();
  } else {
    loadChart();
  }
};
</script>
