<template>
  <!-- アクティビティのセクション。課題の詳細（ページとモーダル）の下部に置く。
       「すべて / コメント / 作業ログ / 履歴」のタブで、コメント・作業ログ・変更履歴を時系列に表示する。コメントは Markdown で投稿・編集・削除できる -->
  <section>
    <div class="flex items-center gap-2 mb-2">
      <h2 class="m-0 text-sm font-semibold text-gray-700">アクティビティ</h2>
      <button
        v-if="items.length > 1"
        type="button"
        class="ml-auto px-2 py-1 bg-transparent border-0 rounded-md text-xs font-semibold text-gray-600 cursor-pointer hover:bg-gray-100"
        :title="newestFirst ? '古い順に並べる' : '新しい順に並べる'"
        @click="toggleOrder"
      >
        {{ newestFirst ? '新しい順' : '古い順' }} ⇅
      </button>
    </div>

    <div class="flex gap-1 mb-3" role="tablist" aria-label="アクティビティの表示">
      <button
        v-for="option in tabs"
        :key="option.value"
        type="button"
        role="tab"
        :aria-selected="tab === option.value"
        :class="[
          'px-3 py-1 border-0 rounded-md text-xs font-semibold cursor-pointer',
          tab === option.value ? 'bg-blue-50 text-blue-700' : 'bg-transparent text-gray-500 hover:bg-gray-100'
        ]"
        @click="tab = option.value"
      >
        {{ option.label }}
        <span v-if="option.count" class="ml-0.5 font-normal">{{ option.count }}</span>
      </button>
    </div>

    <!-- コメントの入力欄（すべて・コメントのタブ）。押すと Markdown の入力欄が開く -->
    <template v-if="tab === 'all' || tab === 'comments'">
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
    </template>
    <p v-if="error" class="mt-1 mb-0 text-xs text-red-600">{{ error }}</p>

    <p v-if="loading" class="mt-3 mb-0 text-sm text-gray-400">読み込み中...</p>
    <ul v-else-if="visibleItems.length" class="mt-4 mb-0 p-0 list-none flex flex-col gap-4">
      <li v-for="item in visibleItems" :key="item.key">
        <!-- コメント -->
        <template v-if="item.kind === 'comment'">
          <div class="flex items-center gap-2 text-xs text-gray-500">
            <!-- 投稿者（ユーザー機能の追加前のコメントは投稿者なし） -->
            <template v-if="item.comment.Author">
              <UserAvatar :user="item.comment.Author" size="sm" />
              <span class="font-semibold text-gray-800">{{ item.comment.Author.DisplayName }}</span>
            </template>
            <span v-else class="font-semibold text-gray-600">コメント</span>
            <!-- 日時は相対表示し、ホバーで絶対日時を出す -->
            <time :datetime="item.comment.CreatedAt" :title="formatDateTime(item.comment.CreatedAt)">{{ formatRelativeTime(item.comment.CreatedAt, now) }}</time>
            <span v-if="item.comment.UpdatedAt" :title="`編集: ${formatDateTime(item.comment.UpdatedAt)}`">・編集済み</span>
          </div>

          <div v-if="editingId === item.comment.ID" class="mt-1">
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
              <MarkdownView :source="item.comment.Body" />
            </div>
            <div class="mt-1 flex gap-3">
              <button
                type="button"
                class="bg-transparent border-0 p-0 text-xs font-semibold text-gray-500 cursor-pointer hover:text-blue-600 hover:underline"
                @click="startEdit(item.comment)"
              >
                編集
              </button>
              <button
                type="button"
                class="bg-transparent border-0 p-0 text-xs font-semibold text-gray-500 cursor-pointer hover:text-red-600 hover:underline"
                @click="confirmingDelete = item.comment"
              >
                削除
              </button>
            </div>
          </template>
        </template>

        <!-- 作業ログ -->
        <template v-else-if="item.kind === 'worklog'">
          <div class="flex flex-wrap items-center gap-x-2 text-xs text-gray-500">
            <span class="font-semibold text-gray-600">作業を記録しました</span>
            <time :datetime="item.worklog.CreatedAt" :title="formatDateTime(item.worklog.CreatedAt)">{{ formatRelativeTime(item.worklog.CreatedAt, now) }}</time>
            <span v-if="item.worklog.UpdatedAt" :title="`編集: ${formatDateTime(item.worklog.UpdatedAt)}`">・編集済み</span>
          </div>
          <p class="mt-1 mb-0 text-sm text-gray-900">
            <span class="font-semibold">{{ formatDuration(item.worklog.Minutes) }}</span>
            <span class="ml-2 text-xs text-gray-500">開始 {{ formatDateTime(item.worklog.StartedAt) }}</span>
          </p>
          <p v-if="item.worklog.Note" class="mt-1 mb-0 text-sm text-gray-700 whitespace-pre-wrap break-words">{{ item.worklog.Note }}</p>
          <div class="mt-1 flex gap-3">
            <button
              type="button"
              class="bg-transparent border-0 p-0 text-xs font-semibold text-gray-500 cursor-pointer hover:text-blue-600 hover:underline"
              @click="editingWorklog = item.worklog"
            >
              編集
            </button>
            <button
              type="button"
              class="bg-transparent border-0 p-0 text-xs font-semibold text-gray-500 cursor-pointer hover:text-red-600 hover:underline"
              @click="confirmingWorklogDelete = item.worklog"
            >
              削除
            </button>
          </div>
        </template>

        <!-- 変更履歴（同じ操作で変わった項目はまとめて出す） -->
        <template v-else>
          <div class="flex items-center gap-2 text-xs text-gray-500">
            <!-- 変更したユーザー（ユーザー機能の追加前の記録は変更者なし） -->
            <template v-if="item.user">
              <UserAvatar :user="item.user" size="sm" />
              <span class="font-semibold text-gray-800">{{ item.user.DisplayName }}</span>
            </template>
            <span class="font-semibold text-gray-600">{{ item.created ? '課題を作成しました' : '変更しました' }}</span>
            <time :datetime="item.time" :title="formatDateTime(item.time)">{{ formatRelativeTime(item.time, now) }}</time>
          </div>
          <ul v-if="item.changes.length" class="mt-1 mb-0 p-0 list-none flex flex-col gap-1">
            <li v-for="entry in item.changes" :key="entry.ID" class="flex flex-wrap items-baseline gap-x-2 text-sm">
              <span class="shrink-0 text-gray-600 font-semibold">{{ historyFieldLabel(entry.Field) }}</span>
              <span
                :class="['min-w-0 break-words line-clamp-3 whitespace-pre-wrap', entry.OldValue == null ? 'text-gray-400' : 'text-gray-500 line-through']"
              >{{ formatHistoryValue(entry.Field, entry.OldValue) }}</span>
              <span class="text-gray-400" aria-label="から">→</span>
              <span
                :class="['min-w-0 break-words line-clamp-3 whitespace-pre-wrap', entry.NewValue == null ? 'text-gray-400' : 'text-gray-900']"
              >{{ formatHistoryValue(entry.Field, entry.NewValue) }}</span>
            </li>
          </ul>
        </template>
      </li>
    </ul>
    <p v-else class="mt-3 mb-0 text-sm text-gray-400">
      {{ { all: 'アクティビティはありません', comments: 'コメントはありません', worklogs: '作業ログはありません', history: '変更履歴はありません' }[tab] }}
    </p>

    <ConfirmDialog
      v-if="confirmingDelete"
      title="コメントを削除しますか？"
      message="このコメントを削除します。この操作は取り消せません。"
      @confirm="deleteComment"
      @cancel="confirmingDelete = null"
    />
    <ConfirmDialog
      v-if="confirmingWorklogDelete"
      title="作業ログを削除しますか？"
      :message="`${formatDuration(confirmingWorklogDelete.Minutes)} の作業ログを削除します。残り時間はその分だけ戻ります。`"
      @confirm="deleteWorklog"
      @cancel="confirmingWorklogDelete = null"
    />
    <WorklogDialog
      v-if="editingWorklog"
      :task="task"
      :worklog="editingWorklog"
      @saved="handleWorklogSaved"
      @cancel="editingWorklog = null"
    />
  </section>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue';
