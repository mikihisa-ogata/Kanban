<template>
  <section class="bg-white/20 rounded-xl p-3">
    <!-- レーンの見出し。スイムレーン「なし」では出さない -->
    <div
      v-if="lane.kind !== 'none'"
      :class="[
        'flex items-center gap-2 px-2 text-white',
        collapsed ? '' : 'pb-3',
        epic && !editing ? 'cursor-grab active:cursor-grabbing' : ''
      ]"
      :draggable="!!epic && !editing"
      :title="epic && !editing ? 'サイドバーのスペースへドラッグすると、子タスクごと移動できます' : ''"
      @dragstart="handleDragStart"
    >
      <button
        type="button"
        class="shrink-0 bg-transparent border-0 p-0 text-white opacity-80 hover:opacity-100 cursor-pointer"
        :title="collapsed ? 'レーンを開く' : 'レーンを折りたたむ'"
        :aria-label="collapsed ? 'レーンを開く' : 'レーンを折りたたむ'"
        :aria-expanded="!collapsed"
        @click="$emit('toggle-collapse')"
      >
        <svg :class="['w-5 h-5 block transition-transform', collapsed ? '-rotate-90' : '']" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
          <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
        </svg>
      </button>
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
        v-else-if="lane.kind === 'epic'"
        class="m-0 text-base font-bold truncate"
        :title="titleTooltip"
        @dblclick="startEdit"
      >
        {{ epic ? epic.Title : 'エピックなし' }}
      </h2>
      <!-- 担当者のレーン -->
      <template v-else-if="lane.kind === 'assignee'">
        <UserAvatar :user="lane.user || { DisplayName: '未割り当て', AvatarColor: '#A5ADBA' }" size="sm" />
        <h2 class="m-0 text-base font-bold truncate">{{ lane.user ? lane.user.DisplayName : '未割り当て' }}</h2>
      </template>
      <!-- 親課題のレーン。押すと親課題の詳細を開く -->
      <button
        v-else-if="lane.kind === 'parent'"
        type="button"
        class="flex items-center gap-2 min-w-0 bg-transparent border-0 p-0 text-white cursor-pointer hover:underline"
        :title="`${lane.parent.Key} ${lane.parent.Title}`"
        @click="$emit('open-task', lane.parent)"
      >
        <IssueTypeIcon :type="lane.parent.IssueType" />
        <span class="shrink-0 font-mono text-sm font-semibold opacity-90">{{ lane.parent.Key }}</span>
        <h2 class="m-0 text-base font-bold truncate">{{ lane.parent.Title }}</h2>
      </button>
      <h2 v-else class="m-0 text-base font-bold truncate">その他の課題</h2>
      <button
        v-if="epic && !editing"
        class="bg-transparent border-0 p-0 leading-none text-sm text-white opacity-60 hover:opacity-100 cursor-pointer"
        title="エピックの名前を変更"
        @click="startEdit"
      >
        ✎
      </button>
      <span v-if="lane.parent" class="shrink-0 px-2 py-0.5 rounded bg-white/25 text-xs font-semibold">{{ lane.parent.StatusName }}</span>
      <span v-if="lane.spaceTitle" class="px-2 py-0.5 rounded-full bg-teal-50 text-teal-700 text-xs font-semibold">{{ lane.spaceTitle }}</span>
      <span class="shrink-0 text-xs opacity-80">{{ progressLabel }}</span>
      <button
        v-if="epic"
        class="ml-auto bg-transparent border-0 p-0 leading-none text-lg text-white opacity-60 hover:opacity-100 cursor-pointer"
        title="エピックを削除"
        @click="$emit('delete-epic', epic.ID)"
      >
        ×
      </button>
    </div>

    <div v-if="!collapsed" class="grid gap-4" :style="gridStyle">
      <KanbanColumn
        v-for="column in columns"
        :key="column.key"
        :column="column"
        :tasks="lane.tasks.filter(task => isInColumn(task, column))"
        :can-add="lane.kind !== 'parent'"
        @drop="$emit('drop', { ...$event, lane })"
        @delete-task="$emit('delete-task', $event)"
        @open-task="$emit('open-task', $event)"
        @add-task="$emit('add-task', { column: $event, lane })"
      />
    </div>
  </section>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue';
import KanbanColumn from './KanbanColumn.vue';
import IssueTypeIcon from './IssueTypeIcon.vue';
import UserAvatar from './UserAvatar.vue';
import { formatDateTime } from '../utils/datetime';
import { isInColumn, isDoneCategory } from '../utils/statuses';

const props = defineProps({
  // スイムレーン（KanbanBoard の lanes の要素）。kind は epic（epic が null なら「エピックなし」）・
  // assignee（user が null なら未割り当て）・parent（親課題のサブタスク）・others（サブタスクのない課題）・none（レーン分けなし）。
  // tasks はこのレーンに表示するタスク
  lane: {
    type: Object,
    required: true
  },
  // 折りたたんでいるか
  collapsed: {
    type: Boolean,
    default: false
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
  }
});

const emit = defineEmits(['drop', 'delete-task', 'add-task', 'delete-epic', 'rename-epic', 'open-task', 'toggle-collapse']);

// エピックのレーンのエピック（名前の変更・削除・スペースへの移動に使う）
const epic = computed(() => (props.lane.kind === 'epic' ? props.lane.epic : null));

// エピック名を編集中かどうかと、入力中の名前
const editing = ref(false);
const editTitle = ref('');
const titleInput = ref(null);

// エピック名の編集を始める
const startEdit = async () => {
  if (!epic.value) return;
  editTitle.value = epic.value.Title;
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
  if (!title || title === epic.value.Title) return;
  emit('rename-epic', { id: epic.value.ID, title });
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
  if (!epic.value) return '';
  const lines = ['ダブルクリックで名前を変更'];
  if (epic.value.CreatedAt) lines.push(`作成: ${formatDateTime(epic.value.CreatedAt)}`);
  if (epic.value.UpdatedAt) lines.push(`更新: ${formatDateTime(epic.value.UpdatedAt)}`);
  return lines.join('\n');
});

// 子タスクのうち完了（Done カテゴリ）の件数 / 全件数
const progressLabel = computed(() => {
  const closed = props.lane.tasks.filter(todo => isDoneCategory(todo.StatusCategory)).length;
  return `${closed}/${props.lane.tasks.length}`;
});

// サイドバーのスペースへドロップすると、エピックを子タスクごと別のスペースへ移動できる
const handleDragStart = (event) => {
  if (!epic.value) return;
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('epicId', epic.value.ID);
};
</script>
