<template>
  <!-- 選択中の値を枠付きのチップで表示し、選択肢から選んだ値だとわかるようにする -->
  <div class="relative inline-flex items-center min-w-0 max-w-full min-h-10 pl-1.5 pr-9 py-1 border-2 border-gray-300 rounded-lg bg-white cursor-pointer transition-all duration-200 hover:border-gray-400 focus-within:border-blue-500 focus-within:shadow-sm focus-within:shadow-blue-100">
    <span
      v-if="selectedLabel"
      :class="[
        'block truncate px-2.5 py-0.5 rounded-md border text-sm font-medium',
        isEmpty ? 'border-gray-300 bg-gray-50 text-gray-500' : 'border-blue-300 bg-blue-50 text-blue-700'
      ]"
    >
      {{ selectedLabel }}
    </span>
    <svg class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
      <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
    </svg>
    <!-- 見た目はチップで出し、操作はブラウザ標準のセレクトに任せる -->
    <select
      :id="id"
      v-model="value"
      class="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
    >
      <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
    </select>
  </div>
</template>

<script setup>
import { computed, defineProps, defineEmits } from 'vue';

const props = defineProps({
  id: {
    type: String,
    required: true
  },
  modelValue: {
    type: [String, Number],
    required: true
  },
  // { value, label } の配列
  options: {
    type: Array,
    required: true
  },
  // 「なし」のように、何も選んでいないことを表す値
  emptyValue: {
    type: [String, Number],
    default: null
  }
});

const emit = defineEmits(['update:modelValue']);

const value = computed({
  get: () => props.modelValue,
  set: (newValue) => emit('update:modelValue', newValue)
});

const selectedLabel = computed(() => props.options.find(option => option.value === props.modelValue)?.label ?? '');

const isEmpty = computed(() => props.modelValue === props.emptyValue);
</script>