import MarkdownEditor from './MarkdownEditor.vue';
import MarkdownView from './MarkdownView.vue';
import UserAvatar from './UserAvatar.vue';
import ConfirmDialog from './ConfirmDialog.vue';
import WorklogDialog from './WorklogDialog.vue';
import { commentsApi } from '../api/comments';
import { worklogsApi } from '../api/worklogs';
import { formatDuration } from '../utils/duration';
import { todosApi } from '../api/todos';
import { formatDateTime, formatRelativeTime } from '../utils/datetime';
import { historyFieldLabel, formatHistoryValue } from '../utils/history';

const props = defineProps({
  // アクティビティを表示する課題。内容が変わる（別のオブジェクトになる）たびに変更履歴を読み直す
  task: {
    type: Object,
    required: true
  }
});

// changed: コメントを投稿・削除した（呼び出し側でカードのコメント数を取り直す）
// worklogs-changed: 作業ログを編集・削除した（呼び出し側で残り時間などを取り直す）
const emit = defineEmits(['changed', 'worklogs-changed']);

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
const history = ref([]);
const worklogs = ref([]);
const loading = ref(false);
const busy = ref(false);
const error = ref('');
const newestFirst = ref(true);
const tab = ref('all');

const tabs = computed(() => [
  { value: 'all', label: 'すべて' },
  { value: 'comments', label: 'コメント', count: comments.value.length },
  { value: 'worklogs', label: '作業ログ', count: worklogs.value.length },
  { value: 'history', label: '履歴' }
]);

