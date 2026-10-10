<template>
  <!-- Markdown の入力欄。「編集」と「プレビュー」を切り替えられる -->
  <div>
    <div class="flex gap-1 mb-1.5" role="tablist">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        type="button"
        role="tab"
        :aria-selected="mode === tab.value"
        :class="[
          'px-3 py-1 border-0 rounded-md text-xs font-semibold cursor-pointer',
          mode === tab.value ? 'bg-gray-200 text-gray-900' : 'bg-transparent text-gray-500 hover:bg-gray-100'
        ]"
        @click="switchMode(tab.value)"
      >
        {{ tab.label }}
      </button>
      <span class="ml-auto self-center text-xs text-gray-400">Markdown が使えます</span>
    </div>
    <textarea
      v-show="mode === 'edit'"
      :id="id"
      ref="textareaRef"
      :value="modelValue"
      :rows="rows"
      :placeholder="placeholder"
      :aria-label="ariaLabel"
      class="block w-full px-3 py-2.5 border-2 border-gray-300 rounded-lg text-sm text-gray-900 bg-white resize-y transition-all duration-200 focus:outline-none focus:border-blue-500 focus:shadow-sm focus:shadow-blue-100"
      @input="$emit('update:modelValue', $event.target.value)"
      @keydown="$emit('keydown', $event)"
    ></textarea>
    <div
      v-if="mode === 'preview'"
      class="min-h-24 px-3 py-2.5 border-2 border-dashed border-gray-300 rounded-lg bg-white"
    >
      <MarkdownView v-if="modelValue.trim()" :source="modelValue" />
      <p v-else class="m-0 text-sm text-gray-400">プレビューする内容がありません</p>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick } from 'vue';
import MarkdownView from './MarkdownView.vue';

defineProps({
  id: {
    type: String,
    default: undefined
  },
  modelValue: {
    type: String,
    required: true
  },
  rows: {
    type: Number,
    default: 6
  },
  placeholder: {
    type: String,
    default: ''
  },
  ariaLabel: {
    type: String,
    default: undefined
  }
});

// keydown: テキストエリアのキー操作（保存のショートカットなどを呼び出し側で扱う）
defineEmits(['update:modelValue', 'keydown']);

const tabs = [
  { value: 'edit', label: '編集' },
  { value: 'preview', label: 'プレビュー' }
];

const mode = ref('edit');
const textareaRef = ref(null);

const switchMode = async (value) => {
  mode.value = value;
  if (value === 'edit') {
    await nextTick();
    textareaRef.value?.focus();
  }
};

// 呼び出し側から入力欄にフォーカスを移す
const focus = () => {
  textareaRef.value?.focus();
};

defineExpose({ focus });
</script>
