<template>
  <!-- スペースのバックログ（Jira のバックログ）。完了していない課題を順位（ボードと共通の並び順）の順に 1 行ずつ並べる -->
  <SpaceShell :spaces="spaces" :todos="todos" :current-space="space" @changed="fetchAll">
    <div class="px-8 py-8">
      <p v-if="loading" class="m-0 text-white">読み込み中...</p>
      <div v-else-if="!space" class="max-w-3xl bg-white rounded-2xl px-8 py-6 shadow-xl text-gray-700">
        スペース「{{ spaceKey }}」が見つかりません。
        <NuxtLink to="/board" class="ml-2 text-blue-600 font-semibold">すべてのスペースのボードを開く</NuxtLink>
      </div>

      <template v-else>
        <h2 class="m-0 text-2xl font-bold text-white">バックログ</h2>

        <section class="mt-6 max-w-6xl bg-white rounded-2xl shadow-xl overflow-hidden">
          <div class="flex flex-wrap items-center gap-3 px-5 py-3 border-b border-gray-200">
            <h3 class="m-0 text-base font-bold text-gray-900">バックログ</h3>
            <span class="text-sm text-gray-500">{{ rows.length }} 件の課題</span>
            <span v-if="hasStoryPoints" class="px-2 py-0.5 rounded-full bg-gray-100 text-xs font-semibold text-gray-700" title="ストーリーポイントの合計">
              {{ formatStoryPoints(sumStoryPoints(rows)) }} pt
            </span>
            <template v-if="selectedIds.size > 1">
              <span class="ml-auto text-sm font-semibold text-blue-700">{{ selectedIds.size }} 件を選択中</span>
              <button
                type="button"
                class="px-2 py-1 border-0 rounded bg-transparent text-sm text-gray-600 cursor-pointer hover:bg-gray-100"
                @click="clearSelection"
              >
                選択を解除
              </button>
            </template>
            <span v-else class="ml-auto text-xs text-gray-500">ドラッグで順位を変更（Shift / ⌘ クリックで複数選択）</span>
          </div>

          <!-- 課題の行。ドラッグで順位を変える（選択中の行はまとめて動かす） -->
          <ol
            class="m-0 p-0 list-none"
            aria-label="バックログの課題"
            @dragover="handleDragOver"
            @dragleave="handleDragLeave"
            @drop="handleDrop"
          >
            <template v-for="(row, index) in rows" :key="row.ID">
              <li v-if="dropIndex === index" class="h-0.5 bg-blue-500" aria-hidden="true"></li>
              <li
                :data-row-id="row.ID"
                :class="[
                  'flex items-center gap-3 px-5 py-2 border-b border-gray-100 text-sm cursor-pointer select-none',
                  selectedIds.has(row.ID) ? 'bg-blue-50' : 'bg-white hover:bg-gray-50',
                  draggingIds.has(row.ID) ? 'opacity-50' : ''
                ]"
                draggable="true"
                :aria-selected="selectedIds.has(row.ID)"
                @click="handleRowClick($event, row)"
                @dragstart="handleDragStart($event, row)"
                @dragend="handleDragEnd"
              >
                <IssueTypeIcon :type="row.IssueType" />
                <span class="shrink-0 w-20 font-mono text-xs font-semibold text-gray-500">{{ row.Key }}</span>
                <span class="flex-1 min-w-0 truncate text-gray-900" :title="row.Title">{{ row.Title }}</span>
                <span
                  v-if="epicOf(row)"
                  class="shrink-0 max-w-40 truncate px-2 py-0.5 rounded bg-purple-100 text-xs font-semibold text-purple-800"
                  :title="`エピック: ${epicOf(row).Title}`"
                >
                  {{ epicOf(row).Title }}
                </span>
                <span :class="['shrink-0 max-w-32 truncate px-2 py-0.5 rounded text-xs font-semibold', STATUS_CATEGORY_BADGE_CLASS[row.StatusCategory]]">
                  {{ row.StatusName }}
                </span>
                <span class="shrink-0 w-6 flex justify-center">
                  <UserAvatar v-if="row.Assignee" :user="row.Assignee" size="sm" :title="`担当者: ${row.Assignee.DisplayName}`" />
                </span>
                <span
                  :class="[
                    'shrink-0 min-w-8 h-6 px-1.5 rounded-full text-xs font-semibold flex items-center justify-center',
                    row.StoryPoints != null ? 'bg-gray-200 text-gray-700' : 'text-gray-300'
                  ]"
                  :title="row.StoryPoints != null ? `ストーリーポイント: ${formatStoryPoints(row.StoryPoints)}` : 'ストーリーポイント未設定'"
                >
                  {{ row.StoryPoints != null ? formatStoryPoints(row.StoryPoints) : '-' }}
                </span>
              </li>
            </template>
            <li v-if="dropIndex === rows.length" class="h-0.5 bg-blue-500" aria-hidden="true"></li>
            <li v-if="!rows.length" class="px-5 py-6 text-center text-sm text-gray-500">完了していない課題はありません</li>
          </ol>

          <!-- その場で課題を作成する（Jira の「+ 課題を作成」）。続けて入力できるよう、作成後も開いたままにする -->
          <form v-if="creating" class="flex items-center gap-2 px-5 py-2" @submit.prevent="handleCreate">
            <select
              v-model="newIssueType"
              aria-label="課題タイプ"
              class="px-2 py-1.5 border-2 border-gray-300 rounded-md text-sm text-gray-800 bg-white cursor-pointer focus:outline-none focus:border-blue-500"
            >
              <option v-for="option in ISSUE_TYPE_OPTIONS" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
            <input
              ref="newTitleInput"
              v-model="newTitle"
              type="text"
              placeholder="要約を入力して Enter"
              aria-label="作成する課題の要約"
              class="flex-1 min-w-0 px-3 py-1.5 border-2 border-blue-500 rounded-md text-sm text-gray-900 bg-white focus:outline-none"
              @keydown.enter="handleCreateEnter"
              @keydown.esc="closeCreate"
            />
            <button
              type="submit"
              :disabled="!newTitle.trim() || saving"
              class="px-3 py-1.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              作成
            </button>
            <button
              type="button"
              class="px-3 py-1.5 bg-transparent border-0 rounded-md text-sm text-gray-600 cursor-pointer hover:bg-gray-100"
              @click="closeCreate"
            >
              キャンセル
            </button>
          </form>
          <button
            v-else
            type="button"
            class="w-full px-5 py-2.5 bg-transparent border-0 text-left text-sm font-semibold text-gray-600 cursor-pointer hover:bg-gray-50"
            @click="openCreate"
          >
            + 課題を作成
          </button>
        </section>
      </template>
    </div>

    <div v-if="error" class="fixed bottom-8 right-8 bg-red-100 text-red-800 px-6 py-4 rounded-xl shadow-xl max-w-96 text-sm font-semibold border-l-4 border-red-500 animate-slideIn">
      {{ error }}
    </div>

    <!-- 課題の詳細のモーダル（ボードと同じく URL の ?selectedIssue=KAN-12 で表す） -->
    <div
      v-if="detailTask"
      class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50"
      @click.self="closeDetail"
    >
      <div class="bg-white rounded-2xl px-8 py-6 shadow-2xl mx-4 max-w-5xl w-full max-h-[90vh] overflow-y-auto">
        <IssueDetail
          :task="detailTask"
          :epics="epics"
          :spaces="spaces"
          :subtasks="todos.filter(todo => todo.ParentID === detailTask.ID)"
          :parent-task="detailTask.ParentID ? todos.find(todo => todo.ID === detailTask.ParentID) || null : null"
          :all-todos="todos"
          :statuses="statuses"
          @update="handleUpdate"
          @subtasks-changed="fetchTodos"
          @links-changed="fetchTodos"
          @comments-changed="fetchTodos"
          @worklogs-changed="fetchTodos"
          @open="openDetail"
        >
          <template #actions>
            <NuxtLink
              :to="issuePath(detailTask.Key)"
              class="p-1.5 rounded text-gray-500 hover:bg-gray-100 hover:text-gray-700"
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
              class="bg-transparent border-0 text-gray-400 text-2xl cursor-pointer p-0 w-8 h-8 flex items-center justify-center rounded hover:bg-gray-100 hover:text-gray-600"
              title="閉じる"
              @click="closeDetail"
            >
              ×
            </button>
          </template>
        </IssueDetail>
      </div>
    </div>
  </SpaceShell>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue';
