<template>
  <div class="bg-white rounded-lg p-4 mb-4 shadow hover:shadow-lg cursor-grab active:cursor-grabbing transition-all duration-200 hover:-translate-y-1" draggable="true" @dragstart="handleDragStart">
    <div class="flex justify-between items-start mb-3">
      <h3 class="m-0 text-base font-semibold text-gray-900 flex-1 leading-relaxed">{{ task.Title }}</h3>
      <button class="bg-transparent border-0 text-gray-400 text-2xl cursor-pointer p-0 w-6 h-6 flex items-center justify-center rounded hover:bg-red-100 hover:text-red-500 transition-all duration-200 leading-none" @click="$emit('delete', task.ID)" title="削除">
        ×
      </button>
    </div>
    <div class="flex justify-between items-center gap-2 flex-wrap">
      <div class="flex items-center gap-1.5 text-sm text-gray-600">
        <span class="text-sm">📅</span>
        <span>{{ formatDate(task.Deadline) }}</span>
      </div>
    </div>
    <div v-if="task.Done" class="mt-3 pt-3 border-t border-gray-200 flex items-center gap-1.5 text-sm text-emerald-600 font-semibold">
      <span class="text-sm">✓</span>
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
