<template>
  <!-- 課題一覧（Jira の課題ナビゲーター）。全スペースを横断して絞り込み・並べ替えし、選んだ課題を右に出す -->
  <SpaceShell :spaces="spaces" :current-space="currentSpace" @changed="handleSpacesChanged">
    <PageToolbar :crumbs="crumbs">
      <template #chips>
        <span v-if="filterModified" :class="toolbarChipClass()" title="開いたフィルターから条件を変えています">変更あり</span>
      </template>

      <!-- フィルター（既定のフィルターと、自分が保存したフィルター）を開く -->
      <ToolbarMenu label="フィルター">
        <template #default="{ close }">
          <div class="w-72 flex flex-col gap-3 text-sm">
            <section>
              <h3 class="m-0 mb-1 text-xs font-semibold text-muted">既定のフィルター</h3>
              <ul class="m-0 p-0 list-none flex flex-col gap-0.5">
                <li v-for="filter in DEFAULT_FILTERS" :key="filter.id">
                  <NuxtLink :to="filterLink(filter)" :class="filterItemClass(filter.id)" :title="filter.jql" @click="close">
                    {{ filter.name }}
                  </NuxtLink>
                </li>
              </ul>
            </section>
            <section>
              <h3 class="m-0 mb-1 text-xs font-semibold text-muted">自分のフィルター</h3>
              <ul v-if="savedFilters.length" class="m-0 p-0 list-none flex flex-col gap-0.5 max-h-64 overflow-y-auto">
                <li v-for="filter in savedFilters" :key="filter.ID" class="flex items-center gap-1">
                  <button
                    type="button"
                    :class="starButtonClass(filter.Favourite)"
                    :title="filter.Favourite ? 'お気に入りから外す' : 'お気に入りにする（左ナビに出す）'"
                    :aria-pressed="filter.Favourite"
                    @click="toggleFavourite(filter)"
                  >
                    <svg class="w-4 h-4 block" viewBox="0 0 24 24" :fill="filter.Favourite ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2" stroke-linejoin="round" aria-hidden="true">
                      <path :d="STAR_ICON" />
                    </svg>
                  </button>
                  <NuxtLink :to="filterLink(filter)" :class="['flex-1 min-w-0 truncate', filterItemClass(String(filter.ID))]" :title="filter.JQL || 'すべての課題'" @click="close">
                    {{ filter.Name }}
                  </NuxtLink>
                </li>
              </ul>
              <p v-else class="m-0 px-2 text-xs text-muted">保存したフィルターはまだありません。条件を決めて「名前を付けて保存」から保存できます。</p>
            </section>
          </div>
        </template>
      </ToolbarMenu>

      <!-- 開いている保存済みフィルターの操作 -->
      <template v-if="currentFilter && currentFilter.saved">
        <button
          type="button"
          :class="[toolbarButtonClass(), currentFilter.favourite ? 'text-warning' : '']"
          :title="currentFilter.favourite ? 'お気に入りから外す' : 'お気に入りにする（左ナビに出す）'"
          :aria-pressed="currentFilter.favourite"
          @click="toggleFavourite(currentSavedFilter)"
        >
          <svg class="w-3.5 h-3.5 block" viewBox="0 0 24 24" :fill="currentFilter.favourite ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2" stroke-linejoin="round" aria-hidden="true">
            <path :d="STAR_ICON" />
          </svg>
          お気に入り
        </button>
        <button v-if="filterModified" type="button" :class="toolbarButtonClass(true)" title="開いているフィルターを今の条件で上書きする" @click="overwriteFilter">
          保存
        </button>
      </template>
      <ToolbarMenu label="名前を付けて保存">
        <template #default="{ close }">
          <form class="w-72 flex flex-col gap-2 text-sm" @submit.prevent="saveAsFilter(close)">
            <label for="filter-name" class="text-xs font-semibold text-muted">フィルターの名前</label>
            <input
              id="filter-name"
              v-model="newFilterName"
              type="text"
              maxlength="100"
              placeholder="例: 今週の自分のバグ"
              class="w-full px-2 py-1 border border-line rounded-md text-sm text-fg bg-surface focus:outline-none focus:border-accent"
            />
            <label class="flex items-center gap-2 text-xs text-fg">
              <input v-model="newFilterFavourite" type="checkbox" />
              お気に入りにする（左ナビに出す）
            </label>
            <p class="m-0 text-xs text-muted break-all">条件: <code class="font-mono">{{ currentJql || '（すべての課題）' }}</code></p>
            <p v-if="filterFormError" class="m-0 text-xs text-danger">{{ filterFormError }}</p>
            <button
              type="submit"
              :disabled="!newFilterName.trim() || filterSaving"
              class="self-end px-2.5 py-1 border-0 rounded-md bg-accent text-on-accent text-xs font-semibold cursor-pointer hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed"
            >
              保存
            </button>
          </form>
        </template>
      </ToolbarMenu>
      <ToolbarMenu v-if="currentFilter && currentFilter.saved" label="編集">
        <template #default="{ close }">
          <form class="w-72 flex flex-col gap-2 text-sm" @submit.prevent="renameFilter(close)">
            <label for="filter-rename" class="text-xs font-semibold text-muted">フィルターの名前</label>
            <input
              id="filter-rename"
              v-model="renameDraft"
              type="text"
              maxlength="100"
              class="w-full px-2 py-1 border border-line rounded-md text-sm text-fg bg-surface focus:outline-none focus:border-accent"
            />
            <p v-if="filterFormError" class="m-0 text-xs text-danger">{{ filterFormError }}</p>
            <div class="flex items-center justify-between gap-2">
              <button
                type="button"
                class="px-2.5 py-1 border-0 rounded-md bg-transparent text-xs font-semibold text-danger cursor-pointer hover:bg-danger/10"
                @click="close(); deletingFilter = true"
              >
                フィルターを削除
              </button>
              <button
                type="submit"
                :disabled="!renameDraft.trim() || filterSaving"
                class="px-2.5 py-1 border-0 rounded-md bg-accent text-on-accent text-xs font-semibold cursor-pointer hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed"
              >
                名前を変更
              </button>
            </div>
          </form>
        </template>
      </ToolbarMenu>

      <!-- 基本（フィルターで選ぶ）と JQL（クエリを書く）の切り替え -->
      <div class="flex p-0.5 rounded-md bg-raised" role="group" aria-label="検索の方法">
        <button type="button" :class="modeButton(!jqlMode)" :aria-pressed="!jqlMode" @click="switchToBasic">基本</button>
        <button type="button" :class="modeButton(jqlMode)" :aria-pressed="jqlMode" @click="switchToJql">JQL</button>
      </div>
    </PageToolbar>
    <div class="px-8 py-6">

      <!-- 詳細検索（JQL） -->
      <div v-if="jqlMode" class="mb-4 flex flex-wrap items-start gap-2">
        <JqlInput v-model="jqlDraft" :masters="jqlMasters" :error="jqlError" @submit="submitJql" />
        <FilterMenu label="表示列" :options="columnOptions" :model-value="columns" @update:model-value="changeColumns" />
      </div>

      <!-- フィルター -->
      <div v-else class="mb-4 flex flex-wrap items-center gap-2">
        <input
          :value="textDraft"
          type="search"
          placeholder="要約・説明・キーを検索"
          aria-label="テキストで検索"
          class="w-60 px-3 py-1.5 border-0 rounded-md text-sm text-fg bg-surface focus:outline-none focus:ring-2 focus:ring-accent/40"
          @input="handleTextInput"
        />
        <FilterMenu label="スペース" :options="spaceOptions" :model-value="search.spaces" @update:model-value="update({ spaces: $event })" />
        <FilterMenu label="タイプ" :options="typeOptions" :model-value="search.types" @update:model-value="update({ types: $event })" />
        <FilterMenu
          label="ステータス"
          :options="statusOptions"
          :model-value="search.statuses"
          @update:model-value="update({ statuses: $event })"
        />
        <FilterMenu
          label="担当者"
          :options="assigneeOptions"
          :model-value="search.assignees"
          @update:model-value="update({ assignees: $event })"
        />
        <FilterMenu
          label="ラベル"
          :options="labelOptions"
          :model-value="search.labels"
          empty-text="ラベルの付いた課題がありません"
          @update:model-value="update({ labels: $event })"
        />
        <button
          v-if="searching"
          type="button"
          class="px-3 py-1.5 border-0 rounded-md bg-transparent text-sm font-semibold text-fg underline-offset-2 cursor-pointer hover:underline"
          @click="clearSearch"
        >
          フィルターをクリア
        </button>
        <div class="ml-auto">
          <FilterMenu label="表示列" :options="columnOptions" :model-value="columns" @update:model-value="changeColumns" />
        </div>
      </div>

      <div :class="['flex gap-4 items-start', detailKey ? 'flex-col xl:flex-row' : '']">
        <!-- 一覧 -->
        <section :class="['min-w-0 bg-surface rounded-2xl shadow-xl overflow-hidden', detailKey ? 'w-full xl:w-1/2' : 'w-full']">
          <div class="overflow-x-auto">
            <table class="w-full border-collapse text-sm">
              <thead>
                <tr class="bg-raised text-left text-xs text-muted">
                  <th v-for="column in visibleColumns" :key="column.id" scope="col" class="px-3 py-2 font-semibold whitespace-nowrap" :aria-sort="ariaSort(column)">
                    <button
                      type="button"
                      class="flex items-center gap-1 p-0 bg-transparent border-0 text-xs font-semibold text-muted cursor-pointer hover:text-fg"
                      :title="`${column.label}で並べ替え`"
                      @click="sortBy(column)"
                    >
                      {{ column.label }}
                      <span v-if="currentSort.sort === column.sort" class="text-accent" aria-hidden="true">{{ currentSort.order === 'asc' ? '▲' : '▼' }}</span>
                    </button>
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="issue in result.Issues"
                  :key="issue.ID"
                  :class="['border-t border-line cursor-pointer', issue.Key === detailKey ? 'bg-accent/10' : 'hover:bg-raised']"
                  :aria-selected="issue.Key === detailKey"
                  @click="openDetail(issue.Key)"
                >
                  <td v-for="column in visibleColumns" :key="column.id" class="px-3 py-2 align-middle">
                    <IssueTypeIcon v-if="column.id === 'type'" :type="issue.IssueType" />
                    <a
                      v-else-if="column.id === 'key'"
                      :href="issuePath(issue.Key)"
                      class="font-mono text-xs font-semibold text-muted whitespace-nowrap no-underline hover:text-accent hover:underline"
                      @click.prevent="openDetail(issue.Key)"
                    >
                      {{ issue.Key }}
                    </a>
                    <span v-else-if="column.id === 'summary'" class="block min-w-48 max-w-md truncate text-fg" :title="issue.Title">{{ issue.Title }}</span>
                    <span
                      v-else-if="column.id === 'status'"
                      :class="['inline-block max-w-32 truncate px-2 py-0.5 rounded text-xs font-semibold whitespace-nowrap', STATUS_CATEGORY_BADGE_CLASS[issue.StatusCategory]]"
                    >
                      {{ issue.StatusName }}
                    </span>
                    <span v-else-if="column.id === 'assignee' || column.id === 'reporter'" class="flex items-center gap-2 whitespace-nowrap">
                      <template v-if="personOf(issue, column.id)">
                        <UserAvatar :user="personOf(issue, column.id)" size="sm" />
                        <span class="text-fg">{{ personOf(issue, column.id).DisplayName }}</span>
                      </template>
                      <span v-else class="text-muted">{{ column.id === 'assignee' ? '未割り当て' : 'なし' }}</span>
                    </span>
                    <span v-else-if="column.id === 'priority'" class="flex items-center gap-1 whitespace-nowrap">
                      <PriorityIcon :priority="issue.Priority" />
                      <span class="text-fg">{{ priorityLabel(issue.Priority) }}</span>
                    </span>
                    <span v-else-if="column.id === 'created'" class="text-muted whitespace-nowrap">{{ formatDate(issue.CreatedAt) }}</span>
                    <span v-else-if="column.id === 'updated'" class="text-muted whitespace-nowrap">{{ formatDate(issue.UpdatedAt) }}</span>
                    <span v-else-if="column.id === 'due'" class="text-muted whitespace-nowrap">{{ issue.Deadline || '-' }}</span>
                  </td>
                </tr>
                <tr v-if="!loading && !result.Issues.length">
                  <td :colspan="visibleColumns.length" class="px-3 py-8 text-center text-sm text-muted">条件に合う課題はありません。</td>
                </tr>
                <tr v-if="loading && !result.Issues.length">
                  <td :colspan="visibleColumns.length" class="px-3 py-8 text-center text-sm text-muted">読み込み中...</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- ページング -->
          <div class="flex flex-wrap items-center gap-3 px-4 py-3 border-t border-line text-sm text-muted">
            <span>{{ rangeText }}</span>
            <div v-if="pageCount > 1" class="ml-auto flex items-center gap-1">
              <button type="button" :class="pageButton" :disabled="search.page <= 1" @click="goToPage(search.page - 1)">前へ</button>
              <span class="px-2">{{ search.page }} / {{ pageCount }}</span>
              <button type="button" :class="pageButton" :disabled="search.page >= pageCount" @click="goToPage(search.page + 1)">次へ</button>
            </div>
          </div>
        </section>

        <!-- 詳細（Jira の分割ビュー） -->
        <section v-if="detailKey" class="w-full xl:w-1/2 min-w-0 xl:sticky xl:top-16 xl:max-h-[calc(100vh-5rem)] overflow-y-auto bg-surface rounded-2xl px-6 py-5 shadow-xl">
          <p v-if="!detailTask" class="m-0 text-sm text-muted">{{ detailLoading ? '読み込み中...' : `課題「${detailKey}」が見つかりません。` }}</p>
          <IssueDetail
            v-else
            :task="detailTask"
            :epics="epics"
            :spaces="spaces"
            :subtasks="allTodos.filter(todo => todo.ParentID === detailTask.ID)"
            :parent-task="detailTask.ParentID ? allTodos.find(todo => todo.ID === detailTask.ParentID) || null : null"
            :all-todos="allTodos"
            :statuses="statuses"
            @update="handleUpdate"
            @subtasks-changed="refreshDetail"
            @links-changed="refreshDetail"
            @comments-changed="refreshDetail"
            @worklogs-changed="refreshDetail"
            @fix-versions-changed="refreshDetail"
            @components-changed="refreshDetail"
            @open="openDetail"
          >
            <template #actions>
              <NuxtLink
                :to="issuePath(detailTask.Key)"
                class="p-1.5 rounded text-muted hover:bg-raised hover:text-fg"
                title="全画面で開く"
              >
                <svg class="w-5 h-5 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <path d="M15 3h6v6" />
                  <path d="M9 21H3v-6" />
                  <path d="M21 3l-7 7" />
                  <path d="M3 21l7-7" />
                </svg>
              </NuxtLink>
              <button
                type="button"
                class="bg-transparent border-0 text-muted text-2xl cursor-pointer p-0 w-8 h-8 flex items-center justify-center rounded hover:bg-raised hover:text-muted"
                title="閉じる"
                @click="closeDetail"
              >
                ×
              </button>
            </template>
          </IssueDetail>
        </section>
      </div>
    </div>

    <ConfirmDialog
      v-if="deletingFilter && currentFilter"
      title="フィルターを削除しますか？"
      :message="`「${currentFilter.name}」を削除します。課題は消えません。`"
      @confirm="deleteFilter"
      @cancel="deletingFilter = false"
    />

    <div v-if="error" class="fixed bottom-8 right-8 bg-danger/10 text-danger px-6 py-4 rounded-xl shadow-xl max-w-96 text-sm font-semibold border-l-4 border-danger animate-slideIn">
      {{ error }}
    </div>
  </SpaceShell>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue';
