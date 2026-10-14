<template>
  <!-- スペースの設定の「ワークフロー」。ボードの列になるステータスを追加・名前変更・並べ替え・削除する -->
  <section class="bg-surface rounded-2xl px-8 py-6 shadow-xl">
    <h3 class="m-0 mb-1 text-lg font-bold text-fg">ワークフロー</h3>
    <p class="m-0 mb-4 text-sm text-muted">
      ステータスはボードの列になります（上から順に左から並びます）。カテゴリが「完了」のステータスにした課題は完了扱いになります。
      「最大」を入れると、列の課題がその件数を超えたときにボードの列が赤くなります（移動は止めません）。
    </p>

    <p v-if="loading" class="m-0 text-sm text-muted">読み込み中...</p>
    <ol v-else class="m-0 p-0 list-none border border-line rounded-lg divide-y divide-line">
      <li v-for="(status, index) in spaceStatuses" :key="status.ID" class="flex items-center gap-3 px-3 py-2">
        <div class="flex flex-col">
          <button
            type="button"
            class="bg-transparent border-0 p-0 w-6 h-4 leading-none text-muted cursor-pointer hover:text-fg disabled:opacity-30 disabled:cursor-default"
            :disabled="index === 0 || busy"
            :aria-label="`${status.Name} を上へ`"
            @click="move(index, -1)"
          >
            ▲
          </button>
          <button
            type="button"
            class="bg-transparent border-0 p-0 w-6 h-4 leading-none text-muted cursor-pointer hover:text-fg disabled:opacity-30 disabled:cursor-default"
            :disabled="index === spaceStatuses.length - 1 || busy"
            :aria-label="`${status.Name} を下へ`"
            @click="move(index, 1)"
          >
            ▼
          </button>
        </div>

        <input
          v-if="renamingId === status.ID"
          ref="renameInput"
          v-model="renameDraft"
          type="text"
          maxlength="50"
          :aria-label="`${status.Name} の新しい名前`"
          class="flex-1 min-w-0 px-2 py-1 border-2 border-accent rounded-md text-sm text-fg bg-surface focus:outline-none"
          @keydown.enter="handleRenameEnter"
          @keydown.esc="renamingId = null"
          @blur="saveRename"
        />
        <button
          v-else
          type="button"
          class="flex-1 min-w-0 truncate bg-transparent border-0 px-2 py-1 rounded-md text-left text-sm font-semibold text-fg cursor-text hover:bg-raised"
          title="クリックして名前を変更"
          @click="startRename(status)"
        >
          {{ status.Name }}
        </button>

        <span :class="['shrink-0 px-2 py-0.5 rounded text-xs font-semibold', STATUS_CATEGORY_BADGE_CLASS[status.Category]]">
          {{ statusCategoryLabel(status.Category) }}
        </span>
        <span class="shrink-0 w-14 text-right text-xs text-muted">{{ usage(status) }} 件</span>
        <!-- 列の最大件数（WIP 制限）。空欄は制限なし。Enter かフォーカスを外すと保存 -->
        <label class="shrink-0 flex items-center gap-1 text-xs text-muted">
          最大
          <input
            :value="status.WIPLimit ?? ''"
            type="number"
            min="1"
            max="999"
            step="1"
            placeholder="なし"
            :aria-label="`${status.Name} の最大件数`"
            class="w-16 px-2 py-1 border-2 border-line rounded-md text-sm text-fg bg-surface focus:outline-none focus:border-accent"
            @keydown.enter.prevent="$event.target.blur()"
            @change="saveWipLimit(status, $event)"
          />
        </label>
        <button
          type="button"
          class="shrink-0 bg-transparent border-0 p-0 w-6 h-6 flex items-center justify-center rounded text-muted text-lg leading-none cursor-pointer hover:bg-danger/10 hover:text-danger disabled:opacity-30 disabled:cursor-default"
          :disabled="spaceStatuses.length <= 1"
          :title="spaceStatuses.length <= 1 ? '最後のステータスは削除できません' : `${status.Name} を削除`"
          @click="startDelete(status)"
        >
          ×
        </button>
      </li>
    </ol>

    <form class="mt-3 flex flex-wrap items-center gap-2" @submit.prevent="addStatus">
      <input
        v-model="newName"
        type="text"
        maxlength="50"
        placeholder="ステータスを追加"
        aria-label="追加するステータスの名前"
        class="flex-1 min-w-40 px-3 py-1.5 border-2 border-line rounded-lg text-sm text-fg bg-surface focus:outline-none focus:border-accent"
        @keydown.enter="handleAddEnter"
      />
      <select
        v-model="newCategory"
        aria-label="追加するステータスのカテゴリ"
        class="px-2 py-1.5 border-2 border-line rounded-lg text-sm text-fg bg-surface cursor-pointer focus:outline-none focus:border-accent"
      >
        <option v-for="category in STATUS_CATEGORIES" :key="category.value" :value="category.value">{{ category.label }}</option>
      </select>
      <button
        type="submit"
        :disabled="!newName.trim() || busy"
        class="px-4 py-1.5 bg-accent text-on-accent border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed"
      >
        追加
      </button>
    </form>
    <p v-if="error" class="mt-2 mb-0 text-sm text-danger">{{ error }}</p>

    <!-- 削除の確認。使われているステータスは移し先を選ばせる -->
    <div
      v-if="deleting"
      class="fixed inset-0 bg-overlay flex items-center justify-center z-50"
      @click.self="deleting = null"
    >
      <div role="alertdialog" aria-modal="true" aria-labelledby="delete-status-title" class="bg-surface rounded-2xl px-8 pt-7 pb-6 shadow-2xl mx-4 max-w-md w-full">
        <h3 id="delete-status-title" class="m-0 text-lg font-bold text-fg">ステータス「{{ deleting.Name }}」を削除しますか？</h3>
        <template v-if="usage(deleting) > 0">
          <p class="mt-3 mb-2 text-sm text-muted">
            このステータスの課題が {{ usage(deleting) }} 件あります。移し先のステータスを選んでください。
          </p>
          <select
            v-model="moveTo"
            aria-label="移し先のステータス"
            class="w-full px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface cursor-pointer focus:outline-none focus:border-accent"
          >
            <option v-for="status in moveTargets" :key="status.ID" :value="status.ID">
              {{ status.Name }}（{{ statusCategoryLabel(status.Category) }}）
            </option>
          </select>
        </template>
        <p v-else class="mt-3 mb-0 text-sm text-muted">このステータスの課題はありません。</p>
        <div class="flex justify-end gap-3 mt-6">
          <button
            type="button"
            class="py-2 px-5 bg-surface text-fg border-2 border-line rounded-lg text-sm font-semibold cursor-pointer hover:bg-raised"
            @click="deleting = null"
          >
            キャンセル
          </button>
          <button
            type="button"
            :disabled="busy"
            class="py-2 px-5 bg-danger text-on-danger border-2 border-danger rounded-lg text-sm font-semibold cursor-pointer hover:bg-danger/90 disabled:opacity-50"
            @click="deleteStatus"
          >
            削除
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue';
import { statusesApi } from '../api/statuses';
import { STATUS_CATEGORIES, STATUS_CATEGORY_BADGE_CLASS, statusCategoryLabel, statusesOfSpace } from '../utils/statuses';

