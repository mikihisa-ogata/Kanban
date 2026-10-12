<template>
  <SpaceShell
    :spaces="spaces"
    :todos="todos"
    :current-space="currentSpace"
    @changed="handleSpacesChanged"
    @move="handleMoveToSpace"
  >
    <div v-if="spaceNotFound" class="px-8 py-8">
      <div class="bg-white rounded-2xl px-8 py-6 shadow-xl text-gray-700">
        スペース「{{ spaceKey }}」が見つかりません。
        <NuxtLink to="/board" class="ml-2 text-blue-600 font-semibold">すべてのスペースのボードを開く</NuxtLink>
      </div>
    </div>

    <div v-else class="py-8">
      <h2 class="m-0 px-8 text-2xl font-bold text-white">{{ selectedSpaceTitle }}</h2>

      <!-- クイックフィルターと検索（Jira のボード上部）。条件は URL のクエリに持つ -->
      <div class="flex flex-wrap items-center gap-2 px-8 mt-4">
        <input
          v-model="searchText"
          type="search"
          placeholder="ボードを検索"
          aria-label="ボードを検索（要約・課題キー）"
          class="w-56 px-3 py-1.5 border-2 border-transparent rounded-md text-sm text-gray-900 bg-white/90 focus:outline-none focus:bg-white focus:border-blue-500"
        />
        <!-- 担当者のアバター。押すとその担当者の課題に絞り込む（複数可） -->
        <div v-if="assigneeOptions.length" class="flex items-center pl-1" role="group" aria-label="担当者で絞り込む">
          <button
            v-for="option in assigneeOptions"
            :key="option.id"
            type="button"
            :class="[
              '-ml-1 p-0 border-0 rounded-full bg-transparent cursor-pointer ring-2 transition-transform hover:-translate-y-0.5 hover:z-10',
              filters.assignees.includes(option.id) ? 'ring-blue-600 z-10' : 'ring-white/80'
            ]"
            :title="option.user.DisplayName"
            :aria-label="option.user.DisplayName"
            :aria-pressed="filters.assignees.includes(option.id)"
            @click="toggleAssignee(option.id)"
          >
            <UserAvatar :user="option.user" />
          </button>
        </div>
        <button
          v-for="toggle in quickToggles"
          :key="toggle.key"
          type="button"
          :class="[
            'px-3 py-1.5 border-0 rounded-md text-sm font-semibold cursor-pointer',
            filters[toggle.key] ? 'bg-blue-50 text-blue-700' : 'bg-white/90 text-gray-700 hover:bg-white'
          ]"
          :aria-pressed="filters[toggle.key]"
          @click="updateFilters({ [toggle.key]: !filters[toggle.key] })"
        >
          {{ toggle.label }}
        </button>
        <FilterMenu
          label="ラベル"
          :options="labelOptions"
          :model-value="filters.labels"
          empty-text="ラベルの付いた課題がありません"
          @update:model-value="updateFilters({ labels: $event })"
        />
        <FilterMenu
          label="課題タイプ"
          :options="ISSUE_TYPE_OPTIONS"
          :model-value="filters.types"
          @update:model-value="updateFilters({ types: $event })"
        />
        <FilterMenu
          label="エピック"
          :options="epicOptions"
          :model-value="filters.epics"
          @update:model-value="updateFilters({ epics: $event })"
        />
        <button
          v-if="filtering"
          type="button"
          class="px-3 py-1.5 border-0 rounded-md bg-transparent text-sm font-semibold text-white underline-offset-2 cursor-pointer hover:underline"
          @click="clearFilters"
        >
          フィルターをクリア
        </button>
        <!-- スイムレーンの種類（Jira の「グループ化」）。スペースごとにブラウザに覚える -->
        <label class="ml-auto flex items-center gap-2 text-sm font-semibold text-white">
          グループ化
          <select
            v-model="swimlane"
            class="px-2 py-1.5 border-0 rounded-md bg-white/90 text-sm font-semibold text-gray-800 cursor-pointer hover:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option v-for="option in SWIMLANE_OPTIONS" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
      </div>

      <!-- 画面が狭いときに画面端で見切れるよう、左右の余白は外側ではなくスクロール領域の内側に持たせる -->
      <div class="overflow-x-auto px-8 pb-6 mt-6">
        <!-- 列はスペースのワークフローのステータス。すべてのスペースではカテゴリの 3 列。列が多いときは横にスクロールする -->
        <div class="flex flex-col gap-4" :style="{ minWidth: `max(64rem, ${columns.length * 15}rem)` }">
          <div class="grid gap-4 px-3" :style="columnGridStyle">
            <div
              v-for="column in columns"
              :key="column.key"
              :class="[
                'flex justify-between items-center gap-2 px-5 py-4 rounded-lg text-white shadow-md',
                STATUS_CATEGORY_HEADER_CLASS[column.category]
              ]"
            >
              <h3 class="m-0 min-w-0 truncate text-xl font-bold tracking-wide" :title="column.title">{{ column.title }}</h3>
              <div class="flex shrink-0 items-center gap-2">
                <!-- 列のストーリーポイントの合計（ポイントの付いたタスクがあるときだけ） -->
                <span
                  v-if="hasStoryPoints(column)"
                  class="bg-white/25 text-white px-2.5 py-1 rounded-full text-xs font-bold"
                  title="ストーリーポイントの合計"
                >
                  {{ formatStoryPoints(storyPointsOf(column)) }} pt
                </span>
                <span class="bg-white bg-opacity-35 text-black px-3 py-1.5 rounded-full text-sm font-bold">{{ countOf(column) }}</span>
              </div>
            </div>
          </div>

          <p v-if="filtering && !lanes.length" class="m-0 px-3 py-6 text-center text-white font-semibold">
            条件に一致する課題はありません
          </p>

          <SwimLane
            v-for="lane in lanes"
            :key="lane.key"
            :lane="lane"
            :collapsed="collapsedLanes.has(lane.key)"
            :columns="columns"
            :grid-style="columnGridStyle"
            @toggle-collapse="toggleLane(lane.key)"
            @drop="handleDrop"
            @delete-task="handleDeleteTodo"
            @add-task="openModal"
            @delete-epic="handleDeleteEpic"
            @rename-epic="handleRenameEpic"
            @open-task="openDetailModal"
          />

          <form class="flex items-center gap-2" @submit.prevent="handleCreateEpic">
            <input
              v-model="newEpicTitle"
              type="text"
              placeholder="新しいエピック"
              class="px-3 py-1.5 border-2 border-gray-300 rounded-full text-sm text-gray-900 bg-white focus:outline-none focus:border-indigo-500"
            />
            <button
              type="submit"
              :disabled="!newEpicTitle.trim()"
              class="px-3 py-1.5 bg-indigo-500 text-white border-0 rounded-full text-sm font-semibold cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              + エピックを追加
            </button>
          </form>
        </div>
      </div>
    </div>

    <div v-if="error" class="fixed bottom-8 right-8 bg-red-100 text-red-800 px-6 py-4 rounded-xl shadow-xl max-w-96 text-sm font-semibold border-l-4 border-red-500 animate-slideIn">
      {{ error }}
    </div>
    
    <!-- モーダル背景 -->
    <div 
      v-if="showModalForm"
      class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50"
      @click.self="closeModal"
    >
      <TaskForm 
        :statuses="statuses"
        :initial-status-id="modalInitialStatusId"
        :initial-category="modalInitialCategory"
        :initial-epic-id="modalInitialEpicId"
        :initial-space-id="modalInitialSpaceId"
        :epics="epics"
        :spaces="spaces"
        @submit="handleCreateTodoFromModal"
        @close="closeModal"
      />
    </div>

    <!-- タスク詳細のモーダル（全画面のページと同じ内容をその場で編集する） -->
    <div
      v-if="detailTask"
      class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50"
      @click.self="closeDetailModal"
    >
      <div class="bg-white rounded-2xl px-8 py-6 shadow-2xl mx-4 max-w-5xl w-full max-h-[90vh] overflow-y-auto">
        <IssueDetail
          :task="detailTask"
          :epics="epics"
          :spaces="spaces"
          :subtasks="subtasksOf(detailTask)"
          :parent-task="parentOf(detailTask)"
          :all-todos="todos"
          :statuses="statuses"
          @update="handleUpdateTodoFromModal"
          @subtasks-changed="fetchTodos"
          @links-changed="fetchTodos"
          @comments-changed="fetchTodos"
          @worklogs-changed="fetchTodos"
          @open="openDetailByKey"
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
              @click="closeDetailModal"
            >
              ×
            </button>
          </template>
        </IssueDetail>
      </div>
    </div>

    <!-- 削除の確認モーダル -->
    <ConfirmDialog
      v-if="confirmDialog"
      :title="confirmDialog.title"
      :message="confirmDialog.message"
      @confirm="closeConfirmDialog(true)"
      @cancel="closeConfirmDialog(false)"
    />
  </SpaceShell>
