<template>
  <!-- ボードのフィルターの複数選択（ラベル・課題タイプ・エピック）。選んでいる数をボタンに出す -->
  <div ref="root" class="relative">
    <button
      type="button"
      :class="[
        'flex items-center gap-1 px-3 py-1.5 border-0 rounded-md text-sm font-semibold cursor-pointer',
        modelValue.length ? 'bg-accent/10 text-accent' : 'bg-surface text-fg hover:bg-raised'
      ]"
      aria-haspopup="listbox"
      :aria-expanded="open"
      @click="open = !open"
    >
      {{ label }}
      <span v-if="modelValue.length" class="px-1.5 rounded-full bg-accent text-on-accent text-xs leading-5">{{ modelValue.length }}</span>
      <svg class="w-4 h-4 block" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
        <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
      </svg>
    </button>
    <div
      v-if="open"
      class="absolute left-0 top-full mt-1 w-60 max-h-72 overflow-y-auto p-1 bg-surface border border-line rounded-lg shadow-lg z-30"
      role="listbox"
      :aria-label="label"
      aria-multiselectable="true"
    >
      <label
        v-for="option in options"
        :key="option.value"
        class="flex items-center gap-2 px-2 py-1.5 rounded text-sm text-fg cursor-pointer hover:bg-raised"
      >
        <input
          type="checkbox"
          class="cursor-pointer"
          :checked="modelValue.includes(option.value)"
          @change="toggle(option.value)"
        />
        <span class="min-w-0 truncate">{{ option.label }}</span>
      </label>
      <p v-if="!options.length" class="m-0 px-2 py-1.5 text-sm text-muted">{{ emptyText }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue';

const props = defineProps({
  label: {
    type: String,
    required: true
  },
  // { value, label } の配列
  options: {
    type: Array,
    required: true
  },
  // 選んでいる値の配列
  modelValue: {
    type: Array,
    required: true
  },
  emptyText: {
    type: String,
    default: '選べるものがありません'
  }
});

const emit = defineEmits(['update:modelValue']);

const root = ref(null);
const open = ref(false);

const toggle = (value) => {
  emit('update:modelValue', props.modelValue.includes(value)
    ? props.modelValue.filter(v => v !== value)
    : [...props.modelValue, value]);
};

// メニューの外を押すか Esc で閉じる
const closeOnOutsideClick = (event) => {
  if (open.value && root.value && !root.value.contains(event.target)) {
    open.value = false;
  }
};
const closeOnEscape = (event) => {
  if (event.key === 'Escape') {
    open.value = false;
  }
};
onMounted(() => {
  document.addEventListener('click', closeOnOutsideClick);
  document.addEventListener('keydown', closeOnEscape);
});
onBeforeUnmount(() => {
  document.removeEventListener('click', closeOnOutsideClick);
  document.removeEventListener('keydown', closeOnEscape);
});
</script>
