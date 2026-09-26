<template>
  <div class="bg-white bg-opacity-95 rounded-xl px-5 py-4 shadow-md">
    <div class="flex items-center gap-3 flex-wrap">
      <span class="text-sm font-bold text-gray-700">スペース</span>

      <button
        :class="chipClass(selectedSpaceId === null, false)"
        @click="$emit('select', null)"
      >
        すべて
      </button>

      <div
        v-for="space in spaces"
        :key="space.ID"
        :class="[chipClass(selectedSpaceId === space.ID, dragOverSpaceId === space.ID), 'flex items-center gap-2']"
        title="タスクやエピックをドラッグすると、このスペースへ移動できます"
        @click="$emit('select', space.ID)"
        @dragover="handleDragOver($event, space.ID)"
        @dragleave="handleDragLeave(space.ID)"
        @drop="handleDrop($event, space.ID)"
      >
        <span>{{ space.Title }}</span>
        <span class="text-xs opacity-75">{{ progressLabel(space.ID) }}</span>
        <button
          class="bg-transparent border-0 p-0 leading-none text-base opacity-60 hover:opacity-100 cursor-pointer"
          title="スペースを削除"
          @click.stop="$emit('delete', space.ID)"
        >
          ×
        </button>
      </div>

      <form class="flex items-center gap-2" @submit.prevent="handleCreate">
        <input
          v-model="newTitle"
          type="text"
          placeholder="新しいスペース"
          class="px-3 py-1.5 border-2 border-gray-300 rounded-full text-sm text-gray-900 bg-white focus:outline-none focus:border-teal-500"
        />
        <button
          type="submit"
          :disabled="!newTitle.trim()"
          class="px-3 py-1.5 bg-teal-500 text-white border-0 rounded-full text-sm font-semibold cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
        >
          + 追加
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue';

const props = defineProps({
  spaces: {
    type: Array,
    required: true
  },
  todos: {
    type: Array,
    required: true
  },
  selectedSpaceId: {
    type: Number,
    default: null
  }
});

const emit = defineEmits(['select', 'create', 'delete', 'move']);

const newTitle = ref('');
const dragOverSpaceId = ref(null);

const chipClass = (active, dragOver) => [
  'px-3 py-1.5 rounded-full text-sm font-semibold cursor-pointer transition-all duration-200 border-2',
  active
    ? 'bg-teal-500 text-white border-teal-500'
    : 'bg-teal-50 text-teal-700 border-teal-200 hover:bg-teal-100',
  dragOver ? 'ring-4 ring-teal-300' : ''
];

// スペース内のタスクのうちクローズ済みの件数 / 全件数
const progressLabel = (spaceId) => {
  const children = props.todos.filter(todo => todo.SpaceID === spaceId);
  const closed = children.filter(todo => todo.Status === 'Closed').length;
  return `${closed}/${children.length}`;
};

// タスクカード（taskId）かエピック（epicId）のドラッグだけを受け付ける
const isMovableDrag = (event) => {
  const types = Array.from(event.dataTransfer.types);
  return types.includes('taskid') || types.includes('epicid');
};

const handleDragOver = (event, spaceId) => {
  if (!isMovableDrag(event)) return;
  event.preventDefault();
  event.dataTransfer.dropEffect = 'move';
  dragOverSpaceId.value = spaceId;
};

const handleDragLeave = (spaceId) => {
  if (dragOverSpaceId.value === spaceId) {
    dragOverSpaceId.value = null;
  }
};

const handleDrop = (event, spaceId) => {
  event.preventDefault();
  dragOverSpaceId.value = null;
  const taskId = parseInt(event.dataTransfer.getData('taskId'));
  if (!Number.isNaN(taskId)) {
    emit('move', { type: 'task', id: taskId, spaceId });
    return;
  }
  const epicId = parseInt(event.dataTransfer.getData('epicId'));
  if (!Number.isNaN(epicId)) {
    emit('move', { type: 'epic', id: epicId, spaceId });
  }
};

const handleCreate = () => {
  const title = newTitle.value.trim();
  if (!title) return;
  emit('create', title);
  newTitle.value = '';
};
</script>