import SpaceShell from '~/components/SpaceShell.vue';
import PageToolbar from '~/components/PageToolbar.vue';
import FilterMenu from '~/components/FilterMenu.vue';
import IssueDetail from '~/components/IssueDetail.vue';
import IssueTypeIcon from '~/components/IssueTypeIcon.vue';
import PriorityIcon from '~/components/PriorityIcon.vue';
import UserAvatar from '~/components/UserAvatar.vue';
import JqlInput from '~/components/JqlInput.vue';
import ToolbarMenu from '~/components/ToolbarMenu.vue';
import ConfirmDialog from '~/components/ConfirmDialog.vue';
import { filtersApi } from '~/api/filters';
import { useSavedFilters } from '~/composables/useSavedFilters';
import { DEFAULT_FILTERS, STAR_ICON, filterLink, findFilter } from '~/utils/filters';
import { toolbarButtonClass, toolbarChipClass } from '~/utils/toolbar';
import { todosApi } from '~/api/todos';
import { spacesApi } from '~/api/spaces';
import { epicsApi } from '~/api/epics';
import { statusesApi } from '~/api/statuses';
import { usersApi } from '~/api/users';
import { labelsApi } from '~/api/labels';
import { sprintsApi } from '~/api/sprints';
import { versionsApi } from '~/api/versions';
import { issuePath } from '~/utils/spaceRoutes';
import { ISSUE_TYPE_OPTIONS, SUBTASK_ISSUE_TYPE } from '~/utils/issueTypes';
import { PRIORITY_OPTIONS } from '~/utils/priorities';
import { STATUS_CATEGORY_BADGE_CLASS } from '~/utils/statuses';
import {
  ISSUE_COLUMNS, PER_PAGE, UNASSIGNED, DEFAULT_SORT, DEFAULT_ORDER,
  parseIssueSearch, withIssueSearch, issueSearchParams, isSearching, loadColumns, saveColumns
} from '~/utils/issueSearch';
import { basicSearchToJql, jqlSortOf, withJqlSort } from '~/utils/jql';

