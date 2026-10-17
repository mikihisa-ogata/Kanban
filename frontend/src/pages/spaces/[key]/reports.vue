<template>
  <!-- スペースのレポート。スプリントを選ぶと、変更履歴から再計算したバーンダウンと日ごとの変動を出す -->
  <SpaceShell :spaces="spaces" :todos="todos" :current-space="space" @changed="fetchAll">
    <PageToolbar :crumbs="[{ label: 'レポート' }, { label: 'バーンダウンチャート' }]">
      <template v-if="space && sprintOptions.length">
        <label for="report-sprint" class="text-xs text-muted">スプリント</label>
        <select
          id="report-sprint"
          :value="selectedId"
          class="max-w-64 px-2 py-1 border border-line rounded-md bg-canvas text-sm font-medium text-fg cursor-pointer focus:outline-none focus:border-accent"
          @change="selectSprint(Number($event.target.value))"
        >
          <option v-for="sprint in sprintOptions" :key="sprint.ID" :value="sprint.ID">
            {{ sprint.Name }}{{ sprint.State === SPRINT_ACTIVE ? '（開始中）' : '' }}
          </option>
        </select>
      </template>
    </PageToolbar>
    <div class="px-4 md:px-6 py-4">
      <p v-if="loading" class="m-0 text-fg">読み込み中...</p>
      <div v-else-if="!space" class="max-w-3xl bg-surface rounded-2xl px-8 py-6 shadow-xl text-fg">
        スペース「{{ spaceKey }}」が見つかりません。
        <NuxtLink to="/board" class="ml-2 text-accent font-semibold">すべてのスペースのボードを開く</NuxtLink>
      </div>
      <div v-else-if="!sprintOptions.length" class="max-w-3xl bg-surface border border-line rounded-xl px-6 py-5 text-sm text-fg">
        開始したスプリントがありません。バーンダウンチャートは、スプリントを開始すると表示できます。
        <NuxtLink :to="backlogPath(space)" class="ml-1 text-accent font-semibold">バックログを開く</NuxtLink>
      </div>

      <template v-else-if="selectedSprint">
        <section class="max-w-5xl flex flex-col gap-4">
          <header class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
            <h1 class="m-0 text-lg font-semibold text-fg">{{ selectedSprint.Name }}</h1>
            <span class="text-sm text-muted">{{ sprintPeriodLabel(selectedSprint) }}</span>
            <span
              :class="[
                'px-1.5 py-0.5 rounded text-[11px] font-bold',
                selectedSprint.State === SPRINT_ACTIVE ? 'bg-accent/15 text-accent' : 'bg-success/15 text-success'
              ]"
            >
              {{ selectedSprint.State === SPRINT_ACTIVE ? '開始中' : '完了' }}
            </span>
            <span v-if="selectedSprint.State === SPRINT_ACTIVE" class="text-xs text-muted">{{ remainingDaysLabel(selectedSprint) }}</span>
            <p v-if="selectedSprint.Goal" class="basis-full m-0 text-sm text-muted">ゴール: {{ selectedSprint.Goal }}</p>
          </header>

          <p v-if="burndownLoading && !burndown" class="m-0 text-sm text-muted">集計中...</p>
          <template v-if="burndown">
            <dl class="grid grid-cols-2 sm:grid-cols-4 gap-3 m-0">
              <div v-for="stat in stats" :key="stat.label" class="bg-surface border border-line rounded-lg px-4 py-3">
                <dt class="text-xs text-muted">{{ stat.label }}</dt>
                <dd class="m-0 mt-1 text-xl font-semibold text-fg">{{ stat.value }}</dd>
              </div>
            </dl>

            <div class="bg-surface border border-line rounded-xl p-4">
              <BurndownChart :burndown="burndown" :active="selectedSprint.State === SPRINT_ACTIVE" />
            </div>

            <div class="bg-surface border border-line rounded-xl overflow-x-auto">
              <table class="w-full min-w-[640px] border-collapse text-sm text-fg">
                <caption class="px-4 pt-3 pb-2 text-left text-sm font-semibold text-fg">日ごとの変動（ストーリーポイント）</caption>
                <thead>
                  <tr class="border-b border-line text-xs text-muted whitespace-nowrap">
                    <th scope="col" class="px-4 py-2 text-left font-medium">日付</th>
                    <th scope="col" class="px-3 py-2 text-right font-medium">追加</th>
                    <th scope="col" class="px-3 py-2 text-right font-medium">除外</th>
                    <th scope="col" class="px-3 py-2 text-right font-medium">見積もり変更</th>
                    <th scope="col" class="px-3 py-2 text-right font-medium">完了</th>
                    <th scope="col" class="px-3 py-2 text-right font-medium">残り</th>
                    <th scope="col" class="px-4 py-2 text-left font-medium">課題</th>
                  </tr>
                </thead>
                <tbody>
                  <tr class="border-b border-line">
                    <th scope="row" class="px-4 py-2 text-left font-medium whitespace-nowrap">スプリント開始</th>
                    <td colspan="4" class="px-3 py-2 text-right text-xs text-muted whitespace-nowrap">{{ formatDateTime(burndown.Start) }}</td>
                    <td class="px-3 py-2 text-right font-semibold">{{ formatStoryPoints(burndown.StartPoints) }}</td>
                    <td class="px-4 py-2" />
                  </tr>
                  <tr v-for="row in rows" :key="row.Date" :class="['border-b border-line last:border-b-0 align-top', row.Remaining === null ? 'text-muted' : '']">
                    <th scope="row" class="px-4 py-2 text-left font-medium whitespace-nowrap">{{ formatDay(row.Date) }}</th>
                    <td class="px-3 py-2 text-right tabular-nums">{{ formatChange(row.Added) }}</td>
                    <td class="px-3 py-2 text-right tabular-nums">{{ formatChange(row.Removed) }}</td>
                    <td class="px-3 py-2 text-right tabular-nums">{{ formatChange(row.Estimate) }}</td>
                    <td class="px-3 py-2 text-right tabular-nums">{{ formatChange(row.Completed) }}</td>
                    <td class="px-3 py-2 text-right tabular-nums font-semibold">{{ row.Remaining === null ? '—' : formatStoryPoints(row.Remaining) }}</td>
                    <td class="px-4 py-2">
                      <ul v-if="row.events.length" class="flex flex-wrap gap-1.5 m-0 p-0 list-none">
                        <li v-for="(event, index) in row.events" :key="index">
                          <NuxtLink
                            :to="issuePath(event.Key)"
                            :title="`${event.Title}（${BURNDOWN_EVENT_LABELS[event.Kind] ?? event.Kind}）`"
                            class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-raised text-xs text-fg no-underline hover:text-accent"
                          >
                            <span class="font-mono">{{ event.Key }}</span>
                            <span class="text-muted">{{ BURNDOWN_EVENT_LABELS[event.Kind] ?? event.Kind }}</span>
                            <span v-if="formatChange(event.Change)" class="font-semibold">{{ formatChange(event.Change) }}</span>
                          </NuxtLink>
                        </li>
                      </ul>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p class="m-0 text-xs text-muted">
              残りはスプリントに入っている未完了の課題（サブタスクを除く）のストーリーポイントの合計です。課題の変更履歴から、追加・除外・完了・ポイント変更を再計算しています。
            </p>
          </template>
        </section>
      </template>
    </div>

    <div v-if="error" class="fixed bottom-8 right-8 bg-danger/10 text-danger px-6 py-4 rounded-xl shadow-xl max-w-96 text-sm font-semibold border-l-4 border-danger animate-slideIn">
      {{ error }}
    </div>
  </SpaceShell>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue';
