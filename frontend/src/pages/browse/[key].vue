<template>
  <!-- 課題の全画面ページ（Jira の /browse/KAN-12）。左ナビは課題のスペースのものを出す -->
  <SpaceShell :spaces="spaces" :current-space="space" @changed="handleSpacesChanged">
    <div class="px-8 py-8">
      <p v-if="loading" class="m-0 text-white">読み込み中...</p>

      <div v-else-if="!issue" class="max-w-3xl bg-white rounded-2xl px-8 py-6 shadow-xl text-gray-700">
        <template v-if="loadError">{{ loadError }}</template>
        <template v-else>
          <p class="m-0 text-lg font-bold text-gray-900">課題が見つかりません</p>
          <p class="mt-2 mb-0 text-sm">
            「{{ issueKey }}」の課題は存在しないか、削除されています。
            <NuxtLink to="/board" class="ml-2 text-blue-600 font-semibold">すべてのスペースのボードを開く</NuxtLink>
          </p>
        </template>
      </div>

      <div v-else class="max-w-6xl bg-white rounded-2xl px-8 py-6 shadow-xl">
        <IssueDetail
          :task="issue"
          :epics="epics"
          :spaces="spaces"
          :subtasks="subtasks"
          :parent-task="parentTask"
          :all-todos="todos"
          @update="handleUpdate"
          @subtasks-changed="fetchTodos"
          @links-changed="handleLinksChanged"
          @open="navigateTo(issuePath($event))"
        />
      </div>
    </div>

    <div v-if="error" class="fixed bottom-8 right-8 bg-red-100 text-red-800 px-6 py-4 rounded-xl shadow-xl max-w-96 text-sm font-semibold border-l-4 border-red-500 animate-slideIn">
      {{ error }}
    </div>
  </SpaceShell>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue';
import SpaceShell from '~/components/SpaceShell.vue';
import IssueDetail from '~/components/IssueDetail.vue';
import { todosApi } from '~/api/todos';
import { epicsApi } from '~/api/epics';
import { spacesApi } from '~/api/spaces';
import { issuePath } from '~/utils/spaceRoutes';

const route = useRoute();
// URL の課題キー（例: /browse/KAN-12）
const issueKey = computed(() => String(route.params.key));

const issue = ref(null);
const spaces = ref([]);
const epics = ref([]);
const loading = ref(true);
// 読み込みに失敗したときのメッセージ（見つからない場合とは分ける）
const loadError = ref('');
// 編集の保存に失敗したときのメッセージ
const error = ref('');

// 子課題・親課題とリンクする課題の候補を出すための全タスク
const todos = ref([]);
const subtasks = computed(() => (issue.value ? todos.value.filter(todo => todo.ParentID === issue.value.ID) : []));
const parentTask = computed(() =>
  issue.value && issue.value.ParentID ? todos.value.find(todo => todo.ID === issue.value.ParentID) || null : null
);

const space = computed(() => (issue.value ? spaces.value.find(s => s.ID === issue.value.SpaceID) || null : null));

useHead(() => ({
  title: issue.value ? `[${issue.value.Key}] ${issue.value.Title} - Kanban` : 'Kanban'
}));

// URL を課題の今のキーに合わせる（小文字で開いたときや、スペースを変えてキーが変わったとき）
const replaceUrlWithKey = async (key) => {
  if (key && key !== issueKey.value) {
    await navigateTo(issuePath(key), { replace: true });
  }
};

const fetchIssue = async () => {
  try {
    loadError.value = '';
    issue.value = await todosApi.fetchByKey(issueKey.value);
    await replaceUrlWithKey(issue.value?.Key);
  } catch (err) {
    issue.value = null;
    loadError.value = '課題の取得に失敗しました: ' + err.message;
    console.error('Failed to fetch issue:', err);
  }
};

const fetchTodos = async () => {
  try {
    todos.value = await todosApi.fetchAll();
  } catch (err) {
    error.value = '子課題の取得に失敗しました: ' + err.message;
    console.error('Failed to fetch todos:', err);
  }
};

const fetchSpacesAndEpics = async () => {
  try {
    [spaces.value, epics.value] = await Promise.all([spacesApi.fetchAll(), epicsApi.fetchAll()]);
  } catch (err) {
    error.value = 'スペースとエピックの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch spaces and epics:', err);
  }
};

// インライン編集した内容を保存する。スペースを変えて課題キーが変わったら URL も変える
const handleUpdate = async (payload) => {
  const id = issue.value.ID;
  try {
    error.value = '';
    const result = await todosApi.update(id, payload);
    issue.value = result.todo;
    await replaceUrlWithKey(result.todo.Key);
    // スペースを変えるとサブタスクの課題キーも変わるので取り直す
    await fetchTodos();
  } catch (err) {
    error.value = '課題の更新に失敗しました: ' + err.message;
    console.error('Failed to update issue:', err);
    await fetchIssue();
  }
};

// リンクを追加・削除したら、この課題のリンクと候補の一覧を取り直す
const handleLinksChanged = async () => {
  await Promise.all([fetchIssue(), fetchTodos()]);
};

// サイドバーでスペースを作成・削除したら取り直す。
// 課題のスペースが削除されると課題キーが変わるので、ID で探し直す
const handleSpacesChanged = async () => {
  await fetchSpacesAndEpics();
  if (!issue.value) return;
  try {
    const id = issue.value.ID;
    todos.value = await todosApi.fetchAll();
    const todo = todos.value.find(t => t.ID === id) || null;
    issue.value = todo;
    await replaceUrlWithKey(todo?.Key);
  } catch (err) {
    error.value = '課題の取得に失敗しました: ' + err.message;
    console.error('Failed to refetch issue:', err);
  }
};

// 別の課題のページへ移ったら読み直す（キーを付け替えただけなら読み直さない）
watch(issueKey, async (key) => {
  if (issue.value && issue.value.Key === key) return;
  loading.value = true;
  await Promise.all([fetchIssue(), fetchTodos()]);
  loading.value = false;
});

onMounted(async () => {
  await Promise.all([fetchIssue(), fetchSpacesAndEpics(), fetchTodos()]);
  loading.value = false;
});
</script>