useHead({ title: '課題 - Kanban' });

const route = useRoute();

const spaces = ref([]);
const epics = ref([]);
const statuses = ref([]);
const users = ref([]);
const labels = ref([]);
const sprints = ref([]);
const versions = ref([]);
const result = ref({ Issues: [], Total: 0, Page: 1, PerPage: PER_PAGE });
const loading = ref(true);
const error = ref('');

// 条件は URL のクエリ（例: /issues?space=DEV&status=進行中&sort=priority&order=asc）
const search = computed(() => parseIssueSearch(route.query));
const searching = computed(() => isSearching(search.value));

// 詳細検索（JQL）。URL の ?jql=… があれば JQL で検索する（Jira と同じ）
const jqlMode = computed(() => typeof route.query.jql === 'string');
const jql = computed(() => (jqlMode.value ? route.query.jql : ''));
// 入力中の JQL。検索するまで URL には入れない
const jqlDraft = ref(jql.value);
watch(jql, (value) => {
  jqlDraft.value = value;
});
// JQL の誤り { message, position }
const jqlError = ref(null);
const jqlMasters = computed(() => ({
  spaces: spaces.value, statuses: statuses.value, users: users.value, labels: labels.value,
  epics: epics.value, sprints: sprints.value, versions: versions.value
}));