import SpaceShell from '~/components/SpaceShell.vue';
import PageToolbar from '~/components/PageToolbar.vue';
import BurndownChart from '~/components/BurndownChart.vue';
import { todosApi } from '~/api/todos';
import { spacesApi } from '~/api/spaces';
import { sprintsApi } from '~/api/sprints';
import { SPRINT_ACTIVE, sprintPeriodLabel, remainingDaysLabel } from '~/utils/sprints';
import { reportableSprints, burndownRows, formatChange, BURNDOWN_EVENT_LABELS } from '~/utils/burndown';
import { formatStoryPoints } from '~/utils/storyPoints';
import { backlogPath, issuePath, reportsPath } from '~/utils/spaceRoutes';

const route = useRoute();
const router = useRouter();
const spaceKey = computed(() => String(route.params.key));

const spaces = ref([]);
const todos = ref([]);
const sprints = ref([]);
const loading = ref(true);
const error = ref('');
const burndown = ref(null);
const burndownLoading = ref(false);

const space = computed(() => spaces.value.find(s => s.Key === spaceKey.value) || null);
useHead(() => ({ title: space.value ? `レポート - ${space.value.Title} - Kanban` : 'Kanban' }));

const sprintOptions = computed(() =>
  (space.value ? reportableSprints(sprints.value.filter(sprint => sprint.SpaceID === space.value.ID)) : []));

