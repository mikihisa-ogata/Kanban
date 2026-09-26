<template>
  <div class="bg-white bg-opacity-95 rounded-xl px-5 py-4 shadow-md">
    <div class="flex items-center gap-3 flex-wrap">
      <span class="text-sm font-bold text-gray-700">エピック</span>

      <button
        :class="chipClass(selectedEpicId === null)"
        @click="$emit('select', null)"
      >
        すべて
      </button>

      <div
        v-for="epic in epics"
        :key="epic.ID"
        :class="[chipClass(selectedEpicId === epic.ID), 'flex items-center gap-2']"
        @click="$emit('select', epic.ID)"
      >
        <span>{{ epic.Title }}</span>
        <span class="text-xs opacity-75">{{ progressLabel(epic.ID) }}</span>
        <button
          class="bg-transparent border-0 p-0 leading-none text-base opacity-60 hover:opacity-100 cursor-pointer"
          title="エピックを削除"
          @click.stop="$emit('delete', epic.ID)"
        >
          ×
        </button>
      </div>

      <form class="flex items-center gap-2" @submit.prevent="handleCreate">
        <input
          v-model="newTitle"
          type="text"
          placeholder="新しいエピック"
          class="px-3 py-1.5 border-2 border-gray-300 rounded-full text-sm text-gray-900 bg-white focus:outline-none focus:border-indigo-500"
        />
        <button
          type="submit"
          :disabled="!newTitle.trim()"
          class="px-3 py-1.5 bg-indigo-500 text-white border-0 rounded-full text-sm font-semibold cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
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
  epics: {
    type: Array,
    required: true
  },
  todos: {
    type: Array,
    required: true
  },
  selectedEpicId: {
    type: Number,
    default: null
  }
});

const emit = defineEmits(['select', 'create', 'delete']);

const newTitle = ref('');

const chipClass = (active) => [
  'px-3 py-1.5 rounded-full text-sm font-semibold cursor-pointer transition-all duration-200 border-2',
  active
    ? 'bg-indigo-500 text-white border-indigo-500'
    : 'bg-indigo-50 text-indigo-700 border-indigo-200 hover:bg-indigo-100'
];

// 子タスクのうちクローズ済みの件数 / 全件数
const progressLabel = (epicId) => {
  const children = props.todos.filter(todo => todo.EpicID === epicId);
  const closed = children.filter(todo => todo.Status === 'Closed').length;
  return `${closed}/${children.length}`;
};

const handleCreate = () => {
  const title = newTitle.value.trim();
  if (!title) return;
  emit('create', title);
  newTitle.value = '';
};
</script>