const BASIC_KEYS = ['space', 'type', 'status', 'assignee', 'label', 'text', 'sort', 'order', 'page'];

// 基本の条件を JQL に書き換えて切り替える
const switchToJql = () => {
  if (jqlMode.value) return;
  const query = { ...route.query, jql: basicSearchToJql(search.value, users.value) };
  for (const key of BASIC_KEYS) delete query[key];
  navigateTo({ query });
};
// 基本に戻すと、条件はクリアした状態から選び直す（JQL は基本の条件で表せないことがあるため）
const switchToBasic = () => {
  if (!jqlMode.value) return;
  const { jql: _, page: __, ...query } = route.query;
  textDraft.value = '';
  navigateTo({ query });
};
const submitJql = () => {
  const { page: _, ...query } = route.query;
  navigateTo({ query: { ...query, jql: jqlDraft.value.trim() } });
  // 同じ JQL でも検索し直す（課題が変わっているかもしれない）
  if (jqlDraft.value.trim() === jql.value) fetchIssues();
};
const modeButton = (active) => [
  'px-2.5 py-0.5 border-0 rounded text-xs font-semibold cursor-pointer',
  active ? 'bg-surface text-fg shadow-sm' : 'bg-transparent text-muted hover:text-fg'
];

// 保存済みフィルター（Jira のフィルター）。URL の ?filter= が開いているフィルター（既定のフィルターの id か、保存したフィルターの ID）
const { filters: savedFilters, reload: reloadFilters } = useSavedFilters();
const currentFilter = computed(() => findFilter(route.query.filter, savedFilters.value));
const currentSavedFilter = computed(() => savedFilters.value.find(filter => String(filter.ID) === route.query.filter) || null);
// 今の条件の JQL。基本モードでは条件を JQL に書き換えたもの
const currentJql = computed(() => (jqlMode.value ? jql.value.trim() : basicSearchToJql(search.value, users.value)));
// 開いたフィルターから条件を変えたか（空白の違いは無視する）
const normalizeJql = (value) => value.trim().replace(/\s+/g, ' ');
const filterModified = computed(() => !!currentFilter.value && normalizeJql(currentJql.value) !== normalizeJql(currentFilter.value.jql));
const crumbs = computed(() => (currentFilter.value
  ? [{ label: '課題一覧', to: '/issues' }, { label: currentFilter.value.name }]
  : [{ label: '課題一覧' }]));