</template>

<script setup>
import { ref, computed, watch, onMounted, provide } from 'vue';
import TaskForm from './TaskForm.vue';
import IssueDetail from './IssueDetail.vue';
import SwimLane from './SwimLane.vue';
import SpaceShell from './SpaceShell.vue';
import ConfirmDialog from './ConfirmDialog.vue';
import FilterMenu from './FilterMenu.vue';
import UserAvatar from './UserAvatar.vue';
import { todosApi } from '../api/todos';
import { epicsApi } from '../api/epics';
import { spacesApi } from '../api/spaces';
import { statusesApi } from '../api/statuses';
import { STATUS_CATEGORIES, STATUS_CATEGORY_HEADER_CLASS, statusesOfSpace, firstStatusInCategory, isInColumn, isDoneCategory } from '../utils/statuses';
import { rememberLastBoard, issuePath } from '../utils/spaceRoutes';
import { formatStoryPoints, sumStoryPoints } from '../utils/storyPoints';
import { ISSUE_TYPE_OPTIONS } from '../utils/issueTypes';
import { NONE, parseFilters, withFilters, isFiltering, matchesFilters } from '../utils/boardFilters';
import { useCurrentUser } from '../composables/useCurrentUser';

const props = defineProps({
  // 表示するスペースのキー（URL の /spaces/:key/board）。null の場合は全スペースを表示する
  spaceKey: {
    type: String,
    default: null
  }
});

