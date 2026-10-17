<template>
  <!-- ダッシュボード（Jira のダッシュボード）。ガジェットを 2 列に並べる。ガジェットの追加・削除・並べ替え・フィルターの選択は、
       変えるたびにユーザーごとに保存する（リロードしても残る） -->
  <SpaceShell :spaces="spaces" :current-space="null" @changed="fetchSpaces">
    <PageToolbar :crumbs="[{ label: 'ダッシュボード' }]">
      <template #chips>
        <span v-if="saving" :class="toolbarChipClass()">保存中...</span>
      </template>
      <ToolbarMenu label="ガジェットを追加">
        <template #default="{ close }">
          <ul class="w-72 m-0 p-0 list-none flex flex-col gap-0.5" aria-label="追加するガジェット">
            <li v-for="type in GADGET_TYPES" :key="type.type">
              <button
                type="button"
                :disabled="gadgets.length >= DASHBOARD_GADGET_MAX"
                class="w-full flex flex-col items-start gap-0.5 px-2 py-1.5 border-0 rounded-md bg-transparent text-left cursor-pointer hover:bg-raised disabled:opacity-50 disabled:cursor-not-allowed"
                @click="addGadget(type.type); close()"
              >
                <span class="text-sm font-semibold text-fg">{{ type.label }}</span>
                <span class="text-xs text-muted">{{ type.description }}</span>
              </button>
            </li>
          </ul>
          <p v-if="gadgets.length >= DASHBOARD_GADGET_MAX" class="mt-2 mb-0 text-xs text-muted">ガジェットは {{ DASHBOARD_GADGET_MAX }} 個まで置けます</p>
        </template>
      </ToolbarMenu>
    </PageToolbar>

    <main class="px-4 md:px-6 py-5">
      <p v-if="error" class="mt-0 mb-4 px-3 py-2 rounded-md bg-danger/10 text-sm text-danger" role="alert">{{ error }}</p>
      <p v-if="loading" class="m-0 text-sm text-muted">読み込み中...</p>
      <div v-else-if="gadgets.length" class="grid grid-cols-1 xl:grid-cols-2 gap-4 items-start">
        <DashboardGadget
          v-for="(gadget, index) in gadgets"
          :key="gadget.ID"
          :gadget="gadget"
          :filters="savedFilters"
          :first="index === 0"
          :last="index === gadgets.length - 1"
          @move="moveTo(index, $event)"
          @remove="removeGadget(index)"
          @change-filter="changeFilter(index, $event)"
        />
      </div>
      <div v-else class="max-w-md mx-auto mt-16 text-center">
        <p class="m-0 text-sm text-fg font-semibold">ダッシュボードにガジェットがありません</p>
        <p class="mt-1 mb-0 text-sm text-muted">右上の「ガジェットを追加」から、担当課題やグラフを並べられます。</p>
      </div>
    </main>
  </SpaceShell>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import SpaceShell from '~/components/SpaceShell.vue';
import PageToolbar from '~/components/PageToolbar.vue';
import ToolbarMenu from '~/components/ToolbarMenu.vue';
import DashboardGadget from '~/components/DashboardGadget.vue';
import { spacesApi } from '~/api/spaces';
import { dashboardApi } from '~/api/dashboard';
import { useSavedFilters } from '~/composables/useSavedFilters';
import { GADGET_TYPES, DASHBOARD_GADGET_MAX, newGadgetId, moveGadget } from '~/utils/dashboard';
import { toolbarChipClass } from '~/utils/toolbar';

useHead({ title: 'ダッシュボード - Kanban' });

const spaces = ref([]);
const gadgets = ref([]);
const loading = ref(true);
const saving = ref(false);
const error = ref('');
const { filters: savedFilters, reload: reloadFilters } = useSavedFilters();

const fetchSpaces = async () => {
  try {
    spaces.value = await spacesApi.fetchAll();
  } catch (err) {
    console.error('Failed to fetch spaces:', err);
  }
};

// ガジェットの並びを変えたら、すぐに保存する。保存した結果（消えたフィルターは未選択になる）で置き換え、
// 失敗したら変える前の並びに戻す。続けて変えたときは、最後の保存の結果だけを使う
let saveRequest = 0;
const update = async (next) => {
  const previous = gadgets.value;
  gadgets.value = next;
  const request = ++saveRequest;
  saving.value = true;
  error.value = '';
  try {
    const saved = await dashboardApi.save(next);
    if (request === saveRequest) gadgets.value = saved;
  } catch (err) {
    if (request === saveRequest) {
      gadgets.value = previous;
      error.value = 'ダッシュボードの保存に失敗しました: ' + err.message;
    }
    console.error('Failed to save dashboard:', err);
  } finally {
    if (request === saveRequest) saving.value = false;
  }
};

// 追加したガジェットは先頭に置く（すぐ目に入るように）
const addGadget = (type) => {
  if (gadgets.value.length >= DASHBOARD_GADGET_MAX) return;
  update([{ ID: newGadgetId(gadgets.value), Type: type, FilterID: 0 }, ...gadgets.value]);
};
const removeGadget = (index) => update(gadgets.value.filter((_, i) => i !== index));
const moveTo = (index, offset) => update(moveGadget(gadgets.value, index, offset));
const changeFilter = (index, filterId) =>
  update(gadgets.value.map((g, i) => (i === index ? { ...g, FilterID: filterId } : g)));

// フィルターを読んでからガジェットを出す（フィルターを選んだガジェットが、すべての課題で一度読まれないように）
onMounted(async () => {
  fetchSpaces();
  try {
    const [loaded] = await Promise.all([dashboardApi.fetch(), reloadFilters()]);
    gadgets.value = loaded;
  } catch (err) {
    error.value = 'ダッシュボードの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch dashboard:', err);
  } finally {
    loading.value = false;
  }
});
</script>