const filterItemClass = (id) => [
  'block px-2 py-1 rounded no-underline',
  route.query.filter === id ? 'bg-raised text-fg font-semibold' : 'text-fg hover:bg-raised'
];
const starButtonClass = (active) => [
  'shrink-0 p-1 border-0 rounded bg-transparent cursor-pointer hover:bg-raised',
  active ? 'text-warning' : 'text-muted hover:text-fg'
];

const newFilterName = ref('');
const newFilterFavourite = ref(false);
const renameDraft = ref('');
const filterFormError = ref('');
const filterSaving = ref(false);
const deletingFilter = ref(false);
// 別のフィルターを開いたら、名前の欄と前の誤りを戻す
watch(() => route.query.filter, () => {
  renameDraft.value = currentFilter.value ? currentFilter.value.name : '';
  filterFormError.value = '';
}, { immediate: true });
watch(() => currentFilter.value?.name, (name) => {
  renameDraft.value = name || '';
});

// フィルターを開き直す（保存した JQL で検索する）。selectedIssue（開いている詳細）は残す
const openFilter = (filter, replace = false) => {
  const link = filterLink(filter);
  const query = route.query.selectedIssue ? { ...link.query, selectedIssue: route.query.selectedIssue } : link.query;
  return navigateTo({ path: link.path, query }, { replace });
};