const props = defineProps({
  space: {
    type: Object,
    required: true
  },
  // すべてのワークフローのステータス
  statuses: {
    type: Array,
    default: () => []
  },
  // すべての課題（ステータスごとの件数を出す）
  todos: {
    type: Array,
    default: () => []
  },
  loading: {
    type: Boolean,
    default: false
  }
});

// changed: ワークフローを変えた（呼び出し側でステータスと課題を取り直す）
const emit = defineEmits(['changed']);

const spaceStatuses = computed(() => statusesOfSpace(props.statuses, props.space.ID));
const usage = (status) => props.todos.filter(todo => todo.StatusID === status.ID).length;

const busy = ref(false);
const error = ref('');

// 失敗したらメッセージを出し、呼び出し側でサーバーの状態に合わせ直す
const run = async (label, action) => {
  if (busy.value) return false;
  busy.value = true;
  error.value = '';
  try {
    await action();
    return true;
  } catch (err) {
    error.value = `${label}に失敗しました: ${err.message}`;
    console.error(`Failed: ${label}`, err);
    return false;
  } finally {
    busy.value = false;
    emit('changed');
  }
};

// 日本語入力の変換確定の Enter では確定しない
const isComposing = (event) => event.isComposing || event.keyCode === 229;

// 追加
const newName = ref('');
const newCategory = ref('InProgress');

