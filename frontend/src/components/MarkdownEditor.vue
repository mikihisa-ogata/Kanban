<template>
  <!-- Markdown の入力欄。「編集」と「プレビュー」を切り替えられる。「@」を打つとメンションするユーザーの候補を出す -->
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
          mode === tab.value ? 'bg-line text-fg' : 'bg-transparent text-muted hover:bg-raised'
        ]"
        @click="switchMode(tab.value)"
      >
        {{ tab.label }}
      </button>
      <span class="ml-auto self-center text-xs text-muted">Markdown が使えます</span>
    </div>
    <div v-show="mode === 'edit'" class="relative">
      <textarea
        :id="id"
        ref="textareaRef"
        :value="modelValue"
        :rows="rows"
        :placeholder="placeholder"
        :aria-label="ariaLabel"
        :aria-expanded="mentionCandidates.length > 0"
        aria-autocomplete="list"
        class="block w-full px-3 py-2.5 border-2 border-line rounded-lg text-sm text-fg bg-surface resize-y transition-all duration-200 focus:outline-none focus:border-accent focus:shadow-sm focus:shadow-accent/20"
        @input="handleInput"
        @keydown="handleKeydown"
        @click="detectMention"
        @blur="closeMention"
      ></textarea>
      <!-- メンションの候補。表示名・メールアドレスの一部で絞り込む -->
      <ul
        v-if="mentionCandidates.length"
        role="listbox"
        aria-label="メンションするユーザー"
        class="absolute left-0 top-full mt-1 m-0 p-1 w-72 max-w-full list-none bg-surface border border-line rounded-lg shadow-lg z-20 max-h-64 overflow-y-auto"
      >
        <li
          v-for="(user, index) in mentionCandidates"
          :key="user.ID"
          role="option"
          :aria-selected="index === mentionIndex"
          :class="['flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer', index === mentionIndex ? 'bg-accent/10' : 'hover:bg-raised']"
          @mousedown.prevent="insertMention(user)"
          @mouseenter="mentionIndex = index"
        >
          <UserAvatar :user="user" size="sm" />
          <span class="min-w-0 flex-1">
            <span class="block truncate text-sm text-fg">{{ user.DisplayName }}</span>
            <span class="block truncate text-xs text-muted">{{ user.Email }}</span>
          </span>
        </li>
      </ul>
    </div>
    <div
      v-if="mode === 'preview'"
      class="min-h-24 px-3 py-2.5 border-2 border-dashed border-line rounded-lg bg-surface"
    >
      <MarkdownView v-if="modelValue.trim()" :source="modelValue" />
      <p v-else class="m-0 text-sm text-muted">プレビューする内容がありません</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue';
import MarkdownView from './MarkdownView.vue';
import UserAvatar from './UserAvatar.vue';
import { usersApi } from '../api/users';
import { mentionOf } from '../utils/mentions';

const props = defineProps({
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

// keydown: テキストエリアのキー操作（保存のショートカットなどを呼び出し側で扱う）。メンションの候補を選ぶキーは渡さない
const emit = defineEmits(['update:modelValue', 'keydown']);

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

// メンションの候補に出す最大の人数
const MENTION_LIMIT = 8;

// ユーザーの一覧は担当者の選択（UserPicker）と共有する
const users = useState('users', () => null);
// 入力中のメンション（「@」の位置と、その後に打った文字）。入力中でなければ null
const mention = ref(null);
const mentionIndex = ref(0);

const loadUsers = async () => {
  try {
    users.value = await usersApi.fetchAll();
  } catch (err) {
    console.error('Failed to fetch users:', err);
  }
};

const mentionCandidates = computed(() => {
  if (!mention.value || !users.value) return [];
  const q = mention.value.query.toLowerCase();
  return users.value
    .filter(u => !q || u.DisplayName.toLowerCase().includes(q) || u.Email.toLowerCase().includes(q))
    .slice(0, MENTION_LIMIT);
});

// カーソルの直前が「@」と続けて打った文字（行頭か空白・括弧・句読点の後の「@」）なら、メンションの入力中とみなす
const detectMention = () => {
  const el = textareaRef.value;
  if (!el || el.selectionStart !== el.selectionEnd) {
    mention.value = null;
    return;
  }
  const before = el.value.slice(0, el.selectionStart);
  const match = /(?:^|[\s(（、。])@([^\s@[\]()]{0,30})$/.exec(before);
  if (!match) {
    mention.value = null;
    return;
  }
  const query = match[1];
  if (!mention.value || mention.value.query !== query) {
    mentionIndex.value = 0;
  }
  mention.value = { start: el.selectionStart - query.length - 1, end: el.selectionStart, query };
  if (!users.value) {
    loadUsers();
  }
};

const closeMention = () => {
  mention.value = null;
};

const handleInput = (event) => {
  emit('update:modelValue', event.target.value);
  detectMention();
};

// 候補が出ている間は、↑↓で選び、Enter か Tab で入れ、Esc で閉じる（日本語入力の変換中は除く）
const handleKeydown = (event) => {
  const candidates = mentionCandidates.value;
  if (candidates.length && !(event.isComposing || event.keyCode === 229)) {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      const delta = event.key === 'ArrowDown' ? 1 : -1;
      mentionIndex.value = (mentionIndex.value + delta + candidates.length) % candidates.length;
      return;
    }
    if (event.key === 'Enter' || event.key === 'Tab') {
      event.preventDefault();
      insertMention(candidates[mentionIndex.value]);
      return;
    }
    if (event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      closeMention();
      return;
    }
  }
  emit('keydown', event);
};

// 「@」と打った文字を、ユーザーへのメンション（@[表示名](user:ID)）と空白に置き換える
const insertMention = async (user) => {
  if (!mention.value) return;
  const { start, end } = mention.value;
  const text = mentionOf(user) + ' ';
  const value = props.modelValue.slice(0, start) + text + props.modelValue.slice(end);
  mention.value = null;
  emit('update:modelValue', value);
  await nextTick();
  const el = textareaRef.value;
  if (el) {
    el.focus();
    el.setSelectionRange(start + text.length, start + text.length);
  }
};

// 呼び出し側から入力欄にフォーカスを移す
const focus = () => {
  textareaRef.value?.focus();
};

defineExpose({ focus });
</script>
