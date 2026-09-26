<template>
  <section class="bg-white/20 rounded-xl p-3">
    <div
      :class="[
        'flex items-center gap-2 px-2 pb-3 text-white',
        epic ? 'cursor-grab active:cursor-grabbing' : ''
      ]"
      :draggable="!!epic"
      :title="epic ? 'サイドバーのスペースへドラッグすると、子タスクごと移動できます' : ''"
      @dragstart="handleDragStart"
    >
      <h2 class="m-0 text-base font-bold truncate">{{ epic ? epic.Title : 'エピックなし' }}</h2>
      <span v-if="spaceTitle" class="px-2 py-0.5 rounded-full bg-teal-50 text-teal-700 text-xs font-semibold">{{ spaceTitle }}</span>
      <span class="text-xs opacity-80">{{ progressLabel }}</span>
      <button
        v-if="epic"
        class="ml-auto bg-transparent border-0 p-0 leading-none text-lg text-white opacity-60 hover:opacity-100 cursor-pointer"
        title="エピックを削除"
        @click="$emit('delete-epic', epic.ID)"
      >
        ×
      </button>
    </div>

    <div class="grid grid-cols-4 gap-4">
      <KanbanColumn
        v-for="column in columns"
        :key="column.status"
        :status="column.status"
        :tasks="tasks.filter(task => task.Status === column.status)"
        :epics="epics"
        @drop="$emit('drop', { ...$event, epicId: epic ? epic.ID : 0 })"
        @change-epic="$emit('change-epic', $event)"
        @delete-task="$emit('delete-task', $event)"
        @open-task="$emit('open-task', $event)"
        @add-task="$emit('add-task', { status: $event, epicId: epic ? epic.ID : 0 })"
      />
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue';
import KanbanColumn from './KanbanColumn.vue';

const props = defineProps({
  // null の場合は「エピックなし」のレーン
  epic: {
    type: Object,
    default: null
  },
  // このレーンに表示するタスク
  tasks: {
    type: Array,
    required: true
  },
  epics: {
    type: Array,
    default: () => []
  },
  columns: {
    type: Array,
    required: true
  },
  // 全スペース表示のときだけ、エピックの属するスペース名を出す
  spaceTitle: {
    type: String,
    default: ''
  }
});

const emit = defineEmits(['drop', 'change-epic', 'delete-task', 'add-task', 'delete-epic', 'open-task']);

// 子タスクのうちクローズ済みの件数 / 全件数
const progressLabel = computed(() => {
  const closed = props.tasks.filter(todo => todo.Status === 'Closed').length;
  return `${closed}/${props.tasks.length}`;
});

// サイドバーのスペースへドロップすると、エピックを子タスクごと別のスペースへ移動できる
const handleDragStart = (event) => {
  if (!props.epic) return;
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('epicId', props.epic.ID);
};
</script>
