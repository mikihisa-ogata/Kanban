<template>
  <!-- 固定したヘッダーの下に張り付ける位置指定と開閉の動きは、呼び出し側（KanbanBoard）で行う -->
  <aside class="h-[calc(100vh-4.25rem)] overflow-y-auto w-64 bg-white bg-opacity-95 px-4 py-6 flex flex-col gap-2">
    <span class="px-1 text-sm font-bold text-gray-700">スペース</span>

    <button
      :class="itemClass(selectedSpaceId === null, false)"
      @click="$emit('select', null)"
    >
      <span class="flex-1 truncate">すべて</span>
    </button>

    <div
      v-for="space in spaces"
      :key="space.ID"
      :class="itemClass(selectedSpaceId === space.ID, dragOverSpaceId === space.ID)"
      title="タスクやエピックをドラッグすると、このスペースへ移動できます"
      @click="$emit('select', space.ID)"
      @dragover="handleDragOver($event, space.ID)"
      @dragleave="handleDragLeave(space.ID)"
      @drop="handleDrop($event, space.ID)"
    >
      <span class="flex-1 truncate">{{ space.Title }}</span>
      <span v-if="space.Key" class="font-mono text-xs opacity-75">{{ space.Key }}</span>
      <span class="text-xs opacity-75">{{ progressLabel(space.ID) }}</span>
      <NuxtLink
        :to="`/spaces/${space.ID}/settings`"
        class="leading-none text-current opacity-60 hover:opacity-100"
        title="スペースの設定"
        @click.stop
      >
        <svg class="w-4 h-4 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <circle cx="12" cy="12" r="3" />
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.01a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.01a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
        </svg>
      </NuxtLink>
      <button
        class="bg-transparent border-0 p-0 leading-none text-base opacity-60 hover:opacity-100 cursor-pointer"
        title="スペースを削除"
        @click.stop="$emit('delete', space.ID)"
      >
        ×
      </button>
    </div>

    <form class="mt-2 flex flex-col gap-2" @submit.prevent="handleCreate">
      <div class="flex items-center gap-2">
        <input
          v-model="newTitle"
          type="text"
          placeholder="新しいスペース"
          class="min-w-0 flex-1 px-3 py-1.5 border-2 border-gray-300 rounded-full text-sm text-gray-900 bg-white focus:outline-none focus:border-teal-500"
        />
        <button
          type="submit"
          :disabled="!canCreate"
          class="shrink-0 px-3 py-1.5 bg-teal-500 text-white border-0 rounded-full text-sm font-semibold cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
        >
          + 追加
        </button>
      </div>
      <!-- 課題キーの接頭辞。名前から候補を入れ、手で変えることもできる -->
      <div v-if="newTitle.trim()" class="flex items-center gap-2 px-1">
        <label for="new-space-key" class="text-xs font-semibold text-gray-600">キー</label>
        <input
          id="new-space-key"
          :value="newKey"
          type="text"
          maxlength="10"
          placeholder="KAN"
          class="w-28 px-2 py-1 border-2 border-gray-300 rounded-md font-mono text-sm uppercase text-gray-900 bg-white focus:outline-none focus:border-teal-500"
          @input="handleKeyInput"
        />
        <span v-if="newKey && !keyError" class="text-xs text-gray-500 font-mono">{{ newKey }}-1</span>
      </div>
      <!-- 候補の取得中（キーが空）はエラーを出さず、追加ボタンを押せないだけにする -->
      <p v-if="newTitle.trim() && newKey && keyError" class="m-0 px-1 text-xs text-red-600">{{ keyError }}</p>
    </form>
  </aside>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { spacesApi } from '../api/spaces';
import { normalizeSpaceKey, spaceKeyError } from '../utils/spaceKey';

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
const newKey = ref('');
// キーを手で入力したら、名前からの候補で上書きしない（空にすると候補に戻る）
const keyEdited = ref(false);
const dragOverSpaceId = ref(null);

const keyError = computed(() => spaceKeyError(newKey.value, props.spaces));

const canCreate = computed(() => newTitle.value.trim() !== '' && !keyError.value);

// 名前の入力が止まったら、サーバーからキーの候補を取得する（古い応答は捨てる）
let suggestTimer = null;
let suggestRequest = 0;
const requestKeySuggestion = (title) => {
  clearTimeout(suggestTimer);
  if (keyEdited.value) return;
  const trimmed = title.trim();
  if (!trimmed) {
    newKey.value = '';
    return;
  }
  suggestTimer = setTimeout(async () => {
    const request = ++suggestRequest;
    try {
      const key = await spacesApi.suggestKey(trimmed);
      if (request === suggestRequest && !keyEdited.value) {
        newKey.value = key;
      }
    } catch (err) {
      console.error('Failed to suggest space key:', err);
    }
  }, 300);
};
watch(newTitle, requestKeySuggestion);

const handleKeyInput = (event) => {
  newKey.value = normalizeSpaceKey(event.target.value);
  event.target.value = newKey.value;
  keyEdited.value = newKey.value !== '';
  if (!keyEdited.value) {
    requestKeySuggestion(newTitle.value);
  }
};

const itemClass = (active, dragOver) => [
  'w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-semibold text-left cursor-pointer transition-all duration-200 border-2',
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
  if (!canCreate.value) return;
  emit('create', { title: newTitle.value.trim(), key: newKey.value });
  newTitle.value = '';
  newKey.value = '';
  keyEdited.value = false;
};
</script>