const todos = ref([]);
const epics = ref([]);
const spaces = ref([]);
// すべてのワークフローのステータス
const statuses = ref([]);
// スペース一覧を取得し終えたか（取得前に「見つかりません」を出さないため）
const spacesLoaded = ref(false);

// 表示中のスペース（null の場合は全スペース）
const currentSpace = computed(() =>
  props.spaceKey === null ? null : (spaces.value.find(s => s.Key === props.spaceKey) || null)
);
const selectedSpaceId = computed(() => (currentSpace.value ? currentSpace.value.ID : null));
const spaceNotFound = computed(() => props.spaceKey !== null && spacesLoaded.value && !currentSpace.value);
const newEpicTitle = ref('');
const error = ref('');
const showModalForm = ref(false);
// 「+ タスクを追加」した列のステータス（すべてのスペースの列なら 0 にしてカテゴリで決める）
const modalInitialStatusId = ref(0);
const modalInitialCategory = ref('ToDo');
const modalInitialEpicId = ref(0);
const route = useRoute();
// 詳細モーダルで表示中のタスク（null の場合は閉じている）。
// Jira と同じく URL の ?selectedIssue=KAN-12 で表し、リロードしても開いたままにする
const detailTask = computed(() => {
  const key = route.query.selectedIssue;
  return key ? (todos.value.find(todo => todo.Key === key) || null) : null;
});
// ヘッダーの「作成」で課題が作られたら増える（GlobalCreate と共有）
const issuesVersion = useState('issuesVersion', () => 0);
// 削除の確認モーダルの内容と、選んだ結果を返す関数（null の場合は閉じている）
const confirmDialog = ref(null);

// ボードの列。スペースを選んでいればそのワークフローのステータス（並び順）、すべてのスペースではカテゴリの 3 列。
// statusId が null の列はカテゴリでまとめた列
const columns = computed(() => {
  if (selectedSpaceId.value === null) {
    return STATUS_CATEGORIES.map(category => ({
      key: `category-${category.value}`,
      title: category.label,
      category: category.value,
      statusId: null
    }));
  }
  return statusesOfSpace(statuses.value, selectedSpaceId.value).map(status => ({
    key: `status-${status.ID}`,
    title: status.Name,
    category: status.Category,
    statusId: status.ID
  }));
});

const columnGridStyle = computed(() => ({ gridTemplateColumns: `repeat(${columns.value.length}, minmax(0, 1fr))` }));

// 列に落としたタスクのステータス。カテゴリの列なら、タスクのスペースの同じカテゴリの先頭のステータス
const statusIdForColumn = (column, spaceId) =>
  column.statusId ?? firstStatusInCategory(statusesOfSpace(statuses.value, spaceId), column.category)?.ID ?? 0;

// 選択中のスペース名（画面上部に表示する）
const selectedSpaceTitle = computed(() => {
  if (selectedSpaceId.value === null) {
    return 'すべてのスペース';
  }
  const space = spaces.value.find(s => s.ID === selectedSpaceId.value);
  return space ? space.Title : '';
});

