<template>
  <!-- 複数選択した課題の操作（Jira の一括変更）。選んだ件数と「一括変更」「削除」「選択を解除」を並べ、変更・削除のダイアログも持つ -->
  <div v-if="issues.length" v-bind="$attrs" class="flex flex-wrap items-center gap-2">
    <span class="text-sm font-semibold text-fg">{{ issues.length }} 件を選択中</span>
    <button type="button" :class="buttonClass" :disabled="loading" @click="openEdit">一括変更</button>
    <button type="button" :class="[buttonClass, 'text-danger']" @click="openDelete">削除</button>
    <button type="button" :class="buttonClass" @click="$emit('clear')">選択を解除</button>
  </div>

  <BulkEditDialog
    v-if="editing"
    :issues="issues"
    :statuses="masters.statuses"
    :users="masters.users"
    :epics="masters.epics"
    :sprints="masters.sprints"
    :versions="masters.versions"
    :busy="busy"
    :error="dialogError"
    @submit="handleUpdate"
    @cancel="editing = false"
  />
  <ConfirmDialog
    v-if="confirmingDelete"
    :title="`${issues.length} 件の課題を削除しますか？`"
    :message="deleteMessage"
    :confirm-label="`${issues.length} 件を削除`"
    @confirm="handleDelete"
    @cancel="confirmingDelete = false"
  />

  <div v-if="notice" class="fixed bottom-8 left-1/2 -translate-x-1/2 z-50 bg-surface text-fg px-5 py-3 rounded-xl shadow-xl text-sm font-semibold border border-line animate-slideIn" role="status">
    <span class="text-success" aria-hidden="true">✓</span> {{ notice }}
  </div>
  <div v-if="error" class="fixed bottom-8 right-8 z-50 bg-danger/10 text-danger px-6 py-4 rounded-xl shadow-xl max-w-96 text-sm font-semibold border-l-4 border-danger animate-slideIn">
    {{ error }}
  </div>
</template>

<script setup>
import { ref, reactive, computed, onUnmounted } from 'vue';
import BulkEditDialog from './BulkEditDialog.vue';
import ConfirmDialog from './ConfirmDialog.vue';
import { todosApi } from '../api/todos';
import { statusesApi } from '../api/statuses';
import { usersApi } from '../api/users';
import { epicsApi } from '../api/epics';
import { sprintsApi } from '../api/sprints';
import { versionsApi } from '../api/versions';

// ダイアログと通知も持つので、呼び出し側の class などは操作の並びにだけ付ける
defineOptions({ inheritAttrs: false });

const props = defineProps({
  // 選んでいる課題。空なら何も出さない（変更・削除の結果の通知は出す）
  issues: {
    type: Array,
    required: true
  }
});

// changed: 課題を変更・削除した（呼び出し側で一覧を取り直す）、clear: 選択を解除する
const emit = defineEmits(['changed', 'clear']);

const buttonClass = 'px-2.5 py-1 border-0 rounded-md bg-surface text-sm font-semibold text-fg cursor-pointer hover:bg-raised disabled:opacity-50 disabled:cursor-not-allowed';

// 選択肢（ステータス・ユーザー・エピック・スプリント・バージョン）は開くたびに読み直す
const masters = reactive({ statuses: [], users: [], epics: [], sprints: [], versions: [] });
const loading = ref(false);
const editing = ref(false);
const busy = ref(false);
const dialogError = ref('');
const confirmingDelete = ref(false);
const error = ref('');
const notice = ref('');
let noticeTimer = null;

const showNotice = (text) => {
  clearTimeout(noticeTimer);
  notice.value = text;
  noticeTimer = setTimeout(() => {
    notice.value = '';
  }, 4000);
};

const openEdit = async () => {
  loading.value = true;
  error.value = '';
  try {
    [masters.statuses, masters.users, masters.epics, masters.sprints, masters.versions] = await Promise.all([
      statusesApi.fetchAll(), usersApi.fetchAll(), epicsApi.fetchAll(), sprintsApi.fetchAll(), versionsApi.fetchAll()
    ]);
    dialogError.value = '';
    editing.value = true;
  } catch (err) {
    error.value = '一括変更の選択肢の取得に失敗しました: ' + err.message;
    console.error('Failed to load bulk edit options:', err);
  } finally {
    loading.value = false;
  }
};

const handleUpdate = async (changes) => {
  busy.value = true;
  dialogError.value = '';
  try {
    const { count } = await todosApi.bulkUpdate({ ids: props.issues.map(issue => issue.ID), ...changes });
    editing.value = false;
    showNotice(count ? `${count} 件の課題を変更しました` : '変わった課題はありませんでした');
    emit('changed');
  } catch (err) {
    // 1 件でも変えられなければどれも変わらないので、ダイアログを開いたまま理由を出す
    dialogError.value = '変更できませんでした（どの課題も変わっていません）: ' + err.message;
    console.error('Failed to bulk update issues:', err);
  } finally {
    busy.value = false;
  }
};

// 削除の確認を開く。一緒に消えるサブタスク（選んでいない子課題）を数えるため、全課題を読む
const allTodos = ref([]);
const openDelete = async () => {
  error.value = '';
  try {
    allTodos.value = await todosApi.fetchAll();
    confirmingDelete.value = true;
  } catch (err) {
    error.value = '課題の取得に失敗しました: ' + err.message;
    console.error('Failed to fetch todos for bulk delete:', err);
  }
};
const extraSubtasks = computed(() => {
  const ids = new Set(props.issues.map(issue => issue.ID));
  return allTodos.value.filter(todo => todo.ParentID && ids.has(todo.ParentID) && !ids.has(todo.ID)).length;
});
const deleteMessage = computed(() => {
  const keys = props.issues.slice(0, 5).map(issue => issue.Key).join('、');
  const more = props.issues.length > 5 ? ` ほか ${props.issues.length - 5} 件` : '';
  const subtasks = extraSubtasks.value ? `\nサブタスク ${extraSubtasks.value} 件も一緒に削除されます。` : '';
  return `${keys}${more} を削除します。${subtasks}\nコメント・添付ファイル・作業ログ・変更履歴も削除され、元に戻せません。`;
});

const handleDelete = async () => {
  confirmingDelete.value = false;
  error.value = '';
  try {
    const { count } = await todosApi.bulkDelete(props.issues.map(issue => issue.ID));
    showNotice(`${count} 件の課題を削除しました`);
    emit('clear');
    emit('changed');
  } catch (err) {
    error.value = '削除できませんでした（どの課題も削除されていません）: ' + err.message;
    console.error('Failed to bulk delete issues:', err);
  }
};

onUnmounted(() => {
  clearTimeout(noticeTimer);
});
</script>
