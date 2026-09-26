<template>
  <div class="px-8 py-8 min-h-screen">
    <EpicBar
      :epics="epics"
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
        :epics="epics"
        @submit="handleCreateTodoFromModal"
        @close="closeModal"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import KanbanColumn from './KanbanColumn.vue';
import TaskForm from './TaskForm.vue';
import EpicBar from './EpicBar.vue';
import { todosApi } from '../api/todos';
import { epicsApi } from '../api/epics';

const todos = ref([]);
const epics = ref([]);
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

// ステータスごとにタスクをフィルタリング（エピック選択中はその子タスクのみ）
const getTasksByStatus = (status) => {
  return todos.value.filter(todo =>
    todo.Status === status &&
    (selectedEpicId.value === null || todo.EpicID === selectedEpicId.value)
  );
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

// エピックを作成
const handleCreateEpic = async (title) => {
  try {
    error.value = '';
    await epicsApi.create({ title });
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
      deadline: formData.deadline,
      status: formData.status,
      epicId: formData.epicId
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
      done: task.Done,
      deadline: task.Deadline,
      status: newStatus,
      epicId: task.EpicID
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
      done: task.Done,
      deadline: task.Deadline,
      status: task.Status,
      epicId
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
});
</script>
