<template>
  <!-- 時間管理（Jira のタイムトラッキング）。「記録済み / 残り」の進捗バー、初期見積もり・残り時間の編集、「作業を記録」を置く -->
  <section class="border border-gray-200 rounded-lg">
    <div class="flex items-center px-4 py-3 border-b border-gray-200">
      <h2 class="m-0 text-sm font-semibold text-gray-700">時間管理</h2>
      <button
        type="button"
        class="ml-auto px-3 py-1 bg-gray-100 text-gray-800 border-0 rounded-md text-xs font-semibold cursor-pointer hover:bg-gray-200"
        @click="logging = true"
      >
        作業を記録
      </button>
    </div>
    <div class="px-4 py-3">
      <div
        class="h-2 rounded-full bg-gray-200 overflow-hidden"
        role="progressbar"
        :aria-valuenow="spent"
        aria-valuemin="0"
        :aria-valuemax="total"
        :aria-label="`記録済み ${formatDuration(spent)} / 残り ${formatDuration(remaining)}`"
      >
        <div class="h-full bg-blue-600 transition-all duration-300" :style="{ width: `${spentPercent}%` }"></div>
      </div>
      <div class="mt-1 flex justify-between text-xs text-gray-600">
        <span>{{ spent ? `${formatDuration(spent)} 記録済み` : '記録なし' }}</span>
        <span>{{ hasRemaining ? `${formatDuration(remaining)} 残り` : '見積もりなし' }}</span>
      </div>

      <dl class="mt-3 mb-0 grid grid-cols-[6rem_minmax(0,1fr)] gap-x-3 gap-y-2 items-center text-sm">
        <template v-for="field in fields" :key="field.name">
          <dt class="text-gray-600 font-semibold"><label :for="inputId(field.name)">{{ field.label }}</label></dt>
          <dd class="m-0 min-w-0">
            <template v-if="editing === field.name">
              <input
                :id="inputId(field.name)"
                :ref="(el) => { if (el) inputEl = el; }"
                v-model="draft"
                type="text"
                placeholder="例: 2d 4h（空で未設定）"
                class="w-full px-3 py-1.5 border-2 border-blue-500 rounded-lg text-sm text-gray-900 bg-white focus:outline-none"
                @keydown.enter="handleEnter"
                @keydown.esc="handleEsc"
                @blur="saveEdit"
              />
              <p v-if="error" class="mt-1 mb-0 text-xs text-red-600">{{ error }}</p>
            </template>
            <button
              v-else
              type="button"
              class="-mx-2 px-2 py-1.5 bg-transparent border-0 rounded-md text-left text-sm cursor-pointer hover:bg-gray-100"
              title="クリックして編集"
              @click="startEdit(field.name)"
            >
              <span v-if="task[field.key] != null" class="text-gray-900">{{ formatDuration(task[field.key]) }}</span>
              <span v-else class="text-gray-400">なし</span>
            </button>
          </dd>
        </template>
      </dl>
    </div>

    <WorklogDialog
      v-if="logging"
      :task="task"
      @saved="handleLogged"
      @cancel="logging = false"
    />
  </section>
</template>

<script setup>
import { ref, computed, nextTick, watch } from 'vue';
import WorklogDialog from './WorklogDialog.vue';
import { formatDuration, parseDuration } from '../utils/duration';

const props = defineProps({
  task: {
    type: Object,
    required: true
  }
});

// update: 見積もりの変更（PUT /todos/:id のリクエストに重ねる項目）。保存は呼び出し側で行う
// worklogs-changed: 作業を記録した（呼び出し側で課題を取り直す）
const emit = defineEmits(['update', 'worklogs-changed']);

const fields = [
  { name: 'originalEstimate', key: 'OriginalEstimate', label: '初期見積もり' },
  { name: 'remainingEstimate', key: 'RemainingEstimate', label: '残り時間' }
];

const spent = computed(() => props.task.TimeSpent || 0);
const hasRemaining = computed(() => props.task.RemainingEstimate != null || props.task.OriginalEstimate != null);
// 残り時間が未設定でも初期見積もりがあれば、bff の自動調整と同じく「初期見積もり − 記録済み」とみなす
const remaining = computed(() => {
  if (props.task.RemainingEstimate != null) return props.task.RemainingEstimate;
  if (props.task.OriginalEstimate != null) return Math.max(props.task.OriginalEstimate - spent.value, 0);
  return 0;
});
const total = computed(() => spent.value + remaining.value);
const spentPercent = computed(() => (total.value ? Math.round((spent.value / total.value) * 100) : 0));

// ページとモーダルが同時にあっても label の for が重ならないよう、課題ごとに ID を分ける
const inputId = (name) => `issue-${props.task.ID}-${name}`;

const editing = ref(null);
const draft = ref('');
const error = ref('');
let inputEl = null;

const startEdit = async (name) => {
  const field = fields.find(f => f.name === name);
  const value = props.task[field.key];
  editing.value = name;
  draft.value = value == null ? '' : formatDuration(value);
  error.value = '';
  await nextTick();
  inputEl?.focus();
};

// 空なら未設定に戻す。読めない値なら編集を続けてエラーを出す
const saveEdit = () => {
  const name = editing.value;
  if (!name) return;
  const field = fields.find(f => f.name === name);
  const { value, error: parseError } = parseDuration(draft.value);
  if (parseError) {
    error.value = parseError;
    return;
  }
  editing.value = null;
  if (value !== (props.task[field.key] ?? null)) {
    emit('update', { [name]: value == null ? null : draft.value });
  }
};

// 日本語入力の変換確定の Enter や Esc では保存・取り消しをしない
const isComposing = (event) => event.isComposing || event.keyCode === 229;

const handleEnter = (event) => {
  if (isComposing(event)) return;
  event.preventDefault();
  saveEdit();
};

const handleEsc = (event) => {
  if (isComposing(event)) return;
  event.preventDefault();
  editing.value = null;
};

const logging = ref(false);

const handleLogged = () => {
  logging.value = false;
  emit('worklogs-changed');
};

// 別の課題に切り替わったら編集をやめる
watch(() => props.task.ID, () => {
  editing.value = null;
  logging.value = false;
});
</script>
