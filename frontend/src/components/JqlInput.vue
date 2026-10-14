<template>
  <!-- 課題一覧の詳細検索（JQL）の入力欄。入力中は項目名・演算子・値の候補を出し、Enter で検索する -->
  <div ref="root" class="relative flex-1 min-w-72">
    <div class="flex items-start gap-2">
      <div class="relative flex-1">
        <textarea
          ref="input"
          :value="modelValue"
          rows="2"
          spellcheck="false"
          placeholder="例: project = DEV AND status != Done AND assignee = currentUser() ORDER BY priority DESC"
          aria-label="JQL"
          aria-autocomplete="list"
          :aria-expanded="showSuggestions"
          :aria-invalid="!!error"
          :class="[
            'block w-full px-3 py-2 border-0 rounded-md font-mono text-sm text-gray-900 bg-white/90 resize-y focus:outline-none focus:ring-2',
            error ? 'ring-2 ring-red-400 focus:ring-red-400' : 'focus:ring-blue-300'
          ]"
          @input="handleInput"
          @keydown="handleKeydown"
          @click="refreshSuggestions"
          @focus="refreshSuggestions"
          @compositionend="refreshSuggestions"
        />
        <ul
          v-if="showSuggestions"
          class="absolute left-0 top-full mt-1 w-80 max-h-72 overflow-y-auto m-0 p-1 list-none bg-white border border-gray-200 rounded-lg shadow-lg z-30"
          role="listbox"
          aria-label="JQL の候補"
        >
          <li
            v-for="(item, index) in suggestions.items"
            :key="`${item.label}-${index}`"
            role="option"
            :aria-selected="index === highlighted"
            :class="['flex items-center gap-3 px-2 py-1.5 rounded text-sm cursor-pointer', index === highlighted ? 'bg-blue-50 text-blue-700' : 'text-gray-800 hover:bg-gray-100']"
            @mousedown.prevent="accept(item)"
            @mouseenter="highlighted = index"
          >
            <span class="font-mono min-w-0 truncate">{{ item.label }}</span>
            <span v-if="item.detail" class="ml-auto shrink-0 text-xs text-gray-400">{{ item.detail }}</span>
          </li>
        </ul>
      </div>
      <button
        type="button"
        class="px-4 py-2 border-0 rounded-md bg-blue-600 text-sm font-semibold text-white cursor-pointer hover:bg-blue-700"
        @click="submit"
      >
        検索
      </button>
    </div>
    <button
      v-if="error"
      type="button"
      class="mt-2 block w-full p-0 bg-transparent border-0 text-left cursor-pointer"
      title="誤りの位置へ移動"
      @click="focusError"
    >
      <span class="block text-sm font-semibold text-danger">{{ error.message }}</span>
      <span v-if="errorExcerpt" class="mt-1 inline-block px-2 py-0.5 rounded bg-white/90 font-mono text-xs text-gray-700 whitespace-pre">{{ errorExcerpt.before }}<mark class="bg-red-200 text-red-800 rounded-sm">{{ errorExcerpt.at }}</mark>{{ errorExcerpt.after }}</span>
    </button>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onBeforeUnmount } from 'vue';
import { jqlSuggestions, applyJqlSuggestion } from '~/utils/jql';

const props = defineProps({
  modelValue: { type: String, required: true },
  // 補完の候補に使う一覧 { spaces, statuses, users, labels, epics, sprints, versions }
  masters: { type: Object, required: true },
  // JQL の誤り { message, position }（position は先頭からの文字数で 0 から。ない場合もある）
  error: { type: Object, default: null }
});

const emit = defineEmits(['update:modelValue', 'submit']);

const root = ref(null);
const input = ref(null);
const suggestions = ref({ from: 0, to: 0, items: [] });
const highlighted = ref(0);
const open = ref(false);

const showSuggestions = computed(() => open.value && suggestions.value.items.length > 0);

// 位置は文字（コードポイント）で数える（bff の誤りの位置と同じ）。textarea の位置は UTF-16 なので変換する
const toCharIndex = (text, utf16Index) => [...text.slice(0, utf16Index)].length;
const toUtf16Index = (text, charIndex) => [...text].slice(0, charIndex).join('').length;

const refreshSuggestions = () => {
  const el = input.value;
  if (!el) return;
  const caret = toCharIndex(el.value, el.selectionStart);
  suggestions.value = jqlSuggestions(el.value, caret, props.masters);
  highlighted.value = 0;
  open.value = document.activeElement === el;
};

const handleInput = (event) => {
  emit('update:modelValue', event.target.value);
  // v-model の値が戻ってくる前でも、入力欄の今の文字列で候補を出す
  if (!event.isComposing) refreshSuggestions();
};

const accept = async (item) => {
  const { value, caret } = applyJqlSuggestion(props.modelValue, suggestions.value, item);
  emit('update:modelValue', value);
  await nextTick();
  const el = input.value;
  if (!el) return;
  el.focus();
  const position = toUtf16Index(value, caret);
  el.setSelectionRange(position, position);
  refreshSuggestions();
};

const submit = () => {
  open.value = false;
  emit('submit');
};

const handleKeydown = (event) => {
  // 日本語の変換中のキーは入力欄に任せる
  if (event.isComposing || event.keyCode === 229) return;
  if (event.key === ' ' && (event.ctrlKey || event.metaKey)) {
    event.preventDefault();
    refreshSuggestions();
    return;
  }
  if (showSuggestions.value) {
    const count = suggestions.value.items.length;
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      highlighted.value = (highlighted.value + (event.key === 'ArrowDown' ? 1 : count - 1)) % count;
      return;
    }
    if (event.key === 'Tab' || (event.key === 'Enter' && !event.ctrlKey && !event.metaKey)) {
      event.preventDefault();
      accept(suggestions.value.items[highlighted.value]);
      return;
    }
    if (event.key === 'Escape') {
      event.preventDefault();
      open.value = false;
      return;
    }
  }
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault();
    submit();
    return;
  }
  // カーソルの移動では候補を出し直す
  if (event.key === 'ArrowLeft' || event.key === 'ArrowRight' || event.key === 'Home' || event.key === 'End') {
    nextTick(refreshSuggestions);
  }
};

// 誤りの位置の前後を抜き出して見せる
const errorExcerpt = computed(() => {
  if (!props.error || typeof props.error.position !== 'number') return null;
  const chars = [...props.modelValue];
  const position = Math.min(props.error.position, chars.length);
  const start = Math.max(0, position - 24);
  return {
    before: (start > 0 ? '…' : '') + chars.slice(start, position).join(''),
    at: chars[position] ?? ' ',
    after: chars.slice(position + 1, position + 24).join('') + (position + 24 < chars.length ? '…' : '')
  };
});

const focusError = () => {
  const el = input.value;
  if (!el || !props.error || typeof props.error.position !== 'number') return;
  el.focus();
  const start = toUtf16Index(props.modelValue, props.error.position);
  const end = toUtf16Index(props.modelValue, props.error.position + 1);
  el.setSelectionRange(start, end);
  open.value = false;
};

// 入力欄の外を押したら候補を閉じる
const handleOutside = (event) => {
  if (root.value && !root.value.contains(event.target)) open.value = false;
};
onMounted(() => document.addEventListener('mousedown', handleOutside));
onBeforeUnmount(() => document.removeEventListener('mousedown', handleOutside));
</script>
