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
        <div class="flex flex-wrap items-center gap-3 max-w-6xl">
          <h2 class="m-0 text-2xl font-bold text-white">バックログ</h2>
          <template v-if="selectedIds.size > 1">
            <span class="ml-auto text-sm font-semibold text-white">{{ selectedIds.size }} 件を選択中</span>
            <button
              type="button"
              class="px-2 py-1 border-0 rounded bg-white/90 text-sm text-gray-700 cursor-pointer hover:bg-white"
              @click="clearSelection"
            >
              選択を解除
            </button>
          </template>
          <span v-else class="ml-auto text-xs text-white/80">ドラッグで順位・スプリントを変更（Shift / ⌘ クリックで複数選択）</span>
        </div>

        <!-- スプリント（開始中・未開始）とバックログの枠。課題はドラッグで枠の間を出し入れできる -->
        <section
          v-for="section in sections"
          :key="section.key"
          :class="[
            'mt-4 max-w-6xl bg-white rounded-2xl shadow-xl overflow-hidden border-2',
            dropTarget?.key === section.key ? 'border-blue-500' : 'border-transparent'
          ]"
        >
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1 px-5 py-3 border-b border-gray-200">
            <h3 class="m-0 text-base font-bold text-gray-900">{{ section.sprint ? section.sprint.Name : 'バックログ' }}</h3>
            <span v-if="section.sprint?.State === SPRINT_ACTIVE" class="px-2 py-0.5 rounded bg-blue-100 text-xs font-semibold text-blue-800">開始中</span>
            <span v-if="section.sprint && sprintPeriodLabel(section.sprint)" class="text-sm text-gray-500">{{ sprintPeriodLabel(section.sprint) }}</span>
            <span v-if="section.sprint?.State === SPRINT_ACTIVE" class="text-sm font-semibold text-gray-700">{{ remainingDaysLabel(section.sprint) }}</span>
            <span class="text-sm text-gray-500">{{ section.rows.length }} 件の課題</span>
            <span
              v-if="section.rows.some(todo => todo.StoryPoints != null)"
              class="px-2 py-0.5 rounded-full bg-gray-100 text-xs font-semibold text-gray-700"
              title="ストーリーポイントの合計"
            >
              {{ formatStoryPoints(sumStoryPoints(section.rows)) }} pt
            </span>
            <div class="ml-auto flex items-center gap-2">
              <template v-if="section.sprint?.State === SPRINT_ACTIVE">
                <button type="button" :class="secondaryButton" @click="openSprintDialog(section.sprint, 'edit')">編集</button>
                <button type="button" :class="primaryButton" @click="completing = section.sprint">スプリントを完了</button>
              </template>
              <template v-else-if="section.sprint">
                <button type="button" :class="secondaryButton" @click="openSprintDialog(section.sprint, 'edit')">編集</button>
                <button type="button" :class="secondaryButton" @click="deleting = section.sprint">削除</button>
                <button
                  type="button"
                  :class="primaryButton"
                  :disabled="!!activeSprint"
                  :title="activeSprint ? '開始中のスプリントを完了してから開始できます' : ''"
                  @click="openSprintDialog(section.sprint, 'start')"
                >
                  スプリントを開始
                </button>
              </template>
              <button v-else type="button" :class="secondaryButton" @click="handleCreateSprint">スプリントを作成</button>
            </div>
            <p v-if="section.sprint?.Goal" class="w-full m-0 text-sm text-gray-600 whitespace-pre-line">{{ section.sprint.Goal }}</p>
          </div>

          <!-- 課題の行。ドラッグで順位を変える（選択中の行はまとめて動かす） -->
          <ol
            class="m-0 p-0 list-none min-h-10"
            :aria-label="`${section.sprint ? section.sprint.Name : 'バックログ'}の課題`"
            @dragover="handleDragOver($event, section)"
            @dragleave="handleDragLeave($event, section)"
            @drop="handleDrop($event, section)"
          >
            <template v-for="(row, index) in section.rows" :key="row.ID">
              <li v-if="dropTarget?.key === section.key && dropTarget.index === index" class="h-0.5 bg-blue-500" aria-hidden="true"></li>
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
                <span :class="['shrink-0 w-20 font-mono text-xs font-semibold', row.Done ? 'text-gray-400 line-through' : 'text-gray-500']">{{ row.Key }}</span>
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
            <li v-if="dropTarget?.key === section.key && dropTarget.index === section.rows.length" class="h-0.5 bg-blue-500" aria-hidden="true"></li>
            <li v-if="!section.rows.length" class="px-5 py-4 text-center text-sm text-gray-500">
              {{ section.sprint ? 'バックログから課題をドラッグして、このスプリントに入れます' : '完了していない課題はありません' }}
            </li>
          </ol>

          <!-- その場で課題を作成する（Jira の「+ 課題を作成」）。続けて入力できるよう、作成後も開いたままにする -->
          <form v-if="creatingKey === section.key" class="flex items-center gap-2 px-5 py-2" @submit.prevent="handleCreate(section)">
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
            @click="openCreate(section)"
          >
            + 課題を作成
          </button>
        </section>
      </template>
    </div>

    <SprintDialog
      v-if="sprintDialog"
      :sprint="sprintDialog.sprint"
      :mode="sprintDialog.mode"
      :issue-count="issuesOfSprint(sprintDialog.sprint.ID).length"
      :busy="saving"
      @submit="handleSprintDialogSubmit"
      @cancel="sprintDialog = null"
    />
    <CompleteSprintDialog
      v-if="completing"
      :sprint="completing"
      :done-count="issuesOfSprint(completing.ID).filter(todo => todo.Done).length"
      :open-count="issuesOfSprint(completing.ID).filter(todo => !todo.Done).length"
      :targets="futureSprints"
      :busy="saving"
      @submit="handleCompleteSprint"
      @cancel="completing = null"
    />
    <ConfirmDialog
      v-if="deleting"
      title="スプリントを削除しますか？"
      :message="`「${deleting.Name}」を削除します。\nこのスプリントの課題は削除されず、バックログに戻ります。`"
      @confirm="handleDeleteSprint"
      @cancel="deleting = null"
    />

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
          @fix-versions-changed="fetchTodos"
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
import ConfirmDialog from '~/components/ConfirmDialog.vue';
import SprintDialog from '~/components/SprintDialog.vue';
import CompleteSprintDialog from '~/components/CompleteSprintDialog.vue';
import { todosApi } from '~/api/todos';
import { sprintsApi } from '~/api/sprints';
import { epicsApi } from '~/api/epics';
import { spacesApi } from '~/api/spaces';
import { statusesApi } from '~/api/statuses';
import { STATUS_CATEGORY_BADGE_CLASS, isDoneCategory } from '~/utils/statuses';
import { ISSUE_TYPE_OPTIONS, DEFAULT_ISSUE_TYPE } from '~/utils/issueTypes';
import { formatStoryPoints, sumStoryPoints } from '~/utils/storyPoints';
import { issuePath } from '~/utils/spaceRoutes';
import { SPRINT_ACTIVE, SPRINT_FUTURE, activeSprintOf, remainingDaysLabel, sprintPeriodLabel } from '~/utils/sprints';

