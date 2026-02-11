<template>
  <div 
    class="kanban-column"
    :class="{ 'drag-over': isDragOver }"
    @dragover.prevent="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <div class="column-header" :class="`header-${status.toLowerCase()}`">
      <h2 class="column-title">{{ title }}</h2>
      <span class="task-count">{{ tasks.length }}</span>
    </div>
    <div class="column-content">
      <TaskCard
        v-for="task in tasks"
        :key="task.ID"
        :task="task"
        @delete="$emit('delete-task', $event)"
      />
      <div v-if="tasks.length === 0" class="empty-state">
        タスクがありません
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, defineProps, defineEmits } from 'vue';
import TaskCard from './TaskCard.vue';

const props = defineProps({
  status: {
    type: String,
    required: true
  },
  title: {
    type: String,
    required: true
  },
  tasks: {
    type: Array,
    required: true
  }
});

const emit = defineEmits(['drop', 'delete-task']);

const isDragOver = ref(false);

const handleDragOver = (event) => {
  event.preventDefault();
  event.dataTransfer.dropEffect = 'move';
  isDragOver.value = true;
};

const handleDragLeave = () => {
  isDragOver.value = false;
};

const handleDrop = (event) => {
  event.preventDefault();
  isDragOver.value = false;
  const taskId = parseInt(event.dataTransfer.getData('taskId'));
  emit('drop', { taskId, newStatus: props.status });
};
</script>

<style scoped>
.kanban-column {
  background: #f9fafb;
  border-radius: 12px;
  padding: 16px;
  flex: 1;
  min-width: 220px;
  max-width: 350px;
  display: flex;
  flex-direction: column;
  height: fit-content;
  min-height: 400px;
  transition: all 0.2s ease;
  border: 2px solid transparent;
}

.kanban-column.drag-over {
  background: #eff6ff;
  border-color: #3b82f6;
  box-shadow: 0 0 12px rgba(59, 130, 246, 0.3);
}

.column-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-radius: 8px;
  margin-bottom: 16px;
}

.header-open {
  background: linear-gradient(135deg, #cbcbcb 0%, #a9a9a9 100%);
}

.header-inprogress {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
}

.header-waiting {
  background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
}

.header-closed {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.column-title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: white;
  letter-spacing: 0.5px;
}

.task-count {
  background: rgba(255, 255, 255, 0.3);
  color: white;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 600;
}

.column-content {
  flex: 1;
  overflow-y: auto;
  padding: 4px;
}

.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: #9ca3af;
  font-size: 14px;
  font-style: italic;
}

/* スクロールバーのスタイリング */
.column-content::-webkit-scrollbar {
  width: 6px;
}

.column-content::-webkit-scrollbar-track {
  background: transparent;
}

.column-content::-webkit-scrollbar-thumb {
  background: #d1d5db;
  border-radius: 3px;
}

.column-content::-webkit-scrollbar-thumb:hover {
  background: #9ca3af;
}
</style>
