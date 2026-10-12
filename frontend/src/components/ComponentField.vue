<template>
  <!-- 課題のコンポーネント（Jira の Components）。クリックで、課題のスペースのコンポーネントから複数選ぶ -->
  <div>
    <div v-if="editing">
      <fieldset :id="id" class="m-0 p-0 border-0 flex flex-col gap-1">
        <legend class="sr-only">コンポーネント</legend>
        <label
          v-for="component in choices"
          :key="component.ID"
          class="flex items-center gap-2 text-sm text-gray-800 cursor-pointer"
        >
          <input v-model="draft" type="checkbox" :value="component.ID" class="cursor-pointer" />
          <span class="truncate">{{ component.Name }}</span>
          <span v-if="component.DefaultAssignee" class="shrink-0 text-xs text-gray-500" :title="`既定の担当者: ${component.DefaultAssignee.DisplayName}`">
            {{ component.DefaultAssignee.DisplayName }}
          </span>
        </label>
        <p v-if="!choices.length" class="m-0 text-sm text-gray-500">このスペースにはコンポーネントがありません（スペースの設定で作れます）。</p>
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
    <!-- スペース未割り当ての課題にはコンポーネントがないので、編集できない -->
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
        v-for="component in assigned"
        :key="component.ID"
        class="max-w-full truncate px-1.5 py-0.5 rounded bg-gray-100 border border-gray-300 text-xs text-gray-800"
      >
        {{ component.Name }}
      </span>
      <span v-if="!assigned.length" class="text-sm text-gray-400">なし</span>
    </button>
    <p v-if="error" class="m-0 mt-1 text-xs text-red-600">{{ error }}</p>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { componentsApi } from '../api/components';

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

// changed: コンポーネントを変えた（担当者が入ることもあるので、呼び出し側で課題を取り直す）
const emit = defineEmits(['changed']);

const components = ref([]);
const editing = ref(false);
const draft = ref([]);
const busy = ref(false);
const error = ref('');

const componentIds = computed(() => props.task.ComponentIDs || []);
const assigned = computed(() => components.value.filter(c => componentIds.value.includes(c.ID)));
const choices = computed(() => components.value.filter(c => c.SpaceID === props.task.SpaceID));

const fetchComponents = async () => {
  try {
    components.value = await componentsApi.fetchAll();
  } catch (err) {
    error.value = 'コンポーネントの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch components:', err);
  }
};

const startEdit = async () => {
  error.value = '';
  // 設定で作ったばかりのコンポーネントも選べるよう、開くたびに取り直す
  await fetchComponents();
  draft.value = [...componentIds.value];
  editing.value = true;
};

const save = async () => {
  busy.value = true;
  try {
    error.value = '';
    await componentsApi.setTodoComponents(props.task.ID, draft.value);
    editing.value = false;
    emit('changed');
  } catch (err) {
    error.value = 'コンポーネントの変更に失敗しました: ' + err.message;
    console.error('Failed to set components:', err);
  } finally {
    busy.value = false;
  }
};

onMounted(fetchComponents);
</script>