// 選択中のスペースに属するエピック
const visibleEpics = computed(() =>
  selectedSpaceId.value === null
    ? epics.value
    : epics.value.filter(epic => epic.SpaceID === selectedSpaceId.value)
);

// スイムレーンの種類（Jira の「グループ化」）
const SWIMLANE_OPTIONS = [
  { value: 'epic', label: 'エピック' },
  { value: 'assignee', label: '担当者' },
  { value: 'subtask', label: 'サブタスク' },
  { value: 'none', label: 'なし' }
];
const swimlane = ref('epic');

// スイムレーンの種類はスペースごとにブラウザに覚える（保存できなくても動く）
const swimlaneStorageKey = () => `kanban.swimlane.${props.spaceKey ?? ''}`;
const loadSwimlane = () => {
  let saved = null;
  try {
    saved = localStorage.getItem(swimlaneStorageKey());
  } catch {
    // 読めなければ既定のエピックにする
  }
  swimlane.value = SWIMLANE_OPTIONS.some(option => option.value === saved) ? saved : 'epic';
};
watch(swimlane, (value) => {
  try {
    localStorage.setItem(swimlaneStorageKey(), value);
  } catch {
    // 保存できなくても、この画面の間は切り替えたままにする
  }
});
watch(() => props.spaceKey, loadSwimlane);

// 折りたたんでいるレーン（lanes の key）
const collapsedLanes = ref(new Set());
const toggleLane = (key) => {
  const next = new Set(collapsedLanes.value);
  if (next.has(key)) next.delete(key);
  else next.add(key);
  collapsedLanes.value = next;
};

// 選択中のスペースに属する課題（サブタスクを含む）
const spaceTodos = computed(() =>
  todos.value.filter(todo => selectedSpaceId.value === null || todo.SpaceID === selectedSpaceId.value)
);

// サブタスクを持つ親課題の ID（スイムレーンが「サブタスク」のときはレーンの見出しにする）
const parentIds = computed(() => new Set(spaceTodos.value.filter(todo => todo.ParentID).map(todo => todo.ParentID)));

// ボードのカードにする課題。サブタスクはカードにせず、親課題の詳細と件数で見せる。
// スイムレーンが「サブタスク」のときだけ、サブタスクをカードにし、その親課題はレーンの見出しにする
const visibleTodos = computed(() =>
  swimlane.value === 'subtask'
    ? spaceTodos.value.filter(todo => !parentIds.value.has(todo.ID))
    : spaceTodos.value.filter(todo => !todo.ParentID)
);

// クイックフィルターの条件（URL のクエリから読む）
const filters = computed(() => parseFilters(route.query));
const filtering = computed(() => isFiltering(filters.value));
const { user: currentUser } = useCurrentUser();

// 条件の一部を変えて URL に書き戻す。履歴を増やさないよう replace にする
const updateFilters = (changes) => {
  navigateTo({ query: withFilters(route.query, { ...filters.value, ...changes }) }, { replace: true });
};

const clearFilters = () => {
  searchText.value = '';
  navigateTo({ query: withFilters(route.query, parseFilters({})) }, { replace: true });
};

// 検索欄は入力のたびに URL を書き換えないよう、少し待ってから反映する
const searchText = ref(filters.value.text);
let searchTimer = null;
watch(searchText, (text) => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    if (text !== filters.value.text) updateFilters({ text });
  }, 250);
});
// 戻る・進むなどで URL が変わったら検索欄も合わせる
watch(() => filters.value.text, (text) => {
  if (text !== searchText.value) {
    clearTimeout(searchTimer);
    searchText.value = text;
  }
});

const toggleAssignee = (id) => {
  const { assignees } = filters.value;
  updateFilters({ assignees: assignees.includes(id) ? assignees.filter(a => a !== id) : [...assignees, id] });
};

const quickToggles = [
  { key: 'mine', label: '自分の課題のみ' },
  { key: 'recent', label: '最近更新したもの' }
];

// 絞り込みの候補。担当者・ラベルはボードの課題に出てくるもの、エピックはボードのエピック
const assigneeOptions = computed(() => {
  const users = new Map();
  let hasUnassigned = false;
  for (const todo of visibleTodos.value) {
    if (todo.Assignee) users.set(todo.Assignee.ID, todo.Assignee);
    else hasUnassigned = true;
  }
  const options = [...users.values()]
    .sort((a, b) => a.DisplayName.localeCompare(b.DisplayName))
    .map(user => ({ id: user.ID, user }));
  if (hasUnassigned) {
    options.push({ id: NONE, user: { DisplayName: '未割り当て', AvatarColor: '#A5ADBA' } });
  }
  return options;
});