const route = useRoute();
const spaceKey = computed(() => String(route.params.key));

const todos = ref([]);
const epics = ref([]);
const spaces = ref([]);
const statuses = ref([]);
const sprints = ref([]);
const loading = ref(true);
const error = ref('');
// ヘッダーの「作成」で課題が作られたら増える（GlobalCreate と共有）
const issuesVersion = useState('issuesVersion', () => 0);

const space = computed(() => spaces.value.find(s => s.Key === spaceKey.value) || null);

useHead(() => ({ title: space.value ? `バックログ - ${space.value.Title} - Kanban` : 'Kanban' }));

const primaryButton = 'px-3 py-1.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed';
const secondaryButton = 'px-3 py-1.5 bg-gray-100 text-gray-700 border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-gray-200';

// スペースの開始中のスプリントと、未開始のスプリント（作成順）
const activeSprint = computed(() => (space.value ? activeSprintOf(sprints.value, space.value.ID) : null));
const futureSprints = computed(() =>
  space.value ? sprints.value.filter(sprint => sprint.SpaceID === space.value.ID && sprint.State === SPRINT_FUTURE) : []
);
const openSprintIds = computed(() => new Set([activeSprint.value, ...futureSprints.value].filter(Boolean).map(sprint => sprint.ID)));

