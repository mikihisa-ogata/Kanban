<template>
  <!-- 課題の詳細。全画面のページ（/browse/:key）と詳細モーダルの両方で使う。
       Jira と同じく左に要約・説明、右にステータスなどの詳細欄を置き、各項目はその場で編集する -->
  <div class="text-gray-900">
    <div class="flex items-center gap-2 min-h-8">
      <nav class="flex items-center gap-1.5 min-w-0 text-sm text-gray-500">
        <span v-if="spaceTitle" class="truncate">{{ spaceTitle }}</span>
        <span v-if="spaceTitle" aria-hidden="true">/</span>
        <NuxtLink :to="issuePath(task.Key)" class="font-mono font-semibold text-gray-600 no-underline hover:text-blue-600 hover:underline">
          {{ task.Key }}
        </NuxtLink>
      </nav>
      <button
        type="button"
        class="shrink-0 bg-transparent border-0 p-1 rounded text-gray-400 cursor-pointer hover:bg-gray-100 hover:text-gray-600"
        :title="copied ? 'コピーしました' : '課題キーをコピー'"
        @click="copyKey"
      >
        <svg v-if="!copied" class="w-4 h-4 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <rect x="9" y="9" width="12" height="12" rx="2" />
          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
        </svg>
        <svg v-else class="w-4 h-4 block text-emerald-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M5 13l4 4L19 7" />
        </svg>
      </button>
      <span v-if="copied" class="text-xs font-semibold text-emerald-600">コピーしました</span>
      <div class="ml-auto flex items-center gap-1">
        <slot name="actions" />
      </div>
    </div>

    <div class="mt-4 flex flex-col md:flex-row gap-8">
      <!-- 左: 要約と説明（子課題・リンク・コメント・履歴は以降のチケットでここに追加する） -->
      <div class="flex-1 min-w-0">
        <input
          v-if="editing === 'title'"
          ref="titleInput"
          v-model="draft"
          type="text"
          aria-label="要約"
          class="w-full -mx-2 px-2 py-1 border-2 border-blue-500 rounded-md text-2xl font-semibold text-gray-900 bg-white focus:outline-none"
          @keydown.enter="handleTitleEnter"
          @keydown.esc="handleEsc"
          @blur="saveTitle"
        />
        <h1
          v-else
          class="m-0 -mx-2 px-2 py-1 border-2 border-transparent rounded-md text-2xl font-semibold leading-snug break-words cursor-text hover:bg-gray-100"
          title="クリックして編集"
          @click="startEdit('title', task.Title)"
        >
          {{ task.Title }}
        </h1>

        <h2 class="mt-6 mb-2 text-sm font-semibold text-gray-700">説明</h2>
        <div v-if="editing === 'description'">
          <textarea
            ref="descriptionInput"
            v-model="draft"
            rows="6"
            aria-label="説明"
            placeholder="説明を入力"
            class="block w-full px-3 py-2.5 border-2 border-blue-500 rounded-lg text-sm text-gray-900 bg-white resize-y focus:outline-none"
            @keydown.esc="handleEsc"
            @keydown.enter.meta="handleDescriptionEnter"
            @keydown.enter.ctrl="handleDescriptionEnter"
          ></textarea>
          <div class="flex gap-2 mt-2">
            <button
              type="button"
              class="px-4 py-1.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700"
              @click="saveDescription"
            >
              保存
            </button>
            <button
              type="button"
              class="px-4 py-1.5 bg-transparent text-gray-700 border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-gray-100"
              @click="cancelEdit"
            >
              キャンセル
            </button>
          </div>
        </div>
        <div
          v-else
          class="-mx-2 px-2 py-1.5 rounded-md text-sm leading-relaxed whitespace-pre-wrap break-words cursor-text hover:bg-gray-100"
          title="クリックして編集"
          @click="startEdit('description', task.Description)"
        >
          <template v-if="task.Description">{{ task.Description }}</template>
          <span v-else class="text-gray-400">説明を追加...</span>
        </div>
      </div>

      <!-- 右: 詳細欄 -->
      <aside class="md:w-80 shrink-0">
        <section class="border border-gray-200 rounded-lg">
          <h2 class="m-0 px-4 py-3 border-b border-gray-200 text-sm font-semibold text-gray-700">詳細</h2>
          <dl class="m-0 px-4 py-3 grid grid-cols-[6rem_minmax(0,1fr)] gap-x-3 gap-y-3 items-center text-sm">
            <dt class="text-gray-600 font-semibold"><label :for="fieldId('status')">ステータス</label></dt>
            <dd class="m-0 min-w-0">
              <OptionSelect
                :id="fieldId('status')"
                :model-value="task.Status"
                :options="statusOptions"
                @update:model-value="save({ status: $event })"
              />
            </dd>

            <template v-if="spaces.length">
              <dt class="text-gray-600 font-semibold"><label :for="fieldId('space')">スペース</label></dt>
              <dd class="m-0 min-w-0">
                <OptionSelect
                  :id="fieldId('space')"
                  :model-value="task.SpaceID"
                  :options="spaceOptions"
                  :empty-value="0"
                  @update:model-value="changeSpace"
                />
              </dd>
            </template>

            <dt class="text-gray-600 font-semibold"><label :for="fieldId('epic')">エピック</label></dt>
            <dd class="m-0 min-w-0">
              <OptionSelect
                :id="fieldId('epic')"
                :model-value="task.EpicID"
                :options="epicOptions"
                :empty-value="0"
                @update:model-value="save({ epicId: $event })"
              />
            </dd>

            <dt class="text-gray-600 font-semibold"><label :for="fieldId('deadline')">期限</label></dt>
            <dd class="m-0 min-w-0">
              <input
                :id="fieldId('deadline')"
                :value="task.Deadline"
                type="date"
                class="px-3 py-2 border-2 border-gray-300 rounded-lg text-sm text-gray-900 bg-white hover:border-gray-400 focus:outline-none focus:border-blue-500"
                @change="save({ deadline: $event.target.value })"
              />
            </dd>
          </dl>
        </section>

        <dl class="mt-3 mb-0 px-1 flex flex-col gap-1 text-xs text-gray-500">
          <div v-if="task.CreatedAt" class="flex gap-1">
            <dt>作成</dt>
            <dd class="m-0">{{ formatDateTime(task.CreatedAt) }}</dd>
          </div>
          <div v-if="task.UpdatedAt" class="flex gap-1">
            <dt>更新</dt>
            <dd class="m-0">{{ formatDateTime(task.UpdatedAt) }}</dd>
          </div>
        </dl>
      </aside>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onUnmounted } from 'vue';