// 同じ日時・同じユーザーの操作で記録された変更履歴を 1 件にまとめる（作成は単独で出す）
const historyItems = computed(() => {
  const groups = [];
  for (const entry of history.value) {
    const created = entry.Field === 'created';
    const last = groups[groups.length - 1];
    if (!created && last && !last.created && last.time === entry.CreatedAt && last.userId === entry.UserID) {
      last.changes.push(entry);
    } else {
      groups.push({
        kind: 'history', key: `history-${entry.ID}`, time: entry.CreatedAt, id: entry.ID, created,
        userId: entry.UserID, user: entry.User, changes: created ? [] : [entry]
      });
    }
  }
  return groups;
});

const commentItems = computed(() =>
  comments.value.map(comment => ({ kind: 'comment', key: `comment-${comment.ID}`, time: comment.CreatedAt, id: comment.ID, comment }))
);

// 作業ログは記録した日時で並べる（開始日時は本文に出す）
const worklogItems = computed(() =>
  worklogs.value.map(worklog => ({ kind: 'worklog', key: `worklog-${worklog.ID}`, time: worklog.CreatedAt, id: worklog.ID, worklog }))
);

// 古い順に並べた全件。同じ日時なら変更履歴を先にする
const items = computed(() =>
  [...historyItems.value, ...commentItems.value, ...worklogItems.value].sort((a, b) => {
    const diff = new Date(a.time) - new Date(b.time);
    if (diff !== 0) return diff;
    if (a.kind !== b.kind) return a.kind === 'history' ? -1 : 1;
    return a.id - b.id;
  })
);

const visibleItems = computed(() => {
  const filtered = items.value.filter(item =>
    tab.value === 'all' || item.kind === { comments: 'comment', worklogs: 'worklog', history: 'history' }[tab.value]
  );
  return newestFirst.value ? filtered.reverse() : filtered;
});

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

const fetchHistory = async () => {
  const todoId = props.task.ID;
  try {
    const result = await todosApi.fetchHistory(todoId);
    if (todoId === props.task.ID) {
      history.value = result;
      now.value = Date.now();
    }
  } catch (err) {
    error.value = '変更履歴の取得に失敗しました: ' + err.message;
    console.error('Failed to fetch history:', err);
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

const fetchWorklogs = async () => {
  const todoId = props.task.ID;
  try {
    const result = await worklogsApi.fetchByTodo(todoId);
    if (todoId === props.task.ID) {
      worklogs.value = result;
    }
  } catch (err) {
    error.value = '作業ログの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch worklogs:', err);
  }
};

// 作業ログの編集・削除。残り時間が変わるので、呼び出し側に課題を取り直してもらう
const editingWorklog = ref(null);
const confirmingWorklogDelete = ref(null);

const handleWorklogSaved = () => {
  editingWorklog.value = null;
  emit('worklogs-changed');
};

const deleteWorklog = async () => {
  const worklog = confirmingWorklogDelete.value;
  confirmingWorklogDelete.value = null;
  try {
    error.value = '';
    await worklogsApi.delete(worklog.ID);
    worklogs.value = worklogs.value.filter(w => w.ID !== worklog.ID);
    emit('worklogs-changed');
  } catch (err) {
    error.value = '作業ログの削除に失敗しました: ' + err.message;
    console.error('Failed to delete worklog:', err);
    await fetchWorklogs();
  }
};

// 別の課題に切り替わったら、入力中の内容を捨てて読み直す。
// 同じ課題の内容が変わった（保存した・作業を記録した）ときは、変更履歴と作業ログを読み直す
watch(() => props.task, async (task, previous) => {
  if (previous && previous.ID === task.ID) {
    await Promise.all([fetchHistory(), fetchWorklogs()]);
    return;
  }
  composing.value = false;
  newBody.value = '';
  editingId.value = null;
  confirmingDelete.value = null;
  editingWorklog.value = null;
  confirmingWorklogDelete.value = null;
  comments.value = [];
  history.value = [];
  worklogs.value = [];
  loading.value = true;
  await Promise.all([fetchComments(), fetchHistory(), fetchWorklogs()]);
  loading.value = false;
}, { immediate: true });
</script>
