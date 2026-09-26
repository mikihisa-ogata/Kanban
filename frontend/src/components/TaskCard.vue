<template>
  <div class="bg-white rounded-lg p-4 mb-4 shadow hover:shadow-lg cursor-grab active:cursor-grabbing transition-all duration-200 hover:-translate-y-1" draggable="true" @dragstart="handleDragStart">
    <div class="flex justify-between items-start mb-3">
      <h3 class="m-0 text-base font-semibold text-gray-900 flex-1 leading-relaxed">{{ task.Title }}</h3>
      <button class="bg-transparent border-0 text-gray-400 text-2xl cursor-pointer p-0 w-6 h-6 flex items-center justify-center rounded hover:bg-red-100 hover:text-red-500 transition-all duration-200 leading-none" @click="$emit('delete', task.ID)" title="削除">
        ×
      </button>
    </div>
    <p v-if="task.Description" class="m-0 mb-3 text-sm text-gray-600 whitespace-pre-wrap break-words line-clamp-3" :title="task.Description">{{ task.Description }}</p>
    <div class="flex justify-between items-center gap-2 flex-wrap">
      <div v-if="task.Deadline" class="flex items-center gap-1.5 text-sm text-gray-600">
        <span class="text-sm">📅</span>
        <span>{{ formatDate(task.Deadline) }}</span>
      </div>
      <select
        :value="task.EpicID"
        :class="[
          'ml-auto max-w-40 truncate px-2 py-0.5 rounded-full text-xs font-semibold border cursor-pointer focus:outline-none',
          task.EpicID ? 'bg-indigo-50 text-indigo-700 border-indigo-200' : 'bg-gray-50 text-gray-400 border-gray-200'
        ]"
        title="エピック"
        @change="$emit('change-epic', { taskId: task.ID, epicId: Number($event.target.value) })"
      >
        <option :value="0">エピックなし</option>
        <option v-for="epic in epics" :key="epic.ID" :value="epic.ID">{{ epic.Title }}</option>
      </select>
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
  },
  epics: {
    type: Array,
    default: () => []
  }
});

const emit = defineEmits(['dragstart', 'delete', 'change-epic']);

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
