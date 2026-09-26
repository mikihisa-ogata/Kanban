<template>
  <div 
    :class="[
      'bg-gray-50 rounded-xl p-3 min-w-0 flex flex-col transition-all duration-200 border-2 border-transparent shadow-md',
      isDragOver ? 'bg-blue-50 border-blue-500 shadow-lg shadow-blue-300' : ''
    ]"
    @dragover.prevent="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <div class="flex-1">
      <TaskCard
        v-for="task in tasks"
        :key="task.ID"
        :task="task"
        :epics="epics"
        @delete="$emit('delete-task', $event)"
        @change-epic="$emit('change-epic', $event)"
        @open="$emit('open-task', $event)"
      />
      <!-- +ボタン -->
      <button
        @click="$emit('add-task', status)"
        class="w-full py-2 px-4 bg-gray-200 hover:bg-gray-300 text-gray-700 border-2 border-dashed border-gray-400 rounded-lg text-sm font-semibold cursor-pointer transition-all duration-200"
      >
        + タスクを追加
      </button>
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
  tasks: {
    type: Array,
    required: true
  },
  epics: {
    type: Array,
    default: () => []
  }
});

const emit = defineEmits(['drop', 'delete-task', 'add-task', 'change-epic', 'open-task']);

const isDragOver = ref(false);

// タスクカード以外（エピックなど）のドラッグは受け付けない
const isTaskDrag = (event) => Array.from(event.dataTransfer.types).includes('taskid');

const handleDragOver = (event) => {
  if (!isTaskDrag(event)) return;
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
  if (Number.isNaN(taskId)) return;
  emit('drop', { taskId, newStatus: props.status });
};
</script>