// URL の ?sprint= のスプリント。なければ開始中か、最後に完了したスプリント
const selectedId = computed(() => {
  const requested = Number(route.query.sprint);
  if (sprintOptions.value.some(sprint => sprint.ID === requested)) return requested;
  return sprintOptions.value[0]?.ID ?? null;
});
const selectedSprint = computed(() => sprintOptions.value.find(sprint => sprint.ID === selectedId.value) ?? null);

const selectSprint = (id) => router.replace(reportsPath(space.value, id));

const rows = computed(() => (burndown.value ? burndownRows(burndown.value) : []));

const stats = computed(() => {
  const b = burndown.value;
  const sum = (kind) => b.Events.filter(event => event.Kind === kind).reduce((total, event) => total + event.Change, 0);
  const remaining = b.Events.length ? b.Events[b.Events.length - 1].Remaining : b.StartPoints;
  const added = sum('added') + sum('estimate');
  return [
    { label: '開始時のポイント', value: formatStoryPoints(b.StartPoints) },
    { label: '完了したポイント', value: formatStoryPoints(-(sum('completed') + sum('reopened'))) },
    { label: '範囲の変更', value: formatChange(added + sum('removed')) || '0' },
    { label: '残りのポイント', value: formatStoryPoints(remaining) }
  ];
});

const formatDay = (text) => {
  const [y, m, d] = text.split('-').map(Number);
  const date = new Date(y, m - 1, d);
  return `${m}/${d}（${'日月火水木金土'[date.getDay()]}）`;
};

const formatDateTime = (value) => {
  const date = new Date(value);
  const pad = (n) => String(n).padStart(2, '0');
  return `${date.getMonth() + 1}/${date.getDate()} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
};

// 選んだスプリントのバーンダウンを読む。切り替えが速いときは最後に選んだものだけを出す
let request = 0;
const loadBurndown = async () => {
  const id = selectedId.value;
  const current = ++request;
  if (!id) {
    burndown.value = null;
    return;
  }
  burndownLoading.value = true;
  try {
    error.value = '';
    const result = await sprintsApi.burndown(id);
    if (current === request) burndown.value = result;
  } catch (err) {
    if (current === request) {
      burndown.value = null;
      error.value = 'バーンダウンの取得に失敗しました: ' + err.message;
    }
    console.error('Failed to fetch burndown:', err);
  } finally {
    if (current === request) burndownLoading.value = false;
  }
};
watch(selectedId, () => {
  burndown.value = null;
  loadBurndown();
});

const fetchAll = async () => {
  try {
    error.value = '';
    [spaces.value, todos.value, sprints.value] = await Promise.all([spacesApi.fetchAll(), todosApi.fetchAll(), sprintsApi.fetchAll()]);
  } catch (err) {
    error.value = 'レポートの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch reports:', err);
  }
};

onMounted(async () => {
  // スプリントが決まると watch で読む
  await fetchAll();
  loading.value = false;
});
</script>
