<template>
  <div class="px-8 py-8 min-h-screen">
    <SpaceBar
      :spaces="spaces"
      :todos="todos"
      :selected-space-id="selectedSpaceId"
      @select="selectSpace"
      @create="handleCreateSpace"
      @delete="handleDeleteSpace"
      @move="handleMoveToSpace"
    />

    <EpicBar
      class="mt-4"
      :epics="visibleEpics"
      :todos="todos"
      :selected-epic-id="selectedEpicId"
      @select="selectedEpicId = $event"
      @create="handleCreateEpic"
      @delete="handleDeleteEpic"
    />

    <div class="flex gap-6 overflow-x-auto pb-6 mt-6">
      <KanbanColumn
        v-for="column in columns"
        :key="column.status"
        :status="column.status"
        :title="column.title"
        :tasks="getTasksByStatus(column.status)"
        :epics="epics"
        @drop="handleDrop"
        @change-epic="handleChangeEpic"
        @delete-task="handleDeleteTodo"
        @add-task="openModal"
      />
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
        :initial-epic-id="selectedEpicId ?? 0"
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
import KanbanColumn from './KanbanColumn.vue';
import TaskForm from './TaskForm.vue';
import EpicBar from './EpicBar.vue';
import SpaceBar from './SpaceBar.vue';
import { todosApi } from '../api/todos';
import { epicsApi } from '../api/epics';
import { spacesApi } from '../api/spaces';

const todos = ref([]);
const epics = ref([]);
const spaces = ref([]);
// null の場合は全スペースを表示
const selectedSpaceId = ref(null);
// null の場合は全タスクを表示
const selectedEpicId = ref(null);
const error = ref('');
const showModalForm = ref(false);
const modalInitialStatus = ref('Open');

const columns = [
  { status: 'Open', title: 'オープン' },
  { status: 'InProgress', title: '進行中' },
  { status: 'Waiting', title: '待ち状況' },
  { status: 'Closed', title: 'クローズ' }
];

// 選択中のスペースに属するエピック
const visibleEpics = computed(() =>
  selectedSpaceId.value === null
    ? epics.value
    : epics.value.filter(epic => epic.SpaceID === selectedSpaceId.value)
);

// タスク作成フォームの初期スペース（エピック選択中はそのエピックのスペース）
const modalInitialSpaceId = computed(() => {
  const epic = epics.value.find(e => e.ID === selectedEpicId.value);
  return epic ? epic.SpaceID : (selectedSpaceId.value ?? 0);
});

// ステータスごとにタスクをフィルタリング（スペース・エピック選択中はそれに属するタスクのみ）
const getTasksByStatus = (status) => {
  return todos.value.filter(todo =>
    todo.Status === status &&
    (selectedSpaceId.value === null || todo.SpaceID === selectedSpaceId.value) &&
    (selectedEpicId.value === null || todo.EpicID === selectedEpicId.value)
  );
};

// スペースを切り替える（エピックの選択は解除する）
const selectSpace = (spaceId) => {
  selectedSpaceId.value = spaceId;
  selectedEpicId.value = null;
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
      if (selectedEpicId.value === id && selectedSpaceId.value !== null) {
        selectedEpicId.value = null;
      }
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
const handleCreateEpic = async (title) => {
  try {
    error.value = '';
    await epicsApi.create({ title, spaceId: selectedSpaceId.value ?? 0 });
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
    if (selectedEpicId.value === epicId) {
      selectedEpicId.value = null;
    }
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

// モーダルを開く
const openModal = (status) => {
  modalInitialStatus.value = status;
  showModalForm.value = true;
};

// モーダルを閉じる
const closeModal = () => {
  showModalForm.value = false;
  modalInitialStatus.value = 'Open';
};

// ドラッグ&ドロップでステータスを更新
const handleDrop = async ({ taskId, newStatus }) => {
  try {
    error.value = '';
    const task = todos.value.find(t => t.ID === taskId);
    if (!task) {
      throw new Error('タスクが見つかりません');
    }
    
    // ステータスが変わらない場合は何もしない
    if (task.Status === newStatus) {
      return;
    }
    
    // タスクを更新
    await todosApi.update(taskId, {
      title: task.Title,
      description: task.Description,
      done: task.Done,
      deadline: task.Deadline,
      status: newStatus,
      epicId: task.EpicID,
      spaceId: task.SpaceID
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
