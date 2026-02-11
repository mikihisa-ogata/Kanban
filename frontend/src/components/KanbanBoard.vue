<template>
  <div class="kanban-board">
    <TaskForm @submit="handleCreateTodo" />
    
    <div class="board-columns">
      <KanbanColumn
        v-for="column in columns"
        :key="column.status"
        :status="column.status"
        :title="column.title"
        :tasks="getTasksByStatus(column.status)"
        @drop="handleDrop"
        @delete-task="handleDeleteTodo"
      />
    </div>
    
    <div v-if="error" class="error-message">
      {{ error }}
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import KanbanColumn from './KanbanColumn.vue';
import TaskForm from './TaskForm.vue';
import { todosApi } from '../api/todos';

const todos = ref([]);
const error = ref('');

const columns = [
  { status: 'Open', title: 'オープン' },
  { status: 'InProgress', title: '進行中' },
  { status: 'Waiting', title: '待ち状況' },
  { status: 'Closed', title: 'クローズ' }
];

// ステータスごとにタスクをフィルタリング
const getTasksByStatus = (status) => {
  return todos.value.filter(todo => todo.Status === status);
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

// タスクを作成
const handleCreateTodo = async (formData) => {
  try {
    error.value = '';
    await todosApi.create({
      title: formData.title,
      deadline: formData.deadline,
      status: formData.status
    });
    await fetchTodos(); // リストを再取得
  } catch (err) {
    error.value = 'タスクの作成に失敗しました: ' + err.message;
    console.error('Failed to create todo:', err);
  }
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
      status: newStatus
    });
    
    await fetchTodos(); // リストを再取得
  } catch (err) {
    error.value = 'タスクの更新に失敗しました: ' + err.message;
    console.error('Failed to update todo:', err);
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
  }
};

// コンポーネントマウント時にタスクを取得
onMounted(() => {
  fetchTodos();
});
</script>

<style scoped>
.kanban-board {
  padding: 24px;
  min-height: 100vh;
}

.board-columns {
  display: flex;
  gap: 20px;
  overflow-x: auto;
  padding-bottom: 20px;
}

.error-message {
  position: fixed;
  bottom: 20px;
  right: 20px;
  background: #fee2e2;
  color: #991b1b;
  padding: 16px 20px;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  max-width: 400px;
  font-size: 14px;
  font-weight: 600;
  animation: slideIn 0.3s ease;
}

@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

/* レスポンシブ対応 */
@media (max-width: 768px) {
  .kanban-board {
    padding: 16px;
  }
  
  .board-columns {
    flex-direction: column;
    gap: 16px;
  }
}
</style>