import SpaceShell from '~/components/SpaceShell.vue';
import IssueDetail from '~/components/IssueDetail.vue';
import IssueTypeIcon from '~/components/IssueTypeIcon.vue';
import UserAvatar from '~/components/UserAvatar.vue';
import { todosApi } from '~/api/todos';
import { epicsApi } from '~/api/epics';
import { spacesApi } from '~/api/spaces';
import { statusesApi } from '~/api/statuses';
import { STATUS_CATEGORY_BADGE_CLASS, isDoneCategory } from '~/utils/statuses';
import { ISSUE_TYPE_OPTIONS, DEFAULT_ISSUE_TYPE } from '~/utils/issueTypes';
import { formatStoryPoints, sumStoryPoints } from '~/utils/storyPoints';
import { issuePath } from '~/utils/spaceRoutes';

const route = useRoute();
const spaceKey = computed(() => String(route.params.key));

const todos = ref([]);
const epics = ref([]);
const spaces = ref([]);
const statuses = ref([]);
const loading = ref(true);
const error = ref('');
// ヘッダーの「作成」で課題が作られたら増える（GlobalCreate と共有）
const issuesVersion = useState('issuesVersion', () => 0);

const space = computed(() => spaces.value.find(s => s.Key === spaceKey.value) || null);

