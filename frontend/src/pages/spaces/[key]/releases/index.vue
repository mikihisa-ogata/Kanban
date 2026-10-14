<template>
  <!-- スペースのリリース（Jira のリリース）。バージョンの一覧と作成 -->
  <SpaceShell :spaces="spaces" :todos="todos" :current-space="space" @changed="fetchAll">
    <PageToolbar :crumbs="[{ label: 'リリース' }]">
      <template v-if="space">
        <label :class="[toolbarButtonClass(showArchived), 'select-none']">
          <input v-model="showArchived" type="checkbox" class="cursor-pointer" />
          アーカイブも表示
        </label>
        <button type="button" :class="toolbarButtonClass()" @click="creating = true">バージョンを作成</button>
      </template>
    </PageToolbar>
    <div class="px-8 py-6">
      <p v-if="loading" class="m-0 text-fg">読み込み中...</p>
      <div v-else-if="!space" class="max-w-3xl bg-white rounded-2xl px-8 py-6 shadow-xl text-gray-700">
        スペース「{{ spaceKey }}」が見つかりません。
        <NuxtLink to="/board" class="ml-2 text-blue-600 font-semibold">すべてのスペースのボードを開く</NuxtLink>
      </div>

      <template v-else>
        <!-- バージョンの作成 -->
        <form v-if="creating" class="mb-4 max-w-6xl bg-white rounded-2xl px-6 py-5 shadow-xl flex flex-col gap-3" @submit.prevent="handleCreate">
          <h3 class="m-0 text-base font-bold text-gray-900">バージョンを作成</h3>
          <VersionFields v-model="form" id-prefix="new-version" />
          <p v-if="formError" class="m-0 text-sm text-red-600">{{ formError }}</p>
          <div class="flex justify-end gap-2">
            <button type="button" class="px-3 py-1.5 bg-transparent border-0 rounded-md text-sm font-semibold text-gray-700 cursor-pointer hover:bg-gray-100" @click="closeCreate">キャンセル</button>
            <button
              type="submit"
              :disabled="!!formError || saving"
              class="px-4 py-1.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              作成
            </button>
          </div>
        </form>

        <section class="max-w-6xl bg-white rounded-2xl shadow-xl overflow-hidden">
          <table class="w-full border-collapse text-sm">
            <thead class="bg-gray-50 text-left text-xs font-semibold text-gray-600">
              <tr>
                <th class="px-5 py-2.5">バージョン</th>
                <th class="px-3 py-2.5">状態</th>
                <th class="px-3 py-2.5 w-56">進捗</th>
                <th class="px-3 py-2.5">開始日</th>
                <th class="px-3 py-2.5">リリース日</th>
                <th class="px-5 py-2.5">説明</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="version in visibleVersions" :key="version.ID" class="border-t border-gray-100 hover:bg-gray-50">
                <td class="px-5 py-2.5 font-semibold">
                  <NuxtLink :to="versionPath(space, version.ID)" class="text-blue-700 no-underline hover:underline">{{ version.Name }}</NuxtLink>
                </td>
                <td class="px-3 py-2.5">
                  <span :class="['px-2 py-0.5 rounded text-xs font-semibold whitespace-nowrap', VERSION_STATUS_BADGE_CLASS[version.Status]]">
                    {{ VERSION_STATUS_LABEL[version.Status] }}
                  </span>
                </td>
                <td class="px-3 py-2.5">
                  <VersionProgress :progress="versionProgress(issuesOfVersion(spaceTodos, version.ID))" />
                </td>
                <td class="px-3 py-2.5 text-gray-700 whitespace-nowrap">{{ version.StartDate || '-' }}</td>
                <td :class="['px-3 py-2.5 whitespace-nowrap', isOverdue(version) ? 'text-red-600 font-semibold' : 'text-gray-700']">
                  {{ version.ReleaseDate || '-' }}
                </td>
                <td class="px-5 py-2.5 text-gray-600 max-w-80 truncate" :title="version.Description">{{ version.Description }}</td>
              </tr>
              <tr v-if="!visibleVersions.length">
                <td colspan="6" class="px-5 py-6 text-center text-gray-500">バージョンはまだありません。「バージョンを作成」から作れます。</td>
              </tr>
            </tbody>
          </table>
        </section>
      </template>
    </div>

    <div v-if="error" class="fixed bottom-8 right-8 bg-red-100 text-red-800 px-6 py-4 rounded-xl shadow-xl max-w-96 text-sm font-semibold border-l-4 border-red-500 animate-slideIn">
      {{ error }}
    </div>
  </SpaceShell>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import SpaceShell from '~/components/SpaceShell.vue';
import PageToolbar from '~/components/PageToolbar.vue';
import { toolbarButtonClass } from '~/utils/toolbar';
import VersionFields from '~/components/VersionFields.vue';
import VersionProgress from '~/components/VersionProgress.vue';
import { todosApi } from '~/api/todos';
import { spacesApi } from '~/api/spaces';
import { versionsApi } from '~/api/versions';
import { versionPath } from '~/utils/spaceRoutes';
import { dateAfter } from '~/utils/sprints';
import {
  VERSION_UNRELEASED, VERSION_ARCHIVED, VERSION_STATUS_LABEL, VERSION_STATUS_BADGE_CLASS, versionProgress, issuesOfVersion
} from '~/utils/versions';
import { versionFieldsError, emptyVersionFields } from '~/utils/versionFields';

const route = useRoute();
const spaceKey = computed(() => String(route.params.key));

const spaces = ref([]);
const todos = ref([]);
const versions = ref([]);
const loading = ref(true);
const error = ref('');

const space = computed(() => spaces.value.find(s => s.Key === spaceKey.value) || null);
useHead(() => ({ title: space.value ? `リリース - ${space.value.Title} - Kanban` : 'Kanban' }));

const spaceTodos = computed(() => (space.value ? todos.value.filter(todo => todo.SpaceID === space.value.ID) : []));

// アーカイブしたバージョンは、チェックを入れたときだけ出す（Jira の既定と同じ）
const showArchived = ref(false);
const visibleVersions = computed(() =>
  space.value
    ? versions.value.filter(v => v.SpaceID === space.value.ID && (showArchived.value || v.Status !== VERSION_ARCHIVED))
    : []
);

// 未リリースでリリース日を過ぎている
const isOverdue = (version) => version.Status === VERSION_UNRELEASED && !!version.ReleaseDate && version.ReleaseDate < dateAfter(0);

const fetchAll = async () => {
  try {
    error.value = '';
    [spaces.value, todos.value, versions.value] = await Promise.all([spacesApi.fetchAll(), todosApi.fetchAll(), versionsApi.fetchAll()]);
  } catch (err) {
    error.value = 'リリースの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch releases:', err);
  }
};

// 作成
const creating = ref(false);
const form = ref(emptyVersionFields());
const saving = ref(false);
const formError = computed(() => versionFieldsError(form.value));

const closeCreate = () => {
  creating.value = false;
  form.value = emptyVersionFields();
};

const handleCreate = async () => {
  if (formError.value) return;
  saving.value = true;
  try {
    error.value = '';
    await versionsApi.create(space.value.ID, form.value);
    closeCreate();
    versions.value = await versionsApi.fetchAll();
  } catch (err) {
    error.value = 'バージョンの作成に失敗しました: ' + err.message;
    console.error('Failed to create version:', err);
  } finally {
    saving.value = false;
  }
};

onMounted(async () => {
  await fetchAll();
  loading.value = false;
});
</script>