const labelOptions = computed(() =>
  [...new Set(visibleTodos.value.flatMap(todo => todo.Labels || []))]
    .sort((a, b) => a.localeCompare(b))
    .map(label => ({ value: label, label }))
);

const epicOptions = computed(() => [
  ...visibleEpics.value.map(epic => ({ value: epic.ID, label: epic.Title })),
  { value: NONE, label: 'エピックなし' }
]);

// 絞り込んだあとのボードの課題
const filteredTodos = computed(() => {
  if (!filtering.value) return visibleTodos.value;
  const options = { currentUserId: currentUser.value?.ID ?? 0, now: Date.now() };
  return visibleTodos.value.filter(todo => matchesFilters(todo, filters.value, options));
});

const subtasksOf = (task) => todos.value.filter(todo => todo.ParentID === task.ID);
const parentOf = (task) => (task.ParentID ? todos.value.find(todo => todo.ID === task.ParentID) || null : null);

// 親課題ごとのサブタスクの件数（完了＝クローズ / 全体）。カードに出す
const subtaskProgress = computed(() => {
  const progress = {};
  for (const todo of todos.value) {
    if (!todo.ParentID) continue;
    const entry = progress[todo.ParentID] || (progress[todo.ParentID] = { done: 0, total: 0 });
    entry.total++;
    if (isDoneCategory(todo.StatusCategory)) entry.done++;
  }
  return progress;
});
provide('subtaskProgress', subtaskProgress);

// スイムレーンの定義。match はそのレーンに並ぶ課題かどうか
const laneDefinitions = computed(() => {
  switch (swimlane.value) {
    // 担当者ごと（表示名の順）。最後に未割り当てのレーンを置く
    case 'assignee': {
      const users = new Map();
      for (const todo of visibleTodos.value) {
        if (todo.Assignee) users.set(todo.Assignee.ID, todo.Assignee);
      }
      return [
        ...[...users.values()]
          .sort((a, b) => a.DisplayName.localeCompare(b.DisplayName))
          .map(user => ({ key: `assignee-${user.ID}`, kind: 'assignee', user, assigneeId: user.ID, match: todo => todo.Assignee?.ID === user.ID })),
        { key: 'assignee-0', kind: 'assignee', user: null, assigneeId: 0, match: todo => !todo.Assignee }
      ];
    }
    // サブタスクを持つ親課題ごと（課題の並び順）。最後にサブタスクのない課題のレーンを置く
    case 'subtask':
      return [
        ...spaceTodos.value
          .filter(todo => parentIds.value.has(todo.ID))
          .map(parent => ({ key: `parent-${parent.ID}`, kind: 'parent', parent, match: todo => todo.ParentID === parent.ID })),
        { key: 'others', kind: 'others', match: todo => !todo.ParentID }
      ];
    case 'none':
      return [{ key: 'none', kind: 'none', match: () => true }];
    // エピックごと。最後にエピック未割り当てのタスクのレーンを置く
    default: {
      const spaceTitleOf = (spaceId) => {
        if (selectedSpaceId.value !== null) return '';
        const space = spaces.value.find(s => s.ID === spaceId);
        return space ? space.Title : '';
      };
      return [
        ...visibleEpics.value.map(epic => ({
          key: `epic-${epic.ID}`,
          kind: 'epic',
          epic,
          spaceTitle: spaceTitleOf(epic.SpaceID),
          match: todo => todo.EpicID === epic.ID
        })),
        { key: 'epic-0', kind: 'epic', epic: null, match: todo => !visibleEpics.value.some(epic => epic.ID === todo.EpicID) }
      ];
    }
  }
});

// スイムレーン。tasks は表示する（絞り込んだ）課題、allTasks は絞り込む前の課題
const lanes = computed(() => {
  const all = laneDefinitions.value.map(lane => ({
    ...lane,
    tasks: filteredTodos.value.filter(lane.match),
    allTasks: visibleTodos.value.filter(lane.match)
  }));
  // 絞り込み中は、一致する課題のないレーンを出さない
  return filtering.value ? all.filter(lane => lane.tasks.length) : all;
});

// 列ごとのタスク件数（選択中のスペース内。絞り込み中は一致する課題だけ）
const countOf = (column) => filteredTodos.value.filter(todo => isInColumn(todo, column)).length;