// スペースの課題（サブタスクは親課題の詳細で見せる）。順位（GET /todos の並び順）の順
const spaceIssues = computed(() =>
  space.value ? todos.value.filter(todo => todo.SpaceID === space.value.ID && !todo.ParentID) : []
);

// 課題が入っている枠のスプリント。完了したスプリントに残った課題（完了後に開き直したもの）はバックログに出す
const sprintIdOfRow = (todo) => (openSprintIds.value.has(todo.SprintID) ? todo.SprintID : 0);

const issuesOfSprint = (sprintId) => spaceIssues.value.filter(todo => todo.SprintID === sprintId);

// 枠: 開始中のスプリント → 未開始のスプリント → バックログ（完了していない課題）。スプリントには完了した課題も出す
const sections = computed(() => [
  ...[activeSprint.value, ...futureSprints.value].filter(Boolean).map(sprint => ({
    key: `sprint-${sprint.ID}`,
    sprint,
    sprintId: sprint.ID,
    rows: issuesOfSprint(sprint.ID)
  })),
  {
    key: 'backlog',
    sprint: null,
    sprintId: 0,
    rows: spaceIssues.value.filter(todo => sprintIdOfRow(todo) === 0 && !isDoneCategory(todo.StatusCategory))
  }
]);

// すべての枠の行（上から順）。Shift クリックの範囲選択に使う
const rows = computed(() => sections.value.flatMap(section => section.rows));

const epicOf = (todo) => (todo.EpicID ? epics.value.find(epic => epic.ID === todo.EpicID) || null : null);

const fetchTodos = async () => {
  try {
    todos.value = await todosApi.fetchAll();
  } catch (err) {
    error.value = '課題の取得に失敗しました: ' + err.message;
    console.error('Failed to fetch todos:', err);
  }
};

