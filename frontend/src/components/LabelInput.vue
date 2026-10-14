<template>
  <!-- ラベルの入力欄。付けたラベルをチップで出し、入力中は既存のラベルを補完候補に出す。新しいラベルはその場で作れる -->
  <div class="relative min-w-0 w-full">
    <div
      class="flex flex-wrap items-center gap-1.5 min-h-10 px-2 py-1.5 border-2 border-line rounded-lg bg-surface cursor-text transition-all duration-200 hover:border-line focus-within:border-accent focus-within:shadow-sm focus-within:shadow-accent/20"
      @click="inputRef?.focus()"
    >
      <span
        v-for="label in modelValue"
        :key="label"
        class="inline-flex items-center gap-1 max-w-full pl-2 pr-1 py-0.5 rounded bg-raised border border-line text-sm text-fg"
      >
        <span class="truncate">{{ label }}</span>
        <button
          type="button"
          class="bg-transparent border-0 p-0 w-4 h-4 flex items-center justify-center rounded text-muted cursor-pointer hover:bg-line hover:text-fg leading-none"
          :title="`${label} を外す`"
          @click.stop="remove(label)"
        >
          ×
        </button>
      </span>
      <input
        :id="id"
        ref="inputRef"
        v-model="text"
        type="text"
        autocomplete="off"
        role="combobox"
        :aria-expanded="showSuggestions"
        :placeholder="modelValue.length ? '' : 'ラベルを入力'"
        class="flex-1 min-w-24 border-0 p-0.5 text-sm text-fg bg-transparent focus:outline-none"
        @keydown="handleKeydown"
        @focus="focused = true"
        @blur="handleBlur"
      />
    </div>

    <ul
      v-if="showSuggestions"
      class="absolute z-10 left-0 right-0 mt-1 m-0 p-1 list-none bg-surface border border-line rounded-lg shadow-lg max-h-56 overflow-y-auto"
      role="listbox"
    >
      <li
        v-for="(option, index) in options"
        :key="option.value + (option.isNew ? ':new' : '')"
        role="option"
        :aria-selected="index === activeIndex"
        :class="[
          'px-3 py-1.5 rounded text-sm cursor-pointer truncate',
          index === activeIndex ? 'bg-accent/10 text-accent' : 'text-fg hover:bg-raised'
        ]"
        @mousedown.prevent="add(option.value)"
        @mouseenter="activeIndex = index"
      >
        <template v-if="option.isNew">「{{ option.value }}」を新しいラベルとして作成</template>
        <template v-else>{{ option.value }}</template>
      </li>
    </ul>
    <p v-if="error" class="mt-1 mb-0 text-xs text-danger">{{ error }}</p>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue';
import { labelsApi } from '../api/labels';

const props = defineProps({
  id: {
    type: String,
    default: undefined
  },
  // 付けているラベルの名前
  modelValue: {
    type: Array,
    required: true
  }
});

const emit = defineEmits(['update:modelValue']);

// ラベル名の最大の文字数（bff と同じ）
const LABEL_MAX_LENGTH = 255;

const inputRef = ref(null);
const text = ref('');
const focused = ref(false);
const activeIndex = ref(0);
const error = ref('');
// 補完候補（いずれかのタスクに付いているラベル）
const existing = ref([]);

const sameLabel = (a, b) => a.toLowerCase() === b.toLowerCase();

// 入力中の文字を含む既存のラベル（付けているものを除く）と、どれとも一致しなければ新しく作る候補
const options = computed(() => {
  const query = text.value.trim();
  const matches = existing.value
    .filter(label => !props.modelValue.some(selected => sameLabel(selected, label)))
    .filter(label => !query || label.toLowerCase().includes(query.toLowerCase()))
    .map(label => ({ value: label, isNew: false }));
  if (query && !existing.value.some(label => sameLabel(label, query)) && !props.modelValue.some(label => sameLabel(label, query))) {
    matches.push({ value: query, isNew: true });
  }
  return matches;
});

const showSuggestions = computed(() => focused.value && options.value.length > 0);

watch(options, () => {
  activeIndex.value = 0;
});

// ラベルを付ける。Jira と同じく空白は使えないので取り除く。既存のラベルと大文字・小文字だけ違う場合は既存の名前に合わせる
const add = (value) => {
  const name = value.replace(/\s/g, '');
  text.value = '';
  if (!name) return;
  if (name.length > LABEL_MAX_LENGTH) {
    error.value = `ラベルは ${LABEL_MAX_LENGTH} 文字以内にしてください`;
    return;
  }
  error.value = '';
  if (props.modelValue.some(label => sameLabel(label, name))) return;
  const known = existing.value.find(label => sameLabel(label, name));
  emit('update:modelValue', [...props.modelValue, known ?? name]);
};

const remove = (label) => {
  emit('update:modelValue', props.modelValue.filter(selected => selected !== label));
};

const handleKeydown = (event) => {
  // 日本語入力の変換確定の Enter では付けない
  if (event.isComposing || event.keyCode === 229) return;
  if (event.key === 'ArrowDown' && options.value.length) {
    event.preventDefault();
    activeIndex.value = (activeIndex.value + 1) % options.value.length;
  } else if (event.key === 'ArrowUp' && options.value.length) {
    event.preventDefault();
    activeIndex.value = (activeIndex.value - 1 + options.value.length) % options.value.length;
  } else if (event.key === 'Enter') {
    event.preventDefault();
    const option = text.value.trim() ? options.value[activeIndex.value] : null;
    add(option ? option.value : text.value);
  } else if (event.key === ' ' || event.key === ',') {
    // 空白やカンマで区切って続けて入力できる
    event.preventDefault();
    add(text.value);
  } else if (event.key === 'Backspace' && !text.value && props.modelValue.length) {
    remove(props.modelValue[props.modelValue.length - 1]);
  }
};

// 入力欄を離れたら、入力途中のラベルも付ける（保存ボタンを押したときに消えないように）
const handleBlur = () => {
  focused.value = false;
  if (text.value.trim()) {
    add(text.value);
  }
};

onMounted(async () => {
  try {
    existing.value = await labelsApi.fetchAll();
  } catch (err) {
    // 候補が出ないだけで、入力はできる
    console.error('Failed to fetch labels:', err);
  }
});
</script>