// 列ごとのストーリーポイントの合計（選択中のスペース内。未設定のタスクは数えない）
const storyPointsOf = (column) => sumStoryPoints(filteredTodos.value.filter(todo => isInColumn(todo, column)));
const hasStoryPoints = (column) => filteredTodos.value.some(todo => isInColumn(todo, column) && todo.StoryPoints != null);

// タスク作成フォームの初期スペース（エピックのレーンから開いたときはそのエピックのスペース）
const modalInitialSpaceId = computed(() => {
  const epic = epics.value.find(e => e.ID === modalInitialEpicId.value);
  return epic ? epic.SpaceID : (selectedSpaceId.value ?? 0);
});

// 確認モーダルを開き、削除を選んだら true、キャンセルしたら false で解決する
const askConfirm = (title, message) => new Promise((resolve) => {
  confirmDialog.value = { title, message, resolve };
});

// 確認モーダルを閉じる
const closeConfirmDialog = (confirmed) => {
  const dialog = confirmDialog.value;
  if (!dialog) return;
  confirmDialog.value = null;
  dialog.resolve(confirmed);
};

// タスク一覧を取得
const fetchTodos = async () => {
  try {
    error.value = '';
    todos.value = await todosApi.fetchAll();
  } catch (err) {
    error.value = 'タスクの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch todos:', err);
  }
};

// 操作失敗後に画面をサーバーの状態へ合わせる（エラー表示は残す）
const resyncTodos = async () => {
  try {
    todos.value = await todosApi.fetchAll();
  } catch (err) {
    console.error('Failed to resync todos:', err);
  }
};

// エピック一覧を取得
const fetchEpics = async () => {
  try {
    error.value = '';
    epics.value = await epicsApi.fetchAll();
  } catch (err) {
    error.value = 'エピックの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch epics:', err);
  }
};

// スペース一覧を取得
const fetchSpaces = async () => {
  try {
    error.value = '';
    spaces.value = await spacesApi.fetchAll();
    spacesLoaded.value = true;
  } catch (err) {
    error.value = 'スペースの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch spaces:', err);
  }
};

// ワークフローのステータスを取得
const fetchStatuses = async () => {
  try {
    statuses.value = await statusesApi.fetchAll();
  } catch (err) {
    error.value = 'ステータスの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch statuses:', err);
  }
};

// サイドバーでスペースを作成・削除したら取り直す（削除すると属するエピック・タスクのスペースも変わる。作るとワークフローができる）
const handleSpacesChanged = async () => {
  await Promise.all([fetchSpaces(), fetchEpics(), fetchTodos(), fetchStatuses()]);
};

// ヘッダーの「作成」で作られた課題をボードに出す
watch(issuesVersion, () => {
  fetchTodos();
});

// `/` を開いたときに戻れるよう、表示したボードを覚える（存在しないスペースは覚えない）
watch([currentSpace, () => props.spaceKey, spacesLoaded], () => {
  if (!spacesLoaded.value || spaceNotFound.value) return;
  rememberLastBoard(currentSpace.value);
});

// タスクまたはエピックを別のスペースへ移動する
const handleMoveToSpace = async ({ type, id, spaceId }) => {
  try {
    error.value = '';
    if (type === 'epic') {
      const epic = epics.value.find(e => e.ID === id);
      if (!epic) {
        throw new Error('エピックが見つかりません');
      }
      if (epic.SpaceID === spaceId) {
        return;
      }
      // 子タスクもエピックと一緒に移動する
      await epicsApi.update(id, { title: epic.Title, spaceId });
      await Promise.all([fetchEpics(), fetchTodos()]);
      return;
    }

    const task = todos.value.find(t => t.ID === id);
    if (!task) {
      throw new Error('タスクが見つかりません');
    }
    if (task.SpaceID === spaceId) {
      return;
    }
    // エピックは移動元のスペースに属するので、紐付けを外す。ステータスは bff が移動先のワークフローの同じ名前かカテゴリに合わせる
    await todosApi.update(id, {
      title: task.Title,
      description: task.Description,
      deadline: task.Deadline,
      statusId: task.StatusID,
      issueType: task.IssueType,
      priority: task.Priority,
      epicId: 0,
      spaceId
    });
    await fetchTodos();
  } catch (err) {
    error.value = 'スペースの移動に失敗しました: ' + err.message;
    console.error('Failed to move to space:', err);
    await Promise.all([resyncTodos(), fetchEpics()]);
  }
};

