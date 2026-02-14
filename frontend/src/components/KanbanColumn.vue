<template>
  <div 
    :class="[
      'bg-gray-50 rounded-xl p-5 flex-1 min-w-64 max-w-sm flex flex-col min-h-96 transition-all duration-200 border-2 border-transparent shadow-md',
      isDragOver ? 'bg-blue-50 border-blue-500 shadow-lg shadow-blue-300' : ''
    ]"
    @dragover.prevent="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <div 
      :class="[
        'flex justify-between items-center px-5 py-4 rounded-lg mb-5 text-white shadow-md',
        statusHeaderClass
      ]"
    >
      <h2 class="m-0 text-xl font-bold tracking-wide">{{ title }}</h2>
      <span class="bg-white bg-opacity-35 text-black px-3 py-1.5 rounded-full text-sm font-bold">{{ tasks.length }}</span>
    </div>
    <div class="flex-1 overflow-y-auto px-2">
      <TaskCard
        v-for="task in tasks"
        :key="task.ID"
        :task="task"
        @delete="$emit('delete-task', $event)"
      />
      <div v-if="tasks.length === 0" class="text-center py-10 text-gray-400 text-sm italic">
        タスクがありません
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, defineProps, defineEmits, computed } from 'vue';
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

const statusHeaderClass = computed(() => {
  const statusMap = {
    'Open': 'bg-gradient-to-r from-gray-400 to-gray-600',
    'InProgress': 'bg-gradient-to-r from-amber-400 to-amber-600',
    'Waiting': 'bg-gradient-to-r from-purple-500 to-purple-700',
    'Closed': 'bg-gradient-to-r from-emerald-500 to-emerald-700'
  };
  return statusMap[props.status] || '';
});

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