const fetchSprints = async () => {
  try {
    sprints.value = await sprintsApi.fetchAll();
  } catch (err) {
    error.value = 'スプリントの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch sprints:', err);
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
  await Promise.all([fetchTodos(), fetchSprints()]);
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

// ドラッグ&ドロップで順位とスプリントを変える。選択中の行をドラッグしたら、選択中の行をまとめて（順位の順のまま）動かす
const draggingIds = ref(new Set());
// 挿入する枠（sections の key）と位置（その枠の rows のインデックス）。null の場合はドラッグ中でない
const dropTarget = ref(null);
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
  dropTarget.value = null;
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

const isBacklogDrag = (event) => Array.from(event.dataTransfer.types).includes(BACKLOG_DRAG_TYPE);

const handleDragOver = (event, section) => {
  if (!isBacklogDrag(event)) return;
  event.preventDefault();
  event.dataTransfer.dropEffect = 'move';
  dropTarget.value = { key: section.key, index: getDropIndex(event) };
};

const handleDragLeave = (event, section) => {
  if (event.currentTarget.contains(event.relatedTarget)) return;
  if (dropTarget.value?.key === section.key) dropTarget.value = null;
};

const handleDrop = async (event, section) => {
  if (!isBacklogDrag(event)) return;
  event.preventDefault();
  const index = dropTarget.value?.key === section.key ? dropTarget.value.index : getDropIndex(event);
  const moving = new Set(event.dataTransfer.getData(BACKLOG_DRAG_TYPE).split(',').map(Number));
  draggingIds.value = new Set();
  dropTarget.value = null;

  // 動かす課題（順位の順）。ほかの枠から持ってきた課題はスプリントを変える
  const movingRows = rows.value.filter(todo => moving.has(todo.ID));
  if (!movingRows.length) return;
  const toChange = movingRows.filter(todo => sprintIdOfRow(todo) !== section.sprintId);

  // 動かした後の枠の行の並び。スプリントも並びも変わらなければ何もしない
  const list = section.rows;
  const rest = list.filter(todo => !moving.has(todo.ID));
  const insertAt = list.slice(0, index).filter(todo => !moving.has(todo.ID)).length;
  const next = [...rest.slice(0, insertAt), ...movingRows, ...rest.slice(insertAt)];
  if (!toChange.length && next.length === list.length && next.every((todo, i) => todo.ID === list[i].ID)) return;

  // 直後の行の前へ入れる。末尾なら、最後の行の（全課題の並びで）次の課題の前へ入れ、ほかの課題の順位を崩さない。
  // 空の枠なら順位は変えない
  let beforeId = rest[insertAt]?.ID ?? null;
  if (beforeId === null && rest.length) {
    const last = rest[rest.length - 1];
    const others = todos.value.filter(todo => !moving.has(todo.ID));
    beforeId = others[others.indexOf(last) + 1]?.ID ?? 0;
  }

  try {
    error.value = '';
    for (const todo of toChange) {
      await sprintsApi.setTodoSprint(todo.ID, section.sprintId);
    }
    if (beforeId !== null) {
      for (const todo of movingRows) {
        await todosApi.move(todo.ID, beforeId);
      }
    }
  } catch (err) {
    error.value = '課題の移動に失敗しました: ' + err.message;
    console.error('Failed to move todos:', err);
  }
  await fetchTodos();
};

// その場で課題を作成する（スペースのワークフローの To Do の先頭。順位は末尾）。スプリントの枠で作ったらそのスプリントに入れる
const creatingKey = ref(null);
const newTitle = ref('');
const newIssueType = ref(DEFAULT_ISSUE_TYPE);
const newTitleInput = ref(null);
const saving = ref(false);

const focusNewTitle = async () => {
  await nextTick();
  const input = Array.isArray(newTitleInput.value) ? newTitleInput.value[0] : newTitleInput.value;
  input?.focus();
};

const openCreate = (section) => {
  creatingKey.value = section.key;
  focusNewTitle();
};

const closeCreate = () => {
  creatingKey.value = null;
  newTitle.value = '';
};

const handleCreate = async (section) => {
  const title = newTitle.value.trim();
  if (!title || saving.value) return;
  saving.value = true;
  try {
    error.value = '';
    const { todo } = await todosApi.create({ title, issueType: newIssueType.value, spaceId: space.value.ID });
    if (section.sprintId) {
      await sprintsApi.setTodoSprint(todo.ID, section.sprintId);
    }
    newTitle.value = '';
    await fetchTodos();
  } catch (err) {
    error.value = '課題の作成に失敗しました: ' + err.message;
    console.error('Failed to create todo:', err);
  } finally {
    saving.value = false;
    focusNewTitle();
  }
};

// スプリントの作成・開始・編集・完了・削除
const sprintDialog = ref(null);
const completing = ref(null);
const deleting = ref(null);

// 失敗したらメッセージを出す。どちらの場合もスプリントと課題を取り直す
const runSprintAction = async (label, action) => {
  saving.value = true;
  try {
    error.value = '';
    await action();
    return true;
  } catch (err) {
    error.value = `${label}に失敗しました: ${err.message}`;
    console.error(`Failed: ${label}`, err);
    return false;
  } finally {
    saving.value = false;
    await Promise.all([fetchSprints(), fetchTodos()]);
  }
};

const handleCreateSprint = () => runSprintAction('スプリントの作成', () => sprintsApi.create(space.value.ID));

const openSprintDialog = (sprint, mode) => {
  sprintDialog.value = { sprint, mode };
};

const handleSprintDialogSubmit = async (fields) => {
  const { sprint, mode } = sprintDialog.value;
  const ok = mode === 'start'
    ? await runSprintAction('スプリントの開始', () => sprintsApi.start(sprint.ID, fields))
    : await runSprintAction('スプリントの更新', () => sprintsApi.update(sprint.ID, fields));
  if (ok) sprintDialog.value = null;
};

const handleCompleteSprint = async (moveTo) => {
  if (await runSprintAction('スプリントの完了', () => sprintsApi.complete(completing.value.ID, moveTo))) {
    completing.value = null;
  }
};

const handleDeleteSprint = async () => {
  const sprint = deleting.value;
  deleting.value = null;
  await runSprintAction('スプリントの削除', () => sprintsApi.delete(sprint.ID));
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