// エピックを作成（スペース選択中はそのスペースに作る）
const handleCreateEpic = async () => {
  const title = newEpicTitle.value.trim();
  if (!title) return;

  try {
    error.value = '';
    await epicsApi.create({ title, spaceId: selectedSpaceId.value ?? 0 });
    newEpicTitle.value = '';
    await fetchEpics();
  } catch (err) {
    error.value = 'エピックの作成に失敗しました: ' + err.message;
    console.error('Failed to create epic:', err);
  }
};

// エピックの名前を変更
const handleRenameEpic = async ({ id, title }) => {
  try {
    error.value = '';
    const epic = epics.value.find(e => e.ID === id);
    if (!epic) {
      throw new Error('エピックが見つかりません');
    }
    await epicsApi.update(id, { title, spaceId: epic.SpaceID });
    await fetchEpics();
  } catch (err) {
    error.value = 'エピックの名前の変更に失敗しました: ' + err.message;
    console.error('Failed to rename epic:', err);
  }
};

// エピックを削除（子タスクは未割り当てに戻る）
const handleDeleteEpic = async (epicId) => {
  const epic = epics.value.find(e => e.ID === epicId);
  const confirmed = await askConfirm(
    'エピックを削除しますか？',
    `「${epic ? epic.Title : ''}」を削除します。\n子タスクは削除されず、エピック未割り当てになります。`
  );
  if (!confirmed) {
    return;
  }

  try {
    error.value = '';
    await epicsApi.delete(epicId);
    await Promise.all([fetchEpics(), fetchTodos()]);
  } catch (err) {
    error.value = 'エピックの削除に失敗しました: ' + err.message;
    console.error('Failed to delete epic:', err);
  }
};

// タスクを作成
const handleCreateTodo = async (formData) => {
  try {
    error.value = '';
    await todosApi.create({
      title: formData.title,
      description: formData.description,
      deadline: formData.deadline,
      // 0 なら bff がスペースの To Do の先頭にする
      statusId: formData.statusId,
      issueType: formData.issueType,
      priority: formData.priority,
      labels: formData.labels,
      storyPoints: formData.storyPoints,
      epicId: formData.epicId,
      spaceId: formData.spaceId
    });
    await fetchTodos(); // リストを再取得
  } catch (err) {
    error.value = 'タスクの作成に失敗しました: ' + err.message;
    console.error('Failed to create todo:', err);
  }
};

// モーダルからのタスク作成
const handleCreateTodoFromModal = async (formData) => {
  await handleCreateTodo(formData);
  closeModal();
};

// モーダルを開く（開いた列のステータスと、エピックのレーンならそのエピックを初期値にする）
const openModal = ({ column, lane }) => {
  modalInitialStatusId.value = column.statusId ?? 0;
  modalInitialCategory.value = column.category;
  modalInitialEpicId.value = lane.kind === 'epic' && lane.epic ? lane.epic.ID : 0;
  showModalForm.value = true;
};

// モーダルを閉じる
const closeModal = () => {
  showModalForm.value = false;
  modalInitialStatusId.value = 0;
  modalInitialCategory.value = 'ToDo';
  modalInitialEpicId.value = 0;
};

// タスク詳細のモーダルを開く
const openDetailModal = (task) => {
  navigateTo({ query: { ...route.query, selectedIssue: task.Key } });
};

// 詳細モーダルで別の課題（親課題・子課題）を開く
const openDetailByKey = (key) => {
  navigateTo({ query: { ...route.query, selectedIssue: key } });
};

// タスク詳細のモーダルを閉じる
const closeDetailModal = () => {
  const { selectedIssue, ...query } = route.query;
  navigateTo({ query }, { replace: true });
};

// タスク詳細のモーダルでインライン編集した内容を保存する（モーダルは開いたまま）
const handleUpdateTodoFromModal = async (payload) => {
  const task = detailTask.value;
  try {
    error.value = '';
    const { todo } = await todosApi.update(task.ID, payload);
    // スペースを変えると課題キーが変わるので、モーダルが閉じないよう先に一覧と URL を新しいキーにする
    todos.value = todos.value.map(t => (t.ID === todo.ID ? todo : t));
    if (todo.Key !== task.Key) {
      await navigateTo({ query: { ...route.query, selectedIssue: todo.Key } }, { replace: true });
    }
    await fetchTodos(); // リストを再取得
  } catch (err) {
    error.value = 'タスクの更新に失敗しました: ' + err.message;
    console.error('Failed to update todo:', err);
    await resyncTodos();
  }
};

