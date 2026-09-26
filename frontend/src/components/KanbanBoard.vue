<template>
  <div class="flex min-h-screen">
    <SpaceSidebar
      v-if="sidebarOpen"
      :spaces="spaces"
      :todos="todos"
      :selected-space-id="selectedSpaceId"
      @select="selectSpace"
      @create="handleCreateSpace"
      @delete="handleDeleteSpace"
      @move="handleMoveToSpace"
    />

    <div class="flex-1 min-w-0 px-8 py-8">
      <h2 class="m-0 text-2xl font-bold text-white">{{ selectedSpaceTitle }}</h2>

      <div class="overflow-x-auto pb-6 mt-6">
        <div class="min-w-5xl flex flex-col gap-4">
          <div class="grid grid-cols-4 gap-4 px-3">
            <div
              v-for="column in columns"
              :key="column.status"
              :class="[
                'flex justify-between items-center px-5 py-4 rounded-lg text-white shadow-md',
                statusHeaderClass[column.status]
              ]"
            >
              <h3 class="m-0 text-xl font-bold tracking-wide">{{ column.title }}</h3>
              <span class="bg-white bg-opacity-35 text-black px-3 py-1.5 rounded-full text-sm font-bold">{{ countByStatus(column.status) }}</span>
            </div>
          </div>

          <EpicLane
            v-for="lane in lanes"
            :key="lane.epic ? lane.epic.ID : 0"
            :epic="lane.epic"
            :tasks="lane.tasks"
            :epics="epics"
            :columns="columns"
            :space-title="lane.spaceTitle"
            @drop="handleDrop"
            @change-epic="handleChangeEpic"
            @delete-task="handleDeleteTodo"
            @add-task="openModal"
            @delete-epic="handleDeleteEpic"
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
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      @click.self="closeModal"
    >
      <TaskForm 
        :initial-status="modalInitialStatus"
        :initial-epic-id="modalInitialEpicId"
        :initial-space-id="modalInitialSpaceId"
        :epics="epics"
        :spaces="spaces"
        @submit="handleCreateTodoFromModal"
        @close="closeModal"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import TaskForm from './TaskForm.vue';
import EpicLane from './EpicLane.vue';
import SpaceSidebar from './SpaceSidebar.vue';
import { todosApi } from '../api/todos';
import { epicsApi } from '../api/epics';
import { spacesApi } from '../api/spaces';

const todos = ref([]);
const epics = ref([]);
const spaces = ref([]);
// null の場合は全スペースを表示
const selectedSpaceId = ref(null);
// サイドバー（スペース一覧）の開閉状態。ヘッダーのハンバーガーメニューで切り替える
const sidebarOpen = useState('sidebarOpen', () => false);
const newEpicTitle = ref('');
const error = ref('');
const showModalForm = ref(false);
const modalInitialStatus = ref('Open');
const modalInitialEpicId = ref(0);

const columns = [
  { status: 'Open', title: 'オープン' },
  { status: 'InProgress', title: '進行中' },
  { status: 'Waiting', title: '待ち状況' },
  { status: 'Closed', title: 'クローズ' }
];

const statusHeaderClass = {
  'Open': 'bg-gradient-to-r from-gray-400 to-gray-600',
  'InProgress': 'bg-gradient-to-r from-amber-400 to-amber-600',
  'Waiting': 'bg-gradient-to-r from-purple-500 to-purple-700',
  'Closed': 'bg-gradient-to-r from-emerald-500 to-emerald-700'
};

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

// 選択中のスペースに属するタスク
const visibleTodos = computed(() =>
  selectedSpaceId.value === null
    ? todos.value
    : todos.value.filter(todo => todo.SpaceID === selectedSpaceId.value)
);

// エピックごとのレーン。最後にエピック未割り当てのタスクのレーンを置く
const lanes = computed(() => {
  const spaceTitleOf = (spaceId) => {
    if (selectedSpaceId.value !== null) return '';
    const space = spaces.value.find(s => s.ID === spaceId);
    return space ? space.Title : '';
  };
  return [
    ...visibleEpics.value.map(epic => ({
      epic,
      tasks: visibleTodos.value.filter(todo => todo.EpicID === epic.ID),
      spaceTitle: spaceTitleOf(epic.SpaceID)
    })),
    {
      epic: null,
      tasks: visibleTodos.value.filter(todo => !visibleEpics.value.some(epic => epic.ID === todo.EpicID)),
      spaceTitle: ''
    }
  ];
});

// ステータスごとのタスク件数（選択中のスペース内）
const countByStatus = (status) => visibleTodos.value.filter(todo => todo.Status === status).length;

