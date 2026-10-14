<template>
  <div class="bg-surface rounded-2xl px-8 pt-10 pb-6 shadow-2xl mx-4 max-w-2xl w-full max-h-[90vh] overflow-y-auto relative">
    <!-- 閉じるボタン -->
    <button 
      @click="$emit('close')"
      class="absolute top-4 right-4 bg-transparent border-0 text-muted text-2xl cursor-pointer p-0 w-8 h-8 flex items-center justify-center rounded hover:bg-raised hover:text-muted transition-all duration-200"
      title="閉じる"
    >
      ×
    </button>

    <form @submit.prevent="handleSubmit">
      <div class="mb-4">
        <label for="title" class="block mb-2 text-sm font-semibold text-fg">
          タイトル <span class="text-danger">*</span>
        </label>
        <input
          id="title"
          v-model="formData.title"
          type="text"
          placeholder="タスクのタイトルを入力"
          class="w-full px-3 py-2.5 border-2 border-line rounded-lg text-sm transition-all duration-200 text-fg bg-surface focus:outline-none focus:border-accent focus:shadow-sm focus:shadow-accent/20"
          required
        />
      </div>
      
      <div>
        <label for="description" class="block mb-2 text-sm font-semibold text-fg">
          説明
        </label>
        <MarkdownEditor
          id="description"
          v-model="formData.description"
          :rows="4"
          placeholder="タスクの詳細を入力（任意。Markdown が使えます）"
        />
      </div>

      <!-- 短い値の項目は、ラベルと値を横に並べて内容の幅に合わせる -->
      <div class="flex flex-col gap-3 mt-4">
        <div class="flex items-center gap-4">
          <label for="issueType" class="w-24 shrink-0 text-sm font-semibold text-fg">
            課題タイプ
          </label>
          <div class="flex items-center gap-2 min-w-0">
            <IssueTypeIcon :type="formData.issueType" />
            <OptionSelect
              id="issueType"
              v-model="formData.issueType"
              :options="ISSUE_TYPE_OPTIONS"
            />
          </div>
        </div>

        <div class="flex items-center gap-4">
          <label for="priority" class="w-24 shrink-0 text-sm font-semibold text-fg">
            優先度
          </label>
          <div class="flex items-center gap-2 min-w-0">
            <PriorityIcon :priority="formData.priority" />
            <OptionSelect
              id="priority"
              v-model="formData.priority"
              :options="PRIORITY_OPTIONS"
            />
          </div>
        </div>

        <div class="flex items-center gap-4">
          <label for="deadline" class="w-24 shrink-0 text-sm font-semibold text-fg">
            期限
          </label>
          <input
            id="deadline"
            v-model="formData.deadline"
            type="date"
            class="px-3 py-2 border-2 border-line rounded-lg text-sm transition-all duration-200 text-fg bg-surface hover:border-line focus:outline-none focus:border-accent focus:shadow-sm focus:shadow-accent/20"
          />
        </div>

        <div class="flex items-center gap-4">
          <label for="status" class="w-24 shrink-0 text-sm font-semibold text-fg">
            ステータス
          </label>
          <OptionSelect
            id="status"
            v-model="formData.statusId"
            :options="statusOptions"
          />
        </div>

        <div class="flex items-center gap-4">
          <label for="storyPoints" class="w-24 shrink-0 text-sm font-semibold text-fg">
            ストーリーポイント
          </label>
          <div class="min-w-0">
            <input
              id="storyPoints"
              v-model="formData.storyPoints"
              type="text"
              inputmode="decimal"
              placeholder="未設定"
              class="w-28 px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface hover:border-line focus:outline-none focus:border-accent"
            />
            <p v-if="storyPointsError" class="mt-1 mb-0 text-xs text-danger">{{ storyPointsError }}</p>
          </div>
        </div>

        <div class="flex items-start gap-4">
          <label for="labels" class="w-24 shrink-0 pt-2.5 text-sm font-semibold text-fg">
            ラベル
          </label>
          <LabelInput id="labels" v-model="formData.labels" />
        </div>

        <div v-if="spaces.length" class="flex items-center gap-4">
          <label for="space" class="w-24 shrink-0 text-sm font-semibold text-fg">
            スペース
          </label>
          <OptionSelect
            id="space"
            v-model="formData.spaceId"
            :options="spaceOptions"
            :empty-value="0"
          />
        </div>

        <div class="flex items-center gap-4">
          <label for="epic" class="w-24 shrink-0 text-sm font-semibold text-fg">
            エピック
          </label>
          <OptionSelect
            id="epic"
            v-model="formData.epicId"
            :options="epicOptions"
            :empty-value="0"
          />
        </div>
      </div>

      <div class="flex items-center justify-end gap-4 mt-6">
        <button 
          type="submit" 
          :disabled="isSubmitting"
          class="py-2.5 px-8 bg-accent text-on-accent border-0 rounded-lg hover:enabled:bg-accent-hover text-base font-semibold cursor-pointer transition-all duration-200 hover:enabled:-translate-y-0.5 hover:enabled:shadow-lg hover:enabled:shadow-accent/40 active:enabled:translate-y-0 active:enabled:shadow-md disabled:opacity-60 disabled:cursor-not-allowed"
        >
          {{ isSubmitting ? '作成中...' : 'タスクを作成' }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { ref, defineEmits, defineProps, watch, computed } from 'vue';
import OptionSelect from './OptionSelect.vue';
import IssueTypeIcon from './IssueTypeIcon.vue';
import PriorityIcon from './PriorityIcon.vue';
import LabelInput from './LabelInput.vue';
import MarkdownEditor from './MarkdownEditor.vue';
import { parseStoryPoints } from '../utils/storyPoints';
import { ISSUE_TYPE_OPTIONS, DEFAULT_ISSUE_TYPE } from '../utils/issueTypes';
import { PRIORITY_OPTIONS, DEFAULT_PRIORITY } from '../utils/priorities';
import { statusesOfSpace, firstStatusInCategory, statusOptionsOf } from '../utils/statuses';

const props = defineProps({
  // すべてのワークフローのステータス（選んだスペースのステータスを選択肢にする）
  statuses: {
    type: Array,
    default: () => []
  },
  // 初期のステータス。0 なら initialCategory の先頭のステータス
  initialStatusId: {
    type: Number,
    default: 0
  },
  initialCategory: {
    type: String,
    default: 'ToDo'
  },
  initialEpicId: {
    type: Number,
    default: 0
  },
  initialSpaceId: {
    type: Number,
    default: 0
  },
  epics: {
    type: Array,
    default: () => []
  },
  spaces: {
    type: Array,
    default: () => []
  }
});

const emit = defineEmits(['submit', 'close']);

const formData = ref({
  title: '',
  description: '',
  deadline: '',
  // 0 は未選択（bff がスペースの To Do の先頭にする）。下の watch で選んだスペースのステータスにする
  statusId: props.initialStatusId,
  issueType: DEFAULT_ISSUE_TYPE,
  priority: DEFAULT_PRIORITY,
  labels: [],
  storyPoints: '',
  epicId: props.initialEpicId,
  spaceId: props.initialSpaceId
});

// 選択中のスペースに属するエピックだけを選べる
const spaceEpics = computed(() => props.epics.filter(epic => epic.SpaceID === formData.value.spaceId));

// 選択中のスペースのワークフローのステータスを選べる
const spaceStatuses = computed(() => statusesOfSpace(props.statuses, formData.value.spaceId));
const statusOptions = computed(() => statusOptionsOf(spaceStatuses.value));

// 選んだステータスが選択中のスペースにない（スペースを変えた・初期値がカテゴリだけ）ときは、
// 同じカテゴリの先頭のステータスにする
const selectedCategory = ref(props.initialCategory);
watch([spaceStatuses, () => formData.value.statusId], () => {
  const current = spaceStatuses.value.find(status => status.ID === formData.value.statusId);
  if (current) {
    selectedCategory.value = current.Category;
    return;
  }
  formData.value.statusId = firstStatusInCategory(spaceStatuses.value, selectedCategory.value)?.ID ?? 0;
}, { immediate: true });

const spaceOptions = computed(() => [
  { value: 0, label: 'なし' },
  ...props.spaces.map(space => ({ value: space.ID, label: space.Title }))
]);

const epicOptions = computed(() => [
  { value: 0, label: 'なし' },
  ...spaceEpics.value.map(epic => ({ value: epic.ID, label: epic.Title }))
]);

// スペースを変えたら、別のスペースのエピックは外す
watch(() => formData.value.spaceId, () => {
  if (!spaceEpics.value.some(epic => epic.ID === formData.value.epicId)) {
    formData.value.epicId = 0;
  }
});

// 開いた列が変わったら、初期のステータスを選び直す
watch(() => [props.initialStatusId, props.initialCategory], ([statusId, category]) => {
  selectedCategory.value = category;
  formData.value.statusId = statusId;
});

const isSubmitting = ref(false);

// ストーリーポイントの入力エラー（空は未設定なのでエラーにしない）
const storyPointsError = computed(() => parseStoryPoints(formData.value.storyPoints).error);

const handleSubmit = async () => {
  if (isSubmitting.value || storyPointsError.value) return;
  
  isSubmitting.value = true;
  try {
    await emit('submit', { ...formData.value, storyPoints: parseStoryPoints(formData.value.storyPoints).value });
    // フォームをリセット
    selectedCategory.value = props.initialCategory;
    formData.value = {
      title: '',
      description: '',
      deadline: '',
      statusId: props.initialStatusId,
      issueType: DEFAULT_ISSUE_TYPE,
      priority: DEFAULT_PRIORITY,
      labels: [],
      storyPoints: '',
      epicId: props.initialEpicId,
      spaceId: props.initialSpaceId
    };
  } finally {
    isSubmitting.value = false;
  }
};
</script>