const runFilterAction = async (action) => {
  filterSaving.value = true;
  filterFormError.value = '';
  try {
    await action();
    return true;
  } catch (err) {
    filterFormError.value = err.message;
    console.error('Failed to save filter:', err);
    return false;
  } finally {
    filterSaving.value = false;
  }
};

// 今の条件に名前を付けて保存し、そのフィルターを開く
const saveAsFilter = async (close) => {
  const ok = await runFilterAction(async () => {
    const { filter } = await filtersApi.create({ name: newFilterName.value, jql: currentJql.value, favourite: newFilterFavourite.value });
    await reloadFilters();
    await openFilter(filter);
  });
  if (!ok) return;
  newFilterName.value = '';
  newFilterFavourite.value = false;
  close();
};

// 開いているフィルターを今の条件で上書きする
const overwriteFilter = async () => {
  const filter = currentSavedFilter.value;
  if (!filter) return;
  try {
    const res = await filtersApi.update(filter.ID, { name: filter.Name, jql: currentJql.value });
    await reloadFilters();
    await openFilter(res.filter, true);
  } catch (err) {
    error.value = 'フィルターの保存に失敗しました: ' + err.message;
    console.error('Failed to overwrite filter:', err);
  }
};

// 名前だけを変える（条件は保存してあるもののまま）
const renameFilter = async (close) => {
  const filter = currentSavedFilter.value;
  if (!filter) return;
  const ok = await runFilterAction(async () => {
    await filtersApi.update(filter.ID, { name: renameDraft.value, jql: filter.JQL });
    await reloadFilters();
  });
  if (ok) close();
};

const toggleFavourite = async (filter) => {
  if (!filter) return;
  try {
    await filtersApi.setFavourite(filter.ID, !filter.Favourite);
    await reloadFilters();
  } catch (err) {
    error.value = 'お気に入りの変更に失敗しました: ' + err.message;
    console.error('Failed to set favourite:', err);
  }
};

// 削除しても今の条件の一覧はそのまま出す（フィルターだけを外す）
const deleteFilter = async () => {
  const filter = currentSavedFilter.value;
  deletingFilter.value = false;
  if (!filter) return;
  try {
    await filtersApi.delete(filter.ID);
    await reloadFilters();
    const { filter: _, ...query } = route.query;
    await navigateTo({ query }, { replace: true });
  } catch (err) {
    error.value = 'フィルターの削除に失敗しました: ' + err.message;
    console.error('Failed to delete filter:', err);
  }
};

// 表の見出しに出す並べ替え。JQL では ORDER BY の最初の列
const currentSort = computed(() => (jqlMode.value ? jqlSortOf(jql.value) : { sort: search.value.sort, order: search.value.order }));

// スペースを 1 つだけ選んでいるときは、左ナビをそのスペースのものにする
const currentSpace = computed(() =>
  search.value.spaces.length === 1 ? spaces.value.find(space => space.Key === search.value.spaces[0]) || null : null
);

// 条件を変えたら 1 ページ目に戻す。履歴を増やさないよう URL は置き換える
const update = (changes) => {
  navigateTo({ query: withIssueSearch(route.query, { ...search.value, page: 1, ...changes }) }, { replace: true });
};

