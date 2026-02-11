<template>
  <div class="task-card" draggable="true" @dragstart="handleDragStart">
    <div class="task-header">
      <h3 class="task-title">{{ task.Title }}</h3>
      <button class="delete-btn" @click="$emit('delete', task.ID)" title="削除">
        ×
      </button>
    </div>
    <div class="task-details">
      <div class="task-deadline">
        <span class="icon">📅</span>
        <span>{{ formatDate(task.Deadline) }}</span>
      </div>
    </div>
    <div v-if="task.Done" class="task-done">
      <span class="icon">✓</span>
      <span>完了</span>
    </div>
  </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue';

const props = defineProps({
  task: {
    type: Object,
    required: true
  }
});

const emit = defineEmits(['dragstart', 'delete']);

const handleDragStart = (event) => {
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('taskId', props.task.ID);
  emit('dragstart', props.task);
};

const formatDate = (dateString) => {
  const date = new Date(dateString);
  return date.toLocaleDateString('ja-JP', { year: 'numeric', month: 'short', day: 'numeric' });
};

const getStatusLabel = (status) => {
  const labels = {
    'Open': 'オープン',
    'InProgress': '進行中',
    'Waiting': '待ち状況',
    'Closed': 'クローズ'
  };
  return labels[status] || status;
};
</script>

<style scoped>
.task-card {
  background: white;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  cursor: grab;
  transition: all 0.2s ease;
  border-left: 4px solid #e5e7eb;
}

.task-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transform: translateY(-2px);
}

.task-card:active {
  cursor: grabbing;
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.task-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
  flex: 1;
  line-height: 1.4;
}

.delete-btn {
  background: none;
  border: none;
  color: #9ca3af;
  font-size: 24px;
  cursor: pointer;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s ease;
  line-height: 1;
}

.delete-btn:hover {
  background: #fee2e2;
  color: #ef4444;
}

.task-details {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.task-deadline {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  color: #6b7280;
}

.icon {
  font-size: 14px;
}

.status-open {
  background: #dbeafe;
  color: #1e40af;
}

.status-inprogress {
  background: #fef3c7;
  color: #92400e;
}

.status-waiting {
  background: #ede9fe;
  color: #5b21b6;
}

.status-closed {
  background: #d1fae5;
  color: #065f46;
}

.task-done {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  color: #10b981;
  font-weight: 600;
}
</style>
