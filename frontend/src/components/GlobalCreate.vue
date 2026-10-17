<template>
  <!-- サイドバーの作成ボタン・ツールバーの「+ 新規課題」から開く課題の作成モーダル。どのページからでも使える -->
  <div
    v-if="open && loaded"
    role="dialog"
    aria-modal="true"
    aria-label="課題を作成"
    class="fixed inset-0 bg-overlay flex items-center justify-center z-50"
    @click.self="close"
  >
    <TaskForm
      :initial-space-id="initialSpaceId"
      :statuses="statuses"
      :epics="epics"
      :spaces="spaces"
      @submit="handleSubmit"
      @close="close"
    />
  </div>

  <!-- 作成した課題のトースト。押すとその課題を開く -->
  <button
    v-if="created"
    type="button"
    class="fixed bottom-8 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 bg-surface text-fg px-5 py-3 rounded-xl shadow-xl text-sm font-semibold border border-line cursor-pointer animate-slideIn"
    @click="openCreated"
  >
    <span class="text-success" aria-hidden="true">✓</span>
    <span><span class="font-mono">{{ created.Key || created.Title }}</span> を作成しました</span>
    <span class="text-accent underline">表示</span>
  </button>

  <div v-if="error" class="fixed bottom-8 right-8 z-50 bg-danger/10 text-danger px-6 py-4 rounded-xl shadow-xl max-w-96 text-sm font-semibold border-l-4 border-danger animate-slideIn">
    {{ error }}
  </div>
</template>

<script setup>
import { ref, computed, watch, onUnmounted } from 'vue';
import TaskForm from './TaskForm.vue';
import { todosApi } from '../api/todos';
import { epicsApi } from '../api/epics';
import { spacesApi } from '../api/spaces';
import { statusesApi } from '../api/statuses';
import { boardPath } from '../utils/spaceRoutes';

// 作成モーダルの開閉（サイドバーの作成ボタン・ツールバーの「+ 新規課題」と共有）
const open = useState('createIssueOpen', () => false);
// 課題を作成するたびに増やす。ボードはこれを見てタスクを取り直す
const issuesVersion = useState('issuesVersion', () => 0);

const route = useRoute();
const spaces = ref([]);
const epics = ref([]);
const statuses = ref([]);
// スペースとエピック・ステータスを読み終えてからフォームを出す（初期スペースを正しく選ぶため）
const loaded = ref(false);
const created = ref(null);
const error = ref('');
let toastTimer = null;

// 今見ているスペース（URL の /spaces/:key/...）を初期値にする。「すべて」や他のページでは未割り当て
const initialSpaceId = computed(() => {
  const key = route.params.key;
  const space = key ? spaces.value.find(s => s.Key === key) : null;
  return space ? space.ID : 0;
});

// 開くたびに最新のスペースとエピックを読む（他のページで作成・変更されていることがある）
watch(open, async (isOpen) => {
  if (!isOpen) {
    loaded.value = false;
    return;
  }
  try {
    error.value = '';
    [spaces.value, epics.value, statuses.value] = await Promise.all([spacesApi.fetchAll(), epicsApi.fetchAll(), statusesApi.fetchAll()]);
    loaded.value = true;
  } catch (err) {
    open.value = false;
    error.value = 'スペースとエピックの取得に失敗しました: ' + err.message;
    console.error('Failed to load create form:', err);
  }
});

const close = () => {
  open.value = false;
};

const handleSubmit = async (formData) => {
  try {
    error.value = '';
    const result = await todosApi.create({
      title: formData.title,
      description: formData.description,
      deadline: formData.deadline,
      // 0 なら bff がスペースの To Do の先頭にする
      statusId: formData.statusId,
      issueType: formData.issueType,
      priority: formData.priority,
      labels: formData.labels,
      storyPoints: formData.storyPoints,
      epicId: formData.epicId,
      spaceId: formData.spaceId
    });
    close();
    issuesVersion.value++;
    showCreated(result.todo);
  } catch (err) {
    error.value = '課題の作成に失敗しました: ' + err.message;
    console.error('Failed to create todo:', err);
  }
};

const showCreated = (todo) => {
  clearTimeout(toastTimer);
  created.value = todo;
  toastTimer = setTimeout(() => {
    created.value = null;
  }, 6000);
};

// 作成した課題を、そのスペースのボードで詳細を開いて表示する
const openCreated = () => {
  const todo = created.value;
  created.value = null;
  clearTimeout(toastTimer);
  const space = spaces.value.find(s => s.ID === todo.SpaceID) || null;
  navigateTo({ path: boardPath(space), query: todo.Key ? { selectedIssue: todo.Key } : {} });
};

onUnmounted(() => {
  clearTimeout(toastTimer);
});
</script>