const clearSearch = () => {
  textDraft.value = '';
  update({ spaces: [], types: [], statuses: [], assignees: [], labels: [], text: '' });
};

// テキストは入力が止まってから検索する
const textDraft = ref(search.value.text);
let textTimer = null;
const handleTextInput = (event) => {
  textDraft.value = event.target.value;
  clearTimeout(textTimer);
  textTimer = setTimeout(() => update({ text: textDraft.value }), 300);
};
// 戻る・進むなどで URL のテキストが変わったら入力欄も合わせる
watch(() => search.value.text, (text) => {
  if (text.trim() !== textDraft.value.trim()) textDraft.value = text;
});

// 選択肢
const spaceOptions = computed(() => [
  ...spaces.value.filter(space => space.Key).map(space => ({ value: space.Key, label: `${space.Title}（${space.Key}）` })),
  { value: 'TASK', label: 'スペース未割り当て' }
]);
const typeOptions = [...ISSUE_TYPE_OPTIONS, { value: SUBTASK_ISSUE_TYPE, label: 'サブタスク' }];
// ステータスは名前でまとめる。スペースを選んでいれば、そのスペースのステータスだけ
const statusOptions = computed(() => {
  const spaceIds = search.value.spaces
    .map(key => (key === 'TASK' ? 0 : spaces.value.find(space => space.Key === key)?.ID))
    .filter(id => id !== undefined);
  const names = [];
  for (const status of statuses.value) {
    if (spaceIds.length && !spaceIds.includes(status.SpaceID)) continue;
    if (!names.includes(status.Name)) names.push(status.Name);
  }
  // 選んでいるのに選択肢にない名前（スペースを変えたとき）も外せるよう残す
  for (const name of search.value.statuses) {
    if (!names.includes(name)) names.push(name);
  }
  return names.map(name => ({ value: name, label: name }));
});
const assigneeOptions = computed(() => [
  { value: UNASSIGNED, label: '未割り当て' },
  ...users.value.map(user => ({ value: user.ID, label: user.DisplayName }))
]);
const labelOptions = computed(() => labels.value.map(label => ({ value: label, label })));

// 表示する列
const columns = ref(loadColumns());
const columnOptions = ISSUE_COLUMNS.map(column => ({ value: column.id, label: column.label }));
const visibleColumns = computed(() => ISSUE_COLUMNS.filter(column => columns.value.includes(column.id)));
const changeColumns = (next) => {
  // 列をすべて外すと表が空になるので、最後の 1 列は残す
  if (!next.length) return;
  columns.value = ISSUE_COLUMNS.map(column => column.id).filter(id => next.includes(id));
  saveColumns(columns.value);
};

// 並べ替え。同じ列を押すと昇順・降順を切り替える。作成日・更新日は新しい順から始める。
// JQL では ORDER BY をその列に書き換える
const sortBy = (column) => {
  const { sort, order } = currentSort.value;
  const next = sort === column.sort
    ? { sort, order: order === 'asc' ? 'desc' : 'asc' }
    : { sort: column.sort, order: column.sort === DEFAULT_SORT || column.sort === 'updated' ? DEFAULT_ORDER : 'asc' };
  if (jqlMode.value) {
    const { page: _, ...query } = route.query;
    navigateTo({ query: { ...query, jql: withJqlSort(jql.value, next.sort, next.order) } }, { replace: true });
    return;
  }
  update(next);
};
const ariaSort = (column) =>
  currentSort.value.sort === column.sort ? (currentSort.value.order === 'asc' ? 'ascending' : 'descending') : 'none';

// ページング
const pageCount = computed(() => Math.max(1, Math.ceil(result.value.Total / PER_PAGE)));
const rangeText = computed(() => {
  const { Total } = result.value;
  if (!Total) return '0 件';
  const start = (search.value.page - 1) * PER_PAGE + 1;
  const end = Math.min(search.value.page * PER_PAGE, Total);
  return `${Total} 件中 ${start}〜${end} 件`;
});
const goToPage = (page) => {
  navigateTo({ query: withIssueSearch(route.query, { ...search.value, page }) });
};
const pageButton = 'px-3 py-1 bg-raised text-fg border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-line disabled:opacity-40 disabled:cursor-not-allowed';

// 表示
const personOf = (issue, column) => (column === 'assignee' ? issue.Assignee : issue.Reporter);
const priorityLabel = (priority) => PRIORITY_OPTIONS.find(option => option.value === priority)?.label ?? priority;
const formatDate = (value) => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return `${date.getFullYear()}/${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')}`;
};