const addStatus = async () => {
  const name = newName.value.trim();
  if (!name) return;
  if (await run('ステータスの追加', () => statusesApi.create(props.space.ID, { name, category: newCategory.value }))) {
    newName.value = '';
  }
};

const handleAddEnter = (event) => {
  if (isComposing(event)) event.preventDefault();
};

// 並べ替え（上下のボタンで 1 つずつ動かす）
const move = (index, delta) => {
  const ids = spaceStatuses.value.map(status => status.ID);
  const [id] = ids.splice(index, 1);
  ids.splice(index + delta, 0, id);
  run('ステータスの並べ替え', () => statusesApi.reorder(props.space.ID, ids));
};

// 名前の変更（クリックで入力欄にし、Enter かフォーカスを外すと保存）
const renamingId = ref(null);
const renameDraft = ref('');
const renameInput = ref(null);

const startRename = async (status) => {
  renamingId.value = status.ID;
  renameDraft.value = status.Name;
  await nextTick();
  const input = Array.isArray(renameInput.value) ? renameInput.value[0] : renameInput.value;
  input?.focus();
  input?.select();
};

const saveRename = () => {
  const id = renamingId.value;
  if (id === null) return;
  renamingId.value = null;
  const status = spaceStatuses.value.find(s => s.ID === id);
  const name = renameDraft.value.trim();
  if (!status || !name || name === status.Name) return;
  run('ステータスの名前の変更', () => statusesApi.rename(id, name));
};

const handleRenameEnter = (event) => {
  if (isComposing(event)) return;
  event.preventDefault();
  saveRename();
};

// 列の最大件数。空欄か 0 で制限なしに戻す。整数でない値や範囲外は保存せず元に戻す
const saveWipLimit = async (status, event) => {
  const text = event.target.value.trim();
  const limit = text === '' ? null : Number(text);
  if (limit !== null && (!Number.isInteger(limit) || limit < 0 || limit > 999)) {
    error.value = '最大件数は 1〜999 の整数で入力してください';
    event.target.value = status.WIPLimit ?? '';
    return;
  }
  const next = limit === 0 ? null : limit;
  if (next === (status.WIPLimit ?? null)) {
    event.target.value = status.WIPLimit ?? '';
    return;
  }
  await run('最大件数の変更', () => statusesApi.setWipLimit(status.ID, next));
};

// 削除（使われていれば移し先を選ぶ。既定は同じカテゴリの別のステータス）
const deleting = ref(null);
const moveTo = ref(0);
const moveTargets = computed(() => spaceStatuses.value.filter(status => status.ID !== deleting.value?.ID));

const startDelete = (status) => {
  deleting.value = status;
  const targets = spaceStatuses.value.filter(s => s.ID !== status.ID);
  moveTo.value = (targets.find(s => s.Category === status.Category) ?? targets[0])?.ID ?? 0;
};

const deleteStatus = async () => {
  const status = deleting.value;
  const target = usage(status) > 0 ? moveTo.value : 0;
  if (await run('ステータスの削除', () => statusesApi.delete(status.ID, target))) {
    deleting.value = null;
  }
};
</script>
