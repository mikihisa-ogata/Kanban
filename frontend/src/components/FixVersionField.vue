<template>
  <!-- 課題の修正バージョン（Jira の Fix versions）。クリックで、課題のスペースのバージョンから複数選ぶ -->
  <div>
    <div v-if="editing">
      <fieldset :id="id" class="m-0 p-0 border-0 flex flex-col gap-1">
        <legend class="sr-only">修正バージョン</legend>
        <label
          v-for="version in choices"
          :key="version.ID"
          class="flex items-center gap-2 text-sm text-gray-800 cursor-pointer"
        >
          <input v-model="draft" type="checkbox" :value="version.ID" class="cursor-pointer" />
          <span class="truncate">{{ version.Name }}</span>
          <span :class="['shrink-0 px-1.5 rounded text-xs', VERSION_STATUS_BADGE_CLASS[version.Status]]">{{ VERSION_STATUS_LABEL[version.Status] }}</span>
        </label>
        <p v-if="!choices.length" class="m-0 text-sm text-gray-500">このスペースにはバージョンがありません。</p>
      </fieldset>
      <div class="flex gap-2 mt-2">
        <button
          type="button"
          class="px-3 py-1 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700 disabled:opacity-50"
          :disabled="busy"
          @click="save"
        >
          保存
        </button>
        <button
          type="button"
          class="px-3 py-1 bg-transparent text-gray-700 border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-gray-100"
          @click="editing = false"
        >
          キャンセル
        </button>
      </div>
    </div>
    <!-- スペース未割り当ての課題にはバージョンがないので、編集できない -->
    <span v-else-if="!task.SpaceID" :id="id" class="text-sm text-gray-400" title="スペースに入れると付けられます">なし</span>
    <button
      v-else
      :id="id"
      type="button"
      class="w-full -mx-2 px-2 py-1.5 flex flex-wrap gap-1 bg-transparent border-0 rounded-md text-left cursor-pointer hover:bg-gray-100"
      title="クリックして編集"
      @click="startEdit"
    >
      <span
        v-for="version in assigned"
        :key="version.ID"
        class="max-w-full truncate px-1.5 py-0.5 rounded bg-gray-100 border border-gray-300 text-xs text-gray-800"
      >
        {{ version.Name }}
      </span>
      <span v-if="!assigned.length" class="text-sm text-gray-400">なし</span>
    </button>
    <p v-if="error" class="m-0 mt-1 text-xs text-red-600">{{ error }}</p>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { versionsApi } from '../api/versions';
import { VERSION_ARCHIVED, VERSION_STATUS_LABEL, VERSION_STATUS_BADGE_CLASS } from '../utils/versions';

const props = defineProps({
  id: {
    type: String,
    required: true
  },
  task: {
    type: Object,
    required: true
  }
});

// changed: 修正バージョンを変えた（呼び出し側で課題を取り直す）
const emit = defineEmits(['changed']);

const versions = ref([]);
const editing = ref(false);
const draft = ref([]);
const busy = ref(false);
const error = ref('');

const fixVersionIds = computed(() => props.task.FixVersionIDs || []);
const assigned = computed(() => versions.value.filter(v => fixVersionIds.value.includes(v.ID)));
// 選べるのは課題のスペースのバージョン。アーカイブ済みは付いているものだけ（bff と同じく新しくは付けられない）
const choices = computed(() =>
  versions.value.filter(v =>
    v.SpaceID === props.task.SpaceID && (v.Status !== VERSION_ARCHIVED || fixVersionIds.value.includes(v.ID))
  )
);

const fetchVersions = async () => {
  try {
    versions.value = await versionsApi.fetchAll();
  } catch (err) {
    error.value = 'バージョンの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch versions:', err);
  }
};

const startEdit = async () => {
  error.value = '';
  // 別の画面で作ったバージョンも選べるよう、開くたびに取り直す
  await fetchVersions();
  draft.value = [...fixVersionIds.value];
  editing.value = true;
};

const save = async () => {
  busy.value = true;
  try {
    error.value = '';
    await versionsApi.setTodoVersions(props.task.ID, draft.value);
    editing.value = false;
    emit('changed');
  } catch (err) {
    error.value = '修正バージョンの変更に失敗しました: ' + err.message;
    console.error('Failed to set fix versions:', err);
  } finally {
    busy.value = false;
  }
};

onMounted(fetchVersions);
</script>