// 一覧の取得。条件が変わるたびに取り直し、古い応答で上書きしない
let requestId = 0;
const fetchIssues = async () => {
  const id = ++requestId;
  loading.value = true;
  try {
    const data = jqlMode.value
      ? await todosApi.searchJql(jql.value, search.value.page, PER_PAGE)
      : await todosApi.search(issueSearchParams(search.value));
    if (id !== requestId) return;
    error.value = '';
    jqlError.value = null;
    result.value = data;
    // 課題を消すなどしてページが範囲外になったら最後のページへ
    if (search.value.page > 1 && !data.Issues.length && data.Total) {
      goToPage(Math.ceil(data.Total / PER_PAGE));
    }
  } catch (err) {
    if (id !== requestId) return;
    // JQL の誤りは入力欄の下に出し、一覧は空にする
    if (jqlMode.value && typeof err.position === 'number') {
      jqlError.value = { message: err.message, position: err.position };
      result.value = { Issues: [], Total: 0, Page: 1, PerPage: PER_PAGE };
      return;
    }
    error.value = '課題の取得に失敗しました: ' + err.message;
    console.error('Failed to search issues:', err);
  } finally {
    if (id === requestId) loading.value = false;
  }
};

const fetchMasters = async () => {
  try {
    [spaces.value, epics.value, statuses.value, users.value, labels.value, sprints.value, versions.value] = await Promise.all([
      spacesApi.fetchAll(), epicsApi.fetchAll(), statusesApi.fetchAll(), usersApi.fetchAll(), labelsApi.fetchAll(),
      sprintsApi.fetchAll(), versionsApi.fetchAll()
    ]);
  } catch (err) {
    error.value = '検索条件の候補の取得に失敗しました: ' + err.message;
    console.error('Failed to fetch masters:', err);
  }
};

watch(() => JSON.stringify([search.value, jql.value, jqlMode.value]), fetchIssues);

// 詳細（URL の ?selectedIssue=KAN-12 で表す。ボードと同じ）
const detailKey = computed(() => (typeof route.query.selectedIssue === 'string' ? route.query.selectedIssue : ''));
const detailTask = ref(null);
const detailLoading = ref(false);
// 子課題・親課題とリンクする課題の候補を出すための全課題。詳細を開いたときだけ読む
const allTodos = ref([]);

const openDetail = (key) => {
  navigateTo({ query: { ...route.query, selectedIssue: key } });
};
const closeDetail = () => {
  const { selectedIssue, ...rest } = route.query;
  navigateTo({ query: rest });
};

const fetchDetail = async () => {
  if (!detailKey.value) {
    detailTask.value = null;
    return;
  }
  detailLoading.value = true;
  try {
    const [issue, todos] = await Promise.all([todosApi.fetchByKey(detailKey.value), todosApi.fetchAll()]);
    detailTask.value = issue;
    allTodos.value = todos;
    // スペースを変えるなどしてキーが変わったら URL も合わせる
    if (issue && issue.Key !== detailKey.value) {
      navigateTo({ query: { ...route.query, selectedIssue: issue.Key } }, { replace: true });
    }
  } catch (err) {
    error.value = '課題の取得に失敗しました: ' + err.message;
    console.error('Failed to fetch issue:', err);
  } finally {
    detailLoading.value = false;
  }
};

watch(detailKey, (key) => {
  if (detailTask.value && detailTask.value.Key === key) return;
  fetchDetail();
});

// 詳細で変えたら、詳細と一覧（並びや絞り込みが変わりうる）を取り直す
const refreshDetail = async () => {
  await Promise.all([fetchDetail(), fetchIssues()]);
};

const handleUpdate = async (payload) => {
  try {
    error.value = '';
    const res = await todosApi.update(detailTask.value.ID, payload);
    detailTask.value = res.todo;
    if (res.todo.Key !== detailKey.value) {
      navigateTo({ query: { ...route.query, selectedIssue: res.todo.Key } }, { replace: true });
    }
  } catch (err) {
    error.value = '課題の更新に失敗しました: ' + err.message;
    console.error('Failed to update issue:', err);
  }
  await refreshDetail();
};

const handleSpacesChanged = async () => {
  await fetchMasters();
  await fetchIssues();
};

onMounted(async () => {
  await Promise.all([fetchMasters(), fetchIssues(), fetchDetail(), reloadFilters()]);
});
</script>