// ドラッグ&ドロップでステータスと並び順を更新。別のレーンへ落としたら、そのレーンのエピック・担当者に変える
// beforeId はドロップ先の直後のタスクの ID（0 は末尾、null は並び順を変えない）
const handleDrop = async ({ taskId, column, lane, beforeId }) => {
  try {
    error.value = '';
    const task = todos.value.find(t => t.ID === taskId);
    if (!task) {
      throw new Error('タスクが見つかりません');
    }

    // 「エピックなし」のレーンには、選択中のスペースで表示していないエピックのタスクも並ぶので、そのまま残す
    const isSameLane = lane.match(task);
    // サブタスクの親課題は変えられない（Jira と同じく、ドラッグでは別の親課題へ移せない）
    if (!isSameLane && (lane.kind === 'parent' || lane.kind === 'others')) {
      error.value = 'サブタスクの親課題はドラッグでは変えられません';
      return;
    }
    const newEpicId = !isSameLane && lane.kind === 'epic' ? (lane.epic ? lane.epic.ID : 0) : task.EpicID;
    // タスクはエピックと同じスペースに置く
    const epic = epics.value.find(e => e.ID === newEpicId);
    const newSpaceId = epic ? epic.SpaceID : task.SpaceID;
    // 同じ列に落とした場合は、ステータスを変えない（カテゴリの列で同じカテゴリの別のステータスに変わらないように）
    const newStatusId = isInColumn(task, column) ? task.StatusID : statusIdForColumn(column, newSpaceId);

    // ステータスもレーンも並び順も変わらない場合は何もしない
    if (task.StatusID === newStatusId && isSameLane && beforeId === null) {
      return;
    }

    if (task.StatusID !== newStatusId || !isSameLane) {
      await todosApi.update(taskId, {
        title: task.Title,
        description: task.Description,
        deadline: task.Deadline,
        statusId: newStatusId,
        issueType: task.IssueType,
        priority: task.Priority,
        epicId: newEpicId,
        spaceId: newSpaceId,
        // 担当者のレーンへ移したときだけ担当者を変える（省略すると今のまま）
        ...(!isSameLane && lane.kind === 'assignee' ? { assigneeId: lane.assigneeId } : {})
      });
    }

    if (beforeId !== null) {
      await todosApi.move(taskId, resolveBeforeId({ taskId, column, lane, beforeId }));
    }

    await fetchTodos(); // リストを再取得
  } catch (err) {
    error.value = 'タスクの更新に失敗しました: ' + err.message;
    console.error('Failed to update todo:', err);
    await resyncTodos();
  }
};

// 絞り込み中に列の末尾へ落としたときは、全体の末尾ではなく、見えている最後の課題の直後に置く。
// 表示していない課題との前後を崩さないため（見えている課題の直前に落としたときは、その ID のままでよい）
const resolveBeforeId = ({ taskId, column, lane, beforeId }) => {
  if (beforeId !== 0 || !filtering.value) return beforeId;
  const cell = lane.allTasks.filter(todo => todo.ID !== taskId && isInColumn(todo, column));
  const shownIds = new Set(lane.tasks.map(todo => todo.ID));
  const lastShown = cell.findLastIndex(todo => shownIds.has(todo.ID));
  if (lastShown === -1) return 0;
  return cell[lastShown + 1]?.ID ?? 0;
};

// タスクを削除
const handleDeleteTodo = async (taskId) => {
  const task = todos.value.find(t => t.ID === taskId);
  // サブタスクも一緒に削除されることを、件数とあわせて伝える
  const subtaskCount = todos.value.filter(t => t.ParentID === taskId).length;
  const confirmed = await askConfirm(
    'タスクを削除しますか？',
    `「${task ? task.Title : ''}」を削除します。この操作は取り消せません。` +
      (subtaskCount ? `\nサブタスク ${subtaskCount} 件も一緒に削除されます。` : '')
  );
  if (!confirmed) {
    return;
  }
  
  try {
    error.value = '';
    await todosApi.delete(taskId);
    await fetchTodos(); // リストを再取得
  } catch (err) {
    error.value = 'タスクの削除に失敗しました: ' + err.message;
    console.error('Failed to delete todo:', err);
    await resyncTodos();
  }
};

// コンポーネントマウント時にタスクを取得
onMounted(() => {
  loadSwimlane();
  fetchTodos();
  fetchEpics();
  fetchSpaces();
  fetchStatuses();
});
</script>