useHead(() => ({ title: space.value ? `バックログ - ${space.value.Title} - Kanban` : 'Kanban' }));

// バックログの行。スペースの完了していない課題（サブタスクは親課題の詳細で見せる）を、順位（GET /todos の並び順）の順に並べる
const rows = computed(() =>
  space.value
    ? todos.value.filter(todo => todo.SpaceID === space.value.ID && !todo.ParentID && !isDoneCategory(todo.StatusCategory))
    : []
);
const hasStoryPoints = computed(() => rows.value.some(todo => todo.StoryPoints != null));

const epicOf = (todo) => (todo.EpicID ? epics.value.find(epic => epic.ID === todo.EpicID) || null : null);

const fetchTodos = async () => {
  try {
    todos.value = await todosApi.fetchAll();
  } catch (err) {
    error.value = '課題の取得に失敗しました: ' + err.message;
    console.error('Failed to fetch todos:', err);
  }
};

const fetchAll = async () => {
  try {
    error.value = '';
    [spaces.value, epics.value, statuses.value] = await Promise.all([spacesApi.fetchAll(), epicsApi.fetchAll(), statusesApi.fetchAll()]);
  } catch (err) {
    error.value = 'スペースの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch spaces:', err);
  }
  await fetchTodos();
};

// ヘッダーの「作成」で作られた課題を出す
watch(issuesVersion, fetchTodos);

// 選択（Jira と同じく、クリックで 1 件を選んで詳細を開き、⌘/Ctrl クリックで追加・解除、Shift クリックで範囲を選ぶ）
const selectedIds = ref(new Set());
// Shift クリックの範囲の起点
const anchorId = ref(null);

const clearSelection = () => {
  selectedIds.value = new Set();
  anchorId.value = null;
};

const handleRowClick = (event, row) => {
  if (event.shiftKey && anchorId.value !== null) {
    const ids = rows.value.map(todo => todo.ID);
    const from = ids.indexOf(anchorId.value);
    const to = ids.indexOf(row.ID);
    if (from !== -1) {
      selectedIds.value = new Set(ids.slice(Math.min(from, to), Math.max(from, to) + 1));
      return;
    }
  }
  if (event.metaKey || event.ctrlKey) {
    const next = new Set(selectedIds.value);
    if (next.has(row.ID)) next.delete(row.ID);
    else next.add(row.ID);
    selectedIds.value = next;
    anchorId.value = row.ID;
    return;
  }
  selectedIds.value = new Set([row.ID]);
  anchorId.value = row.ID;
  openDetail(row.Key);
};

// 一覧から消えた課題（完了にした・削除した）は選択から外す
watch(rows, (list) => {
  const ids = new Set(list.map(todo => todo.ID));
  if ([...selectedIds.value].some(id => !ids.has(id))) {
    selectedIds.value = new Set([...selectedIds.value].filter(id => ids.has(id)));
  }
});

// ドラッグ&ドロップで順位を変える。選択中の行をドラッグしたら、選択中の行をまとめて（順位の順のまま）動かす
const draggingIds = ref(new Set());
// 挿入する位置（rows のインデックス）。null の場合はドラッグ中でない
const dropIndex = ref(null);
const BACKLOG_DRAG_TYPE = 'application/x-kanban-backlog';

const handleDragStart = (event, row) => {
  if (!selectedIds.value.has(row.ID)) {
    selectedIds.value = new Set([row.ID]);
    anchorId.value = row.ID;
  }
  draggingIds.value = new Set(selectedIds.value);
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData(BACKLOG_DRAG_TYPE, [...draggingIds.value].join(','));
};

const handleDragEnd = () => {
  draggingIds.value = new Set();
  dropIndex.value = null;
};

// マウスの位置から挿入先を求める。行の上半分ならその前、下半分ならその後ろ
const getDropIndex = (event) => {
  const items = Array.from(event.currentTarget.querySelectorAll('[data-row-id]'));
  const index = items.findIndex(item => {
    const rect = item.getBoundingClientRect();
    return event.clientY < rect.top + rect.height / 2;
  });
  return index === -1 ? items.length : index;
};

