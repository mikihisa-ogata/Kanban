<template>
  <!-- 子課題（サブタスク）のセクション。課題の詳細（ページとモーダル）で使い、その場で追加・ステータス変更・削除できる -->
  <section>
    <div class="flex items-center gap-3 mb-2">
      <h2 class="m-0 text-sm font-semibold text-gray-700">子課題</h2>
      <template v-if="subtasks.length">
        <!-- 完了（クローズ）したサブタスクの割合 -->
        <div
          class="flex-1 h-2 rounded-full bg-gray-200 overflow-hidden"
          role="progressbar"
          :aria-valuenow="doneCount"
          aria-valuemin="0"
          :aria-valuemax="subtasks.length"
          :aria-label="`子課題の進捗 ${doneCount}/${subtasks.length}`"
        >
          <div class="h-full bg-emerald-500 transition-all duration-300" :style="{ width: `${progressPercent}%` }"></div>
        </div>
        <span class="shrink-0 text-xs text-gray-500">{{ doneCount }}/{{ subtasks.length }} 完了</span>
      </template>
    </div>

    <ul v-if="subtasks.length" class="m-0 p-0 list-none border border-gray-200 rounded-lg divide-y divide-gray-200">
      <li v-for="subtask in subtasks" :key="subtask.ID" class="flex items-center gap-2 px-3 py-2">
        <IssueTypeIcon :type="subtask.IssueType" />
        <button
          type="button"
          class="shrink-0 bg-transparent border-0 p-0 font-mono text-xs font-semibold text-gray-500 cursor-pointer hover:text-blue-600 hover:underline"
          :title="`${subtask.Key} を開く`"
          @click="$emit('open', subtask.Key)"
        >
          {{ subtask.Key }}
        </button>
        <span :class="['flex-1 min-w-0 truncate text-sm', isDoneCategory(subtask.StatusCategory) ? 'text-gray-400 line-through' : 'text-gray-900']">
          {{ subtask.Title }}
        </span>
        <select
          :value="subtask.StatusID"
          :aria-label="`${subtask.Key} のステータス`"
          class="shrink-0 max-w-40 px-2 py-1 border border-gray-300 rounded-md text-xs text-gray-800 bg-white cursor-pointer hover:border-gray-400 focus:outline-none focus:border-blue-500"
          @change="changeStatus(subtask, Number($event.target.value))"
        >
          <option v-for="option in statusOptionsFor(subtask)" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
        <button
          type="button"
          class="shrink-0 bg-transparent border-0 p-0 w-6 h-6 flex items-center justify-center rounded text-gray-400 text-lg leading-none cursor-pointer hover:bg-red-100 hover:text-red-500"
          :title="`${subtask.Key} を削除`"
          @click="confirmingDelete = subtask"
        >
          ×
        </button>
      </li>
    </ul>

    <form class="flex items-center gap-2 mt-2" @submit.prevent="addSubtask">
      <input
        v-model="newTitle"
        type="text"
        placeholder="子課題を追加"
        aria-label="子課題のタイトル"
        class="flex-1 min-w-0 px-3 py-1.5 border-2 border-gray-300 rounded-lg text-sm text-gray-900 bg-white focus:outline-none focus:border-blue-500"
        @keydown.enter="handleEnter"
      />
      <button
        type="submit"
        :disabled="!newTitle.trim() || busy"
        class="shrink-0 px-3 py-1.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        作成
      </button>
    </form>
    <p v-if="error" class="mt-1 mb-0 text-xs text-red-600">{{ error }}</p>

    <ConfirmDialog
      v-if="confirmingDelete"
      title="子課題を削除しますか？"
      :message="`${confirmingDelete.Key}「${confirmingDelete.Title}」を削除します。この操作は取り消せません。`"
      @confirm="deleteSubtask"
      @cancel="confirmingDelete = null"
    />
  </section>
</template>

<script setup>
import { ref, computed } from 'vue';
import IssueTypeIcon from './IssueTypeIcon.vue';
import ConfirmDialog from './ConfirmDialog.vue';
import { todosApi } from '../api/todos';
import { statusesOfSpace, statusOptionsOf, isDoneCategory } from '../utils/statuses';

const props = defineProps({
  // 親課題
  parent: {
    type: Object,
    required: true
  },
  // 親課題のサブタスク
  subtasks: {
    type: Array,
    default: () => []
  },
  // すべてのワークフローのステータス（サブタスクのスペースのステータスを選択肢にする）
  statuses: {
    type: Array,
    default: () => []
  }
});

// changed: サブタスクを追加・変更・削除した（呼び出し側で一覧を取り直す）
// open: サブタスクを開く（課題キー）
const emit = defineEmits(['changed', 'open']);

// サブタスクのスペース（親と同じ）のステータス。ステータスを読み込む前でも今の値は表示する
const statusOptionsFor = (subtask) => {
  const options = statusOptionsOf(statusesOfSpace(props.statuses, subtask.SpaceID));
  if (!options.some(option => option.value === subtask.StatusID)) {
    options.unshift({ value: subtask.StatusID, label: subtask.StatusName || '—' });
  }
  return options;
};

// 完了（Done カテゴリ）のサブタスクの数
const doneCount = computed(() => props.subtasks.filter(subtask => isDoneCategory(subtask.StatusCategory)).length);
const progressPercent = computed(() => (props.subtasks.length ? Math.round((doneCount.value / props.subtasks.length) * 100) : 0));

const newTitle = ref('');
const busy = ref(false);
const error = ref('');
const confirmingDelete = ref(null);

// 日本語入力の変換確定の Enter では作成しない
const handleEnter = (event) => {
  if (event.isComposing || event.keyCode === 229) {
    event.preventDefault();
  }
};

// スペース・エピックは bff が親と同じにする
const addSubtask = async () => {
  const title = newTitle.value.trim();
  if (!title || busy.value) return;
  busy.value = true;
  try {
    error.value = '';
    await todosApi.create({ title, parentId: props.parent.ID });
    newTitle.value = '';
    emit('changed');
  } catch (err) {
    error.value = '子課題の作成に失敗しました: ' + err.message;
    console.error('Failed to create subtask:', err);
  } finally {
    busy.value = false;
  }
};

// ステータスだけを変える（他の項目は今の値を送る）
const changeStatus = async (subtask, statusId) => {
  try {
    error.value = '';
    await todosApi.update(subtask.ID, {
      title: subtask.Title,
      description: subtask.Description,
      deadline: subtask.Deadline,
      statusId,
      epicId: subtask.EpicID,
      spaceId: subtask.SpaceID
    });
    emit('changed');
  } catch (err) {
    error.value = '子課題の更新に失敗しました: ' + err.message;
    console.error('Failed to update subtask:', err);
    emit('changed');
  }
};

const deleteSubtask = async () => {
  const subtask = confirmingDelete.value;
  confirmingDelete.value = null;
  try {
    error.value = '';
    await todosApi.delete(subtask.ID);
    emit('changed');
  } catch (err) {
    error.value = '子課題の削除に失敗しました: ' + err.message;
    console.error('Failed to delete subtask:', err);
  }
};
</script>