import OptionSelect from './OptionSelect.vue';
import { formatDateTime } from '../utils/datetime';
import { issuePath } from '../utils/spaceRoutes';

const props = defineProps({
  task: {
    type: Object,
    required: true
  },
  epics: {
    type: Array,
    default: () => []
  },
  spaces: {
    type: Array,
    default: () => []
  }
});

// update: 変更後の内容（PUT /todos/:id のリクエストの形）。保存は呼び出し側で行う
const emit = defineEmits(['update']);

const statusOptions = [
  { value: 'Open', label: 'オープン' },
  { value: 'InProgress', label: '進行中' },
  { value: 'Waiting', label: '待ち状況' },
  { value: 'Closed', label: 'クローズ' }
];

const spaceTitle = computed(() => props.spaces.find(space => space.ID === props.task.SpaceID)?.Title ?? '');

const spaceOptions = computed(() => [
  { value: 0, label: 'なし' },
  ...props.spaces.map(space => ({ value: space.ID, label: space.Title }))
]);

// 課題と同じスペースのエピックだけを選べる
const epicOptions = computed(() => [
  { value: 0, label: 'なし' },
  ...props.epics
    .filter(epic => epic.SpaceID === props.task.SpaceID)
    .map(epic => ({ value: epic.ID, label: epic.Title }))
]);

// ページとモーダルが同時にあっても label の for が重ならないよう、課題ごとに ID を分ける
const fieldId = (name) => `issue-${props.task.ID}-${name}`;

// 今の内容に changes を重ねて保存を依頼する
const save = (changes) => {
  emit('update', {
    title: props.task.Title,
    description: props.task.Description,
    done: props.task.Done,
    deadline: props.task.Deadline,
    status: props.task.Status,
    epicId: props.task.EpicID,
    spaceId: props.task.SpaceID,
    ...changes
  });
};

// エピックは移動元のスペースに属するので、スペースを変えたら外す
const changeSpace = (spaceId) => {
  save({ spaceId, epicId: 0 });
};

// インライン編集中の項目（'title' | 'description' | null）と入力中の値
const editing = ref(null);
const draft = ref('');
const titleInput = ref(null);
const descriptionInput = ref(null);

const startEdit = async (field, value) => {
  editing.value = field;
  draft.value = value;
  await nextTick();
  (field === 'title' ? titleInput : descriptionInput).value?.focus();
};

const cancelEdit = () => {
  editing.value = null;
};

// 日本語入力の変換確定の Enter や Esc では保存・取り消しをしない
const isComposing = (event) => event.isComposing || event.keyCode === 229;

const handleEsc = (event) => {
  if (isComposing(event)) return;
  event.preventDefault();
  cancelEdit();
};

const handleTitleEnter = (event) => {
  if (isComposing(event)) return;
  event.preventDefault();
  saveTitle();
};

// 説明は改行できるよう、⌘+Enter / Ctrl+Enter で保存する
const handleDescriptionEnter = (event) => {
  if (isComposing(event)) return;
  event.preventDefault();
  saveDescription();
};

// 要約は空にできない。空や変更なしなら元に戻すだけ
const saveTitle = () => {
  if (editing.value !== 'title') return;
  const title = draft.value.trim();
  editing.value = null;
  if (title && title !== props.task.Title) {
    save({ title });
  }
};

const saveDescription = () => {
  if (editing.value !== 'description') return;
  editing.value = null;
  if (draft.value !== props.task.Description) {
    save({ description: draft.value });
  }
};

// 別の課題に切り替わったら編集をやめる
watch(() => props.task.ID, () => {
  editing.value = null;
});

const copied = ref(false);
let copiedTimer = null;

const copyKey = async () => {
  try {
    await navigator.clipboard.writeText(props.task.Key);
    copied.value = true;
    clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      copied.value = false;
    }, 2000);
  } catch (err) {
    console.error('Failed to copy issue key:', err);
  }
};

onUnmounted(() => {
  clearTimeout(copiedTimer);
});
</script>