const handleDragOver = (event) => {
  if (!Array.from(event.dataTransfer.types).includes(BACKLOG_DRAG_TYPE)) return;
  event.preventDefault();
  event.dataTransfer.dropEffect = 'move';
  dropIndex.value = getDropIndex(event);
};

const handleDragLeave = (event) => {
  if (event.currentTarget.contains(event.relatedTarget)) return;
  dropIndex.value = null;
};

const handleDrop = async (event) => {
  if (!Array.from(event.dataTransfer.types).includes(BACKLOG_DRAG_TYPE)) return;
  event.preventDefault();
  const index = dropIndex.value ?? getDropIndex(event);
  const moving = new Set(event.dataTransfer.getData(BACKLOG_DRAG_TYPE).split(',').map(Number));
  draggingIds.value = new Set();
  dropIndex.value = null;

  const list = rows.value;
  const movingRows = list.filter(todo => moving.has(todo.ID));
  if (!movingRows.length) return;
  // 動かした後の行の並び。変わらなければ何もしない
  const rest = list.filter(todo => !moving.has(todo.ID));
  const insertAt = list.slice(0, index).filter(todo => !moving.has(todo.ID)).length;
  const next = [...rest.slice(0, insertAt), ...movingRows, ...rest.slice(insertAt)];
  if (next.every((todo, i) => todo.ID === list[i].ID)) return;

  // 直後の行の前へ入れる。末尾なら、最後の行の（全課題の並びで）次の課題の前へ入れ、ほかのスペースや完了した課題の順位を崩さない
  let beforeId = rest[insertAt]?.ID ?? null;
  if (beforeId === null) {
    const last = rest[rest.length - 1];
    const others = todos.value.filter(todo => !moving.has(todo.ID));
    beforeId = last ? (others[others.indexOf(last) + 1]?.ID ?? 0) : 0;
  }

  try {
    error.value = '';
    for (const todo of movingRows) {
      await todosApi.move(todo.ID, beforeId);
    }
  } catch (err) {
    error.value = '順位の変更に失敗しました: ' + err.message;
    console.error('Failed to rank todos:', err);
  }
  await fetchTodos();
};

// その場で課題を作成する（スペースのワークフローの To Do の先頭。順位はバックログの末尾）
const creating = ref(false);
const newTitle = ref('');
const newIssueType = ref(DEFAULT_ISSUE_TYPE);
const newTitleInput = ref(null);
const saving = ref(false);

const openCreate = async () => {
  creating.value = true;
  await nextTick();
  newTitleInput.value?.focus();
};

const closeCreate = () => {
  creating.value = false;
  newTitle.value = '';
};

const handleCreate = async () => {
  const title = newTitle.value.trim();
  if (!title || saving.value) return;
  saving.value = true;
  try {
    error.value = '';
    await todosApi.create({ title, issueType: newIssueType.value, spaceId: space.value.ID });
    newTitle.value = '';
    await fetchTodos();
  } catch (err) {
    error.value = '課題の作成に失敗しました: ' + err.message;
    console.error('Failed to create todo:', err);
  } finally {
    saving.value = false;
    await nextTick();
    newTitleInput.value?.focus();
  }
};

// 日本語入力の変換確定の Enter では作成しない
const handleCreateEnter = (event) => {
  if (event.isComposing || event.keyCode === 229) event.preventDefault();
};

// 課題の詳細のモーダル
const detailTask = computed(() => {
  const key = route.query.selectedIssue;
  return key ? (todos.value.find(todo => todo.Key === key) || null) : null;
});

const openDetail = (key) => {
  navigateTo({ query: { ...route.query, selectedIssue: key } });
};

const closeDetail = () => {
  const { selectedIssue, ...query } = route.query;
  navigateTo({ query }, { replace: true });
};

// 詳細でインライン編集した内容を保存する（モーダルは開いたまま）
const handleUpdate = async (payload) => {
  const task = detailTask.value;
  try {
    error.value = '';
    const { todo } = await todosApi.update(task.ID, payload);
    // スペースを変えると課題キーが変わるので、モーダルが閉じないよう先に一覧と URL を新しいキーにする
    todos.value = todos.value.map(t => (t.ID === todo.ID ? todo : t));
    if (todo.Key !== task.Key) {
      await navigateTo({ query: { ...route.query, selectedIssue: todo.Key } }, { replace: true });
    }
  } catch (err) {
    error.value = '課題の更新に失敗しました: ' + err.message;
    console.error('Failed to update todo:', err);
  }
  await fetchTodos();
};

onMounted(async () => {
  await fetchAll();
  loading.value = false;
});
</script>
