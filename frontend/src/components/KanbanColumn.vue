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
      <template v-for="(task, index) in tasks" :key="task.ID">
        <!-- ドロップ先の位置を示す線。前後の余白に重ねてレイアウトを動かさない -->
        <div v-if="dropIndex === index" class="h-1 -mt-2.5 mb-1.5 rounded-full bg-blue-500"></div>
        <TaskCard
          :task="task"
          :data-task-id="task.ID"
          @delete="$emit('delete-task', $event)"
          @open="$emit('open-task', $event)"
        />
      </template>
      <div v-if="dropIndex === tasks.length" class="h-1 -mt-2.5 mb-1.5 rounded-full bg-blue-500"></div>
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
  }
});

const emit = defineEmits(['drop', 'delete-task', 'add-task', 'open-task']);

const isDragOver = ref(false);
// ドロップしたときに挿入する位置（tasks のインデックス）。null の場合はドラッグ中でない
const dropIndex = ref(null);

// マウスの位置から挿入先を求める。カードの上半分ならその前、下半分ならその後ろ
const getDropIndex = (event) => {
  const cards = Array.from(event.currentTarget.querySelectorAll('[data-task-id]'));
  const index = cards.findIndex(card => {
    const rect = card.getBoundingClientRect();
    return event.clientY < rect.top + rect.height / 2;
  });
  return index === -1 ? cards.length : index;
};

// タスクカード以外（エピックなど）のドラッグは受け付けない
const isTaskDrag = (event) => Array.from(event.dataTransfer.types).includes('taskid');

const handleDragOver = (event) => {
  if (!isTaskDrag(event)) return;
  event.preventDefault();
  event.dataTransfer.dropEffect = 'move';
  isDragOver.value = true;
  dropIndex.value = getDropIndex(event);
};

const handleDragLeave = (event) => {
  // 列の中のカードへ移っただけのときは表示を消さない
  if (event.currentTarget.contains(event.relatedTarget)) return;
  isDragOver.value = false;
  dropIndex.value = null;
};

const handleDrop = (event) => {
  event.preventDefault();
  const index = dropIndex.value ?? getDropIndex(event);
  isDragOver.value = false;
  dropIndex.value = null;
  const taskId = parseInt(event.dataTransfer.getData('taskId'));
  if (Number.isNaN(taskId)) return;

  // 同じ列で元の位置（自分の直前・直後）に落とした場合は並び順を変えない（beforeId: null）
  const currentIndex = props.tasks.findIndex(task => task.ID === taskId);
  const isSamePosition = currentIndex !== -1 && (index === currentIndex || index === currentIndex + 1);
  // 挿入先の直後のタスクの ID。列の末尾に落とした場合は 0（全体の末尾へ移動）
  const beforeId = isSamePosition ? null : (props.tasks[index]?.ID ?? 0);
  emit('drop', { taskId, newStatus: props.status, beforeId });
};
</script>
