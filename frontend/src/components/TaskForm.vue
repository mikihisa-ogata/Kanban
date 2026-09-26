<template>
  <div class="bg-white rounded-2xl px-8 pt-10 pb-6 shadow-2xl mx-4 max-w-2xl w-full max-h-[90vh] overflow-y-auto relative">
    <!-- 閉じるボタン -->
    <button 
      @click="$emit('close')"
      class="absolute top-4 right-4 bg-transparent border-0 text-gray-400 text-2xl cursor-pointer p-0 w-8 h-8 flex items-center justify-center rounded hover:bg-gray-100 hover:text-gray-600 transition-all duration-200"
      title="閉じる"
    >
      ×
    </button>

    <form @submit.prevent="handleSubmit">
      <div class="mb-4">
        <label for="title" class="block mb-2 text-sm font-semibold text-gray-700">
          タイトル <span class="text-red-500">*</span>
        </label>
        <input
          id="title"
          v-model="formData.title"
          type="text"
          placeholder="タスクのタイトルを入力"
          class="w-full px-3 py-2.5 border-2 border-gray-300 rounded-lg text-sm transition-all duration-200 text-gray-900 bg-white focus:outline-none focus:border-blue-500 focus:shadow-sm focus:shadow-blue-100"
          required
        />
      </div>
      
      <div>
        <label for="description" class="block mb-2 text-sm font-semibold text-gray-700">
          説明
        </label>
        <textarea
          id="description"
          v-model="formData.description"
          rows="3"
          placeholder="タスクの詳細を入力（任意）"
          class="block w-full px-3 py-2.5 border-2 border-gray-300 rounded-lg text-sm transition-all duration-200 text-gray-900 bg-white resize-y focus:outline-none focus:border-blue-500 focus:shadow-sm focus:shadow-blue-100"
        ></textarea>
      </div>

      <!-- 短い値の項目は、ラベルと値を横に並べて内容の幅に合わせる -->
      <div class="flex flex-col gap-3 mt-4">
        <div class="flex items-center gap-4">
          <label for="deadline" class="w-24 shrink-0 text-sm font-semibold text-gray-700">
            期限
          </label>
          <input
            id="deadline"
            v-model="formData.deadline"
            type="date"
            class="px-3 py-2 border-2 border-gray-300 rounded-lg text-sm transition-all duration-200 text-gray-900 bg-white hover:border-gray-400 focus:outline-none focus:border-blue-500 focus:shadow-sm focus:shadow-blue-100"
          />
        </div>

        <div class="flex items-center gap-4">
          <label for="status" class="w-24 shrink-0 text-sm font-semibold text-gray-700">
            ステータス
          </label>
          <OptionSelect
            id="status"
            v-model="formData.status"
            :options="statusOptions"
          />
        </div>

        <div v-if="spaces.length" class="flex items-center gap-4">
          <label for="space" class="w-24 shrink-0 text-sm font-semibold text-gray-700">
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
          <label for="epic" class="w-24 shrink-0 text-sm font-semibold text-gray-700">
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

      <div class="flex justify-end mt-6">
        <button 
          type="submit" 
          :disabled="isSubmitting"
          class="py-2.5 px-8 bg-gradient-to-r from-blue-500 to-blue-600 text-white border-0 rounded-lg text-base font-semibold cursor-pointer transition-all duration-200 hover:enabled:-translate-y-0.5 hover:enabled:shadow-lg hover:enabled:shadow-blue-400 active:enabled:translate-y-0 active:enabled:shadow-md disabled:opacity-60 disabled:cursor-not-allowed"
        >
          <template v-if="task">{{ isSubmitting ? '保存中...' : '保存' }}</template>
          <template v-else>{{ isSubmitting ? '作成中...' : 'タスクを作成' }}</template>
        </button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { ref, defineEmits, defineProps, watch, computed } from 'vue';
import OptionSelect from './OptionSelect.vue';

const props = defineProps({
  initialStatus: {
    type: String,
    default: 'Open'
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
  },
  // 指定した場合は、そのタスクの詳細を表示・編集する
  task: {
    type: Object,
    default: null
  }
});

const emit = defineEmits(['submit', 'close']);

const formData = ref(props.task
  ? {
      title: props.task.Title,
      description: props.task.Description,
      deadline: props.task.Deadline,
      status: props.task.Status,
      epicId: props.task.EpicID,
      spaceId: props.task.SpaceID
    }
  : {
      title: '',
      description: '',
      deadline: '',
      status: props.initialStatus,
      epicId: props.initialEpicId,
      spaceId: props.initialSpaceId
    });

// 選択中のスペースに属するエピックだけを選べる
const spaceEpics = computed(() => props.epics.filter(epic => epic.SpaceID === formData.value.spaceId));

const statusOptions = [
  { value: 'Open', label: 'オープン' },
  { value: 'InProgress', label: '進行中' },
  { value: 'Waiting', label: '待ち状況' },
  { value: 'Closed', label: 'クローズ' }
];

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

// initialStatusが変更された時に formData.status を更新
watch(() => props.initialStatus, (newStatus) => {
  formData.value.status = newStatus;
});

const isSubmitting = ref(false);

const handleSubmit = async () => {
  if (isSubmitting.value) return;
  
  isSubmitting.value = true;
  try {
    await emit('submit', { ...formData.value });
    if (props.task) return;
    // フォームをリセット
    formData.value = {
      title: '',
      description: '',
      deadline: '',
      status: props.initialStatus,
      epicId: props.initialEpicId,
      spaceId: props.initialSpaceId
    };
  } finally {
    isSubmitting.value = false;
  }
};
</script>
