<template>
  <!-- スペースのタイムライン（Jira のロードマップ）。エピックを日付軸のバーで並べ、ドラッグで期間を変える -->
  <SpaceShell :spaces="spaces" :todos="todos" :current-space="space" @changed="fetchAll">
    <PageToolbar :crumbs="[{ label: 'タイムライン' }]">
      <template v-if="space">
        <div class="inline-flex rounded-md border border-line overflow-hidden" role="radiogroup" aria-label="表示の単位">
          <button
            v-for="option in TIMELINE_SCALES"
            :key="option.value"
            type="button"
            role="radio"
            :aria-checked="scaleValue === option.value"
            :class="[
              'px-2.5 py-1 border-0 text-xs font-medium cursor-pointer whitespace-nowrap',
              scaleValue === option.value ? 'bg-accent/15 text-accent' : 'bg-transparent text-fg hover:bg-raised'
            ]"
            @click="scaleValue = option.value"
          >
            {{ option.label }}
          </button>
        </div>
        <button type="button" :class="toolbarButtonClass()" @click="timeline?.scrollToToday()">今日</button>
      </template>
    </PageToolbar>
    <div class="px-4 md:px-6 py-4">
      <p v-if="loading" class="m-0 text-fg">読み込み中...</p>
      <div v-else-if="!space" class="max-w-3xl bg-surface rounded-2xl px-8 py-6 shadow-xl text-fg">
        スペース「{{ spaceKey }}」が見つかりません。
        <NuxtLink to="/board" class="ml-2 text-accent font-semibold">すべてのスペースのボードを開く</NuxtLink>
      </div>

      <template v-else>
        <p class="m-0 mb-3 text-xs text-muted">
          バーをドラッグして期間を移動、両端をドラッグして期間を変更できます。期間のないエピックは行をクリックすると期間を設定できます。
        </p>
        <EpicTimeline
          ref="timeline"
          :epics="spaceEpics"
          :todos="spaceTodos"
          :sprints="spaceSprints"
          :scale-value="scaleValue"
          @change-span="handleChangeSpan"
          @change-color="handleChangeColor"
          @create="handleCreate"
        />
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
import EpicTimeline from '~/components/EpicTimeline.vue';
import { toolbarButtonClass } from '~/utils/toolbar';
import { todosApi } from '~/api/todos';
import { spacesApi } from '~/api/spaces';
import { epicsApi } from '~/api/epics';
import { sprintsApi } from '~/api/sprints';
import { TIMELINE_SCALES, DEFAULT_TIMELINE_SCALE } from '~/utils/timeline';

const route = useRoute();
const spaceKey = computed(() => String(route.params.key));

const spaces = ref([]);
const todos = ref([]);
const epics = ref([]);
const sprints = ref([]);
const loading = ref(true);
const error = ref('');
const timeline = ref(null);

const space = computed(() => spaces.value.find(s => s.Key === spaceKey.value) || null);
useHead(() => ({ title: space.value ? `タイムライン - ${space.value.Title} - Kanban` : 'Kanban' }));

const spaceEpics = computed(() => (space.value ? epics.value.filter(epic => epic.SpaceID === space.value.ID) : []));
const spaceTodos = computed(() => (space.value ? todos.value.filter(todo => todo.SpaceID === space.value.ID) : []));
const spaceSprints = computed(() => (space.value ? sprints.value.filter(sprint => sprint.SpaceID === space.value.ID) : []));

// 表示の単位（週・月・四半期）はブラウザに覚えておく。使えない環境では毎回「月」
const SCALE_STORAGE_KEY = 'kanban:timelineScale';
const scaleValue = ref(DEFAULT_TIMELINE_SCALE);
watch(scaleValue, (value) => {
  try {
    localStorage.setItem(SCALE_STORAGE_KEY, value);
  } catch {
    // 覚えられなくても動作に支障はない
  }
});

const fetchAll = async () => {
  try {
    error.value = '';
    [spaces.value, todos.value, epics.value, sprints.value] = await Promise.all([
      spacesApi.fetchAll(), todosApi.fetchAll(), epicsApi.fetchAll(), sprintsApi.fetchAll()
    ]);
  } catch (err) {
    error.value = 'タイムラインの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch timeline:', err);
  }
};

// エピックを保存する。画面には先に反映し、失敗したら元に戻す
const saveEpic = async (epic, changes, body) => {
  const index = epics.value.findIndex(e => e.ID === epic.ID);
  if (index === -1) return;
  const before = epics.value[index];
  epics.value.splice(index, 1, { ...before, ...changes });
  try {
    error.value = '';
    await epicsApi.update(epic.ID, { title: before.Title, spaceId: before.SpaceID, ...body });
  } catch (err) {
    const current = epics.value.findIndex(e => e.ID === epic.ID);
    if (current !== -1) epics.value.splice(current, 1, before);
    error.value = 'エピックの保存に失敗しました: ' + err.message;
    console.error('Failed to update epic:', err);
  }
};

const handleChangeSpan = (epic, { start, end }) =>
  saveEpic(epic, { StartDate: start, DueDate: end }, { startDate: start, dueDate: end });

const handleChangeColor = (epic, color) => saveEpic(epic, { Color: color }, { color });

const handleCreate = async (title) => {
  try {
    error.value = '';
    await epicsApi.create({ title, spaceId: space.value.ID });
    epics.value = await epicsApi.fetchAll();
  } catch (err) {
    error.value = 'エピックの作成に失敗しました: ' + err.message;
    console.error('Failed to create epic:', err);
  }
};

onMounted(async () => {
  try {
    const saved = localStorage.getItem(SCALE_STORAGE_KEY);
    if (TIMELINE_SCALES.some(option => option.value === saved)) scaleValue.value = saved;
  } catch {
    // ブラウザ保存が使えなければ既定の単位
  }
  await fetchAll();
  loading.value = false;
});
</script>
