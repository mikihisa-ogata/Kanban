<template>
  <section class="bg-white/20 rounded-xl p-3">
    <div
      :class="[
        'flex items-center gap-2 px-2 pb-3 text-white',
        epic && !editing ? 'cursor-grab active:cursor-grabbing' : ''
      ]"
      :draggable="!!epic && !editing"
      :title="epic && !editing ? 'サイドバーのスペースへドラッグすると、子タスクごと移動できます' : ''"
      @dragstart="handleDragStart"
    >
      <input
        v-if="editing"
        ref="titleInput"
        v-model="editTitle"
        type="text"
        class="min-w-0 w-64 px-2 py-0.5 border-2 border-indigo-300 rounded-md text-base font-bold text-gray-900 bg-white focus:outline-none focus:border-indigo-500"
        @keydown.enter="handleEnter"
        @keydown.esc="cancelEdit"
        @blur="commitEdit"
      />
      <h2
        v-else
        class="m-0 text-base font-bold truncate"
        :title="titleTooltip"
        @dblclick="startEdit"
      >
        {{ epic ? epic.Title : 'エピックなし' }}
      </h2>
      <button
        v-if="epic && !editing"
        class="bg-transparent border-0 p-0 leading-none text-sm text-white opacity-60 hover:opacity-100 cursor-pointer"
        title="エピックの名前を変更"
        @click="startEdit"
      >
        ✎
      </button>
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

    <div class="grid gap-4" :style="gridStyle">
      <KanbanColumn
        v-for="column in columns"
        :key="column.key"
        :column="column"
        :tasks="tasks.filter(task => isInColumn(task, column))"
        @drop="$emit('drop', { ...$event, epicId: epic ? epic.ID : 0 })"
        @delete-task="$emit('delete-task', $event)"
        @open-task="$emit('open-task', $event)"
        @add-task="$emit('add-task', { column: $event, epicId: epic ? epic.ID : 0 })"
      />
    </div>
  </section>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue';
import KanbanColumn from './KanbanColumn.vue';
import { formatDateTime } from '../utils/datetime';
import { isInColumn, isDoneCategory } from '../utils/statuses';

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
  // ボードの列（KanbanBoard の columns）
  columns: {
    type: Array,
    required: true
  },
  // 列の並べ方（見出しと同じ grid-template-columns）
  gridStyle: {
    type: Object,
    default: () => ({})
  },
  // 全スペース表示のときだけ、エピックの属するスペース名を出す
  spaceTitle: {
    type: String,
    default: ''
  }
});

const emit = defineEmits(['drop', 'delete-task', 'add-task', 'delete-epic', 'rename-epic', 'open-task']);

// エピック名を編集中かどうかと、入力中の名前
const editing = ref(false);
const editTitle = ref('');
const titleInput = ref(null);

// エピック名の編集を始める
const startEdit = async () => {
  if (!props.epic) return;
  editTitle.value = props.epic.Title;
  editing.value = true;
  await nextTick();
  titleInput.value?.focus();
  titleInput.value?.select();
};

// 入力した名前で確定する。空欄や変更なしの場合は元の名前に戻す
const commitEdit = () => {
  if (!editing.value) return;
  editing.value = false;
  const title = editTitle.value.trim();
  if (!title || title === props.epic.Title) return;
  emit('rename-epic', { id: props.epic.ID, title });
};

// 日本語入力の変換確定の Enter では確定しない
const handleEnter = (event) => {
  if (event.isComposing || event.keyCode === 229) return;
  commitEdit();
};

// 編集をやめて元の名前に戻す
const cancelEdit = () => {
  editing.value = false;
};

// エピック名にホバーしたときの説明。作成日時と最終更新日時も出す
const titleTooltip = computed(() => {
  if (!props.epic) return '';
  const lines = ['ダブルクリックで名前を変更'];
  if (props.epic.CreatedAt) lines.push(`作成: ${formatDateTime(props.epic.CreatedAt)}`);
  if (props.epic.UpdatedAt) lines.push(`更新: ${formatDateTime(props.epic.UpdatedAt)}`);
  return lines.join('\n');
});

// 子タスクのうち完了（Done カテゴリ）の件数 / 全件数
const progressLabel = computed(() => {
  const closed = props.tasks.filter(todo => isDoneCategory(todo.StatusCategory)).length;
  return `${closed}/${props.tasks.length}`;
});

// サイドバーのスペースへドロップすると、エピックを子タスクごと別のスペースへ移動できる
const handleDragStart = (event) => {
  if (!props.epic) return;
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('epicId', props.epic.ID);
};
</script>
