<template>
  <!-- スイムレーンの中の 1 つの列の区画。カードのない区画は点線の枠にする -->
  <div
    :class="[
      'group/cell rounded-lg p-1 min-w-0 min-h-14 flex flex-col transition-colors duration-150 border',
      isDragOver
        ? 'bg-accent/10 border-accent'
        : column.overLimit ? 'bg-danger/5 border-danger/30' : tasks.length ? 'border-transparent' : 'border-dashed border-line'
    ]"
    @dragover.prevent="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <div class="flex-1 flex flex-col gap-1.5">
      <template v-for="(task, index) in tasks" :key="task.ID">
        <!-- ドロップ先の位置を示す線。前後の余白に重ねてレイアウトを動かさない -->
        <div v-if="dropIndex === index" class="h-0.5 -my-1 rounded-full bg-accent"></div>
        <TaskCard
          :task="task"
          :data-task-id="task.ID"
          @open="$emit('open-task', $event)"
        />
      </template>
      <div v-if="dropIndex === tasks.length && tasks.length" class="h-0.5 -my-1 rounded-full bg-accent"></div>
      <!-- 課題の追加。カードがあるときは目立たせず、区画にマウスを乗せたときに出す -->
      <button
        v-if="canAdd"
        :class="[
          'w-full py-1 px-2 border-0 rounded-md bg-transparent text-left text-xs text-muted cursor-pointer hover:bg-raised hover:text-fg transition-colors duration-150',
          tasks.length ? 'opacity-0 group-hover/cell:opacity-100 focus:opacity-100' : ''
        ]"
        @click="$emit('add-task', column)"
      >
        + 課題を追加
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, defineProps, defineEmits } from 'vue';
import TaskCard from './TaskCard.vue';

const props = defineProps({
  // ボードの列（ステータスかカテゴリ）
  column: {
    type: Object,
    required: true
  },
  tasks: {
    type: Array,
    required: true
  },
  // 「+ タスクを追加」を出すか（親課題のレーンでは、サブタスクは課題の詳細から作るので出さない）
  canAdd: {
    type: Boolean,
    default: true
  }
});

const emit = defineEmits(['drop', 'add-task', 'open-task']);

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
  emit('drop', { taskId, column: props.column, beforeId });
};
</script>
