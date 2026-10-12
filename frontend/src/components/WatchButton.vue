<template>
  <!-- 課題のウォッチ。Jira と同じく目のアイコンとウォッチャーの数を出し、押すとウォッチの開始・停止を切り替える。
       ホバーでウォッチャーの名前を出す -->
  <button
    type="button"
    :disabled="busy"
    :class="[
      'shrink-0 flex items-center gap-1 px-2 py-1 border-0 rounded-md text-sm font-semibold cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed',
      watching ? 'bg-blue-50 text-blue-700 hover:bg-blue-100' : 'bg-transparent text-gray-500 hover:bg-gray-100'
    ]"
    :title="title"
    :aria-pressed="watching"
    @click="toggle"
  >
    <svg class="w-4 h-4 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
    <span>{{ watching ? 'ウォッチ中' : 'ウォッチ' }}</span>
    <span class="font-normal">{{ watchers.length }}</span>
  </button>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue';
import { watchersApi } from '../api/watchers';
import { useCurrentUser } from '../composables/useCurrentUser';

const props = defineProps({
  task: {
    type: Object,
    required: true
  }
});

const { user: currentUser } = useCurrentUser();
const watchers = ref([]);
const busy = ref(false);

const watching = computed(() => !!currentUser.value && watchers.value.some(w => w.ID === currentUser.value.ID));

const title = computed(() => {
  const names = watchers.value.map(w => w.DisplayName).join('、');
  const action = watching.value ? 'クリックでウォッチをやめる' : 'クリックでウォッチして更新の通知を受け取る';
  return `${action}\nウォッチャー: ${names || 'なし'}`;
});

const fetchWatchers = async () => {
  try {
    watchers.value = await watchersApi.fetchByTodo(props.task.ID);
  } catch (err) {
    console.error('Failed to fetch watchers:', err);
  }
};

const toggle = async () => {
  busy.value = true;
  try {
    watchers.value = watching.value ? await watchersApi.unwatch(props.task.ID) : await watchersApi.watch(props.task.ID);
  } catch (err) {
    console.error('Failed to toggle watch:', err);
  } finally {
    busy.value = false;
  }
};

// 担当者の変更やコメントでウォッチャーが増えるので、課題が変わるたびに取り直す（セッションの Cookie を送れるブラウザ側でだけ取る）
onMounted(fetchWatchers);
watch(() => [props.task.ID, props.task.UpdatedAt, props.task.CommentCount], fetchWatchers);
</script>
