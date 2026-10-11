<template>
  <!-- コメントのセクション。課題の詳細（ページとモーダル）の下部に置き、Markdown で投稿・編集・削除できる -->
  <section>
    <div class="flex items-center gap-2 mb-2">
      <h2 class="m-0 text-sm font-semibold text-gray-700">コメント</h2>
      <span v-if="comments.length" class="text-xs text-gray-500">{{ comments.length }} 件</span>
      <button
        v-if="comments.length > 1"
        type="button"
        class="ml-auto px-2 py-1 bg-transparent border-0 rounded-md text-xs font-semibold text-gray-600 cursor-pointer hover:bg-gray-100"
        :title="newestFirst ? '古い順に並べる' : '新しい順に並べる'"
        @click="toggleOrder"
      >
        {{ newestFirst ? '新しい順' : '古い順' }} ⇅
      </button>
    </div>

    <!-- 入力欄。押すと Markdown の入力欄が開く -->
    <div v-if="composing">
      <MarkdownEditor
        ref="newInput"
        v-model="newBody"
        :rows="4"
        aria-label="コメント"
        placeholder="コメントを追加（Markdown が使えます）"
        @keydown="handleNewKeydown"
      />
      <div class="flex gap-2 mt-2">
        <button
          type="button"
          :disabled="!newBody.trim() || busy"
          class="px-4 py-1.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          @click="createComment"
        >
          保存
        </button>
        <button
          type="button"
          class="px-4 py-1.5 bg-transparent text-gray-700 border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-gray-100"
          @click="cancelNew"
        >
          キャンセル
        </button>
      </div>
    </div>
    <button
      v-else
      type="button"
      class="w-full px-3 py-2 border-2 border-gray-300 rounded-lg bg-white text-left text-sm text-gray-400 cursor-text hover:border-gray-400"
      @click="startNew"
    >
      {{ newBody.trim() ? '書きかけのコメントを続ける...' : 'コメントを追加...' }}
    </button>
    <p v-if="error" class="mt-1 mb-0 text-xs text-red-600">{{ error }}</p>

    <p v-if="loading" class="mt-3 mb-0 text-sm text-gray-400">読み込み中...</p>
    <ul v-else-if="sortedComments.length" class="mt-4 mb-0 p-0 list-none flex flex-col gap-4">
      <li v-for="comment in sortedComments" :key="comment.ID">
        <div class="flex items-center gap-2 text-xs text-gray-500">
          <!-- 投稿日時は相対表示し、ホバーで絶対日時を出す -->
          <time :datetime="comment.CreatedAt" :title="formatDateTime(comment.CreatedAt)">{{ formatRelativeTime(comment.CreatedAt, now) }}</time>
          <span v-if="comment.UpdatedAt" :title="`編集: ${formatDateTime(comment.UpdatedAt)}`">・編集済み</span>
        </div>

        <div v-if="editingId === comment.ID" class="mt-1">
          <MarkdownEditor
            ref="editInput"
            v-model="editBody"
            :rows="4"
            aria-label="コメントを編集"
            @keydown="handleEditKeydown"
          />
          <div class="flex gap-2 mt-2">
            <button
              type="button"
              :disabled="!editBody.trim() || busy"
              class="px-4 py-1.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              @click="saveEdit"
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
        <template v-else>
          <div class="mt-1">
            <MarkdownView :source="comment.Body" />
          </div>
          <div class="mt-1 flex gap-3">
            <button
              type="button"
              class="bg-transparent border-0 p-0 text-xs font-semibold text-gray-500 cursor-pointer hover:text-blue-600 hover:underline"
              @click="startEdit(comment)"
            >
              編集
            </button>
            <button
              type="button"
              class="bg-transparent border-0 p-0 text-xs font-semibold text-gray-500 cursor-pointer hover:text-red-600 hover:underline"
              @click="confirmingDelete = comment"
            >
              削除
            </button>
          </div>
        </template>
      </li>
    </ul>

    <ConfirmDialog
      v-if="confirmingDelete"
      title="コメントを削除しますか？"
      message="このコメントを削除します。この操作は取り消せません。"
      @confirm="deleteComment"
      @cancel="confirmingDelete = null"
    />
  </section>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue';
import MarkdownEditor from './MarkdownEditor.vue';
import MarkdownView from './MarkdownView.vue';
import ConfirmDialog from './ConfirmDialog.vue';
import { commentsApi } from '../api/comments';
import { formatDateTime, formatRelativeTime } from '../utils/datetime';

const props = defineProps({
  // コメントを表示・投稿する課題
  task: {
    type: Object,
    required: true
  }
});

// changed: コメントを投稿・削除した（呼び出し側でカードのコメント数を取り直す）
const emit = defineEmits(['changed']);

// 並び順は閲覧者ごとの好みとしてブラウザに覚えておく（保存できなくても動く）
const ORDER_STORAGE_KEY = 'kanban.commentOrder';
const readOrder = () => {
  try {
    return localStorage.getItem(ORDER_STORAGE_KEY) !== 'oldest';
  } catch {
    return true;
  }
};