// タスク作成フォームの初期スペース（エピックのレーンから開いたときはそのエピックのスペース）
const modalInitialSpaceId = computed(() => {
  const epic = epics.value.find(e => e.ID === modalInitialEpicId.value);
  return epic ? epic.SpaceID : (selectedSpaceId.value ?? 0);
});

// スペースを切り替える
const selectSpace = (spaceId) => {
  selectedSpaceId.value = spaceId;
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
  } catch (err) {
    error.value = 'スペースの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch spaces:', err);
  }
};

// スペースを作成
const handleCreateSpace = async (title) => {
  try {
    error.value = '';
    await spacesApi.create({ title });
    await fetchSpaces();
  } catch (err) {
    error.value = 'スペースの作成に失敗しました: ' + err.message;
    console.error('Failed to create space:', err);
  }
};

// スペースを削除（エピックとタスクは削除されず、スペース未割り当てになる）
const handleDeleteSpace = async (spaceId) => {
  if (!confirm('このスペースを削除してもよろしいですか？\nエピックとタスクは削除されず、スペース未割り当てになります。')) {
    return;
  }

  try {
    error.value = '';
    await spacesApi.delete(spaceId);
    if (selectedSpaceId.value === spaceId) {
      selectSpace(null);
    }
    await Promise.all([fetchSpaces(), fetchEpics(), fetchTodos()]);
  } catch (err) {
    error.value = 'スペースの削除に失敗しました: ' + err.message;
    console.error('Failed to delete space:', err);
  }
};

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
    // エピックは移動元のスペースに属するので、紐付けを外す
    await todosApi.update(id, {
      title: task.Title,
      description: task.Description,
      done: task.Done,
      deadline: task.Deadline,
      status: task.Status,
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

// エピックを削除（子タスクは未割り当てに戻る）
const handleDeleteEpic = async (epicId) => {
  if (!confirm('このエピックを削除してもよろしいですか？\n子タスクは削除されず、エピック未割り当てになります。')) {
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
      status: formData.status,
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

// モーダルを開く（開いたレーンのエピックを初期値にする）
const openModal = ({ status, epicId }) => {
  modalInitialStatus.value = status;
  modalInitialEpicId.value = epicId;
  showModalForm.value = true;
};

// モーダルを閉じる
const closeModal = () => {
  showModalForm.value = false;
  modalInitialStatus.value = 'Open';
  modalInitialEpicId.value = 0;
};

// ドラッグ&ドロップでステータスを更新（別のエピックのレーンへ落としたらエピックも変える）
const handleDrop = async ({ taskId, newStatus, epicId }) => {
  try {
    error.value = '';
    const task = todos.value.find(t => t.ID === taskId);
    if (!task) {
      throw new Error('タスクが見つかりません');
    }
    
    // 「エピックなし」のレーンには、選択中のスペースで表示していないエピックのタスクも並ぶので、そのまま残す
    const isSameLane = epicId === 0
      ? !visibleEpics.value.some(e => e.ID === task.EpicID)
      : task.EpicID === epicId;
    const newEpicId = isSameLane ? task.EpicID : epicId;

    // ステータスもレーンも変わらない場合は何もしない
    if (task.Status === newStatus && isSameLane) {
      return;
    }

    // タスクはエピックと同じスペースに置く
    const epic = epics.value.find(e => e.ID === newEpicId);
    
    // タスクを更新
    await todosApi.update(taskId, {
      title: task.Title,
      description: task.Description,
      done: task.Done,
      deadline: task.Deadline,
      status: newStatus,
      epicId: newEpicId,
      spaceId: epic ? epic.SpaceID : task.SpaceID
    });
    
    await fetchTodos(); // リストを再取得
  } catch (err) {
    error.value = 'タスクの更新に失敗しました: ' + err.message;
    console.error('Failed to update todo:', err);
    await resyncTodos();
  }
};

// タスクのエピックを変更
const handleChangeEpic = async ({ taskId, epicId }) => {
  try {
    error.value = '';
    const task = todos.value.find(t => t.ID === taskId);
    if (!task) {
      throw new Error('タスクが見つかりません');
    }

    await todosApi.update(taskId, {
      title: task.Title,
      description: task.Description,
      done: task.Done,
      deadline: task.Deadline,
      status: task.Status,
      epicId,
      spaceId: task.SpaceID
    });

    await fetchTodos(); // リストを再取得
  } catch (err) {
    error.value = 'エピックの変更に失敗しました: ' + err.message;
    console.error('Failed to change epic:', err);
    await resyncTodos();
  }
};

// タスクを削除
const handleDeleteTodo = async (taskId) => {
  if (!confirm('このタスクを削除してもよろしいですか？')) {
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
  fetchTodos();
  fetchEpics();
  fetchSpaces();
});
</script>