const comments = ref([]);
const loading = ref(false);
const busy = ref(false);
const error = ref('');
const newestFirst = ref(true);

const sortedComments = computed(() => (newestFirst.value ? [...comments.value].reverse() : comments.value));

const toggleOrder = () => {
  newestFirst.value = !newestFirst.value;
  try {
    localStorage.setItem(ORDER_STORAGE_KEY, newestFirst.value ? 'newest' : 'oldest');
  } catch {
    // 保存できなくても、この画面の間は切り替えたままにする
  }
};

// 相対表示（「3 分前」）を進めるため、1 分ごとに今の時刻を更新する
const now = ref(Date.now());
let nowTimer = null;

onMounted(() => {
  newestFirst.value = readOrder();
  nowTimer = setInterval(() => {
    now.value = Date.now();
  }, 60000);
});

onUnmounted(() => {
  clearInterval(nowTimer);
});

const fetchComments = async () => {
  const todoId = props.task.ID;
  try {
    error.value = '';
    const result = await commentsApi.fetchByTodo(todoId);
    // 読み込み中に別の課題へ切り替わった場合は捨てる
    if (todoId === props.task.ID) {
      comments.value = result;
      now.value = Date.now();
    }
  } catch (err) {
    error.value = 'コメントの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch comments:', err);
  }
};

// 日本語入力の変換確定の Enter や Esc では保存・取り消しをしない
const isComposing = (event) => event.isComposing || event.keyCode === 229;

// 新しいコメント。キャンセルしても書きかけの本文は残す
const composing = ref(false);
const newBody = ref('');
const newInput = ref(null);

const startNew = async () => {
  composing.value = true;
  await nextTick();
  newInput.value?.focus();
};

const cancelNew = () => {
  composing.value = false;
};

const createComment = async () => {
  if (!newBody.value.trim() || busy.value) return;
  busy.value = true;
  try {
    error.value = '';
    const result = await commentsApi.create(props.task.ID, newBody.value);
    comments.value = [...comments.value, result.comment];
    newBody.value = '';
    composing.value = false;
    emit('changed');
  } catch (err) {
    error.value = 'コメントの投稿に失敗しました: ' + err.message;
    console.error('Failed to create comment:', err);
  } finally {
    busy.value = false;
  }
};

// ⌘+Enter / Ctrl+Enter で保存し、Esc で閉じる
const handleNewKeydown = (event) => {
  if (isComposing(event)) return;
  if (event.key === 'Escape') {
    event.preventDefault();
    cancelNew();
  } else if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
    event.preventDefault();
    createComment();
  }
};

// コメントの編集
const editingId = ref(null);
const editBody = ref('');
const editInput = ref(null);

const startEdit = async (comment) => {
  editingId.value = comment.ID;
  editBody.value = comment.Body;
  await nextTick();
  // v-for の中の ref は配列になる
  const input = Array.isArray(editInput.value) ? editInput.value[0] : editInput.value;
  input?.focus();
};

const cancelEdit = () => {
  editingId.value = null;
};

const saveEdit = async () => {
  const id = editingId.value;
  if (id === null || !editBody.value.trim() || busy.value) return;
  const current = comments.value.find(comment => comment.ID === id);
  if (current && current.Body === editBody.value) {
    editingId.value = null;
    return;
  }
  busy.value = true;
  try {
    error.value = '';
    const result = await commentsApi.update(id, editBody.value);
    comments.value = comments.value.map(comment => (comment.ID === id ? result.comment : comment));
    editingId.value = null;
  } catch (err) {
    error.value = 'コメントの更新に失敗しました: ' + err.message;
    console.error('Failed to update comment:', err);
  } finally {
    busy.value = false;
  }
};

const handleEditKeydown = (event) => {
  if (isComposing(event)) return;
  if (event.key === 'Escape') {
    event.preventDefault();
    cancelEdit();
  } else if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
    event.preventDefault();
    saveEdit();
  }
};

// コメントの削除（確認モーダルで確定する）
const confirmingDelete = ref(null);

const deleteComment = async () => {
  const comment = confirmingDelete.value;
  confirmingDelete.value = null;
  try {
    error.value = '';
    await commentsApi.delete(comment.ID);
    comments.value = comments.value.filter(c => c.ID !== comment.ID);
    if (editingId.value === comment.ID) {
      editingId.value = null;
    }
    emit('changed');
  } catch (err) {
    error.value = 'コメントの削除に失敗しました: ' + err.message;
    console.error('Failed to delete comment:', err);
    await fetchComments();
  }
};

// 別の課題に切り替わったら、入力中の内容を捨てて読み直す
watch(() => props.task.ID, async () => {
  composing.value = false;
  newBody.value = '';
  editingId.value = null;
  confirmingDelete.value = null;
  comments.value = [];
  loading.value = true;
  await fetchComments();
  loading.value = false;
}, { immediate: true });
</script>
