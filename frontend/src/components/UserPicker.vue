<template>
  <!-- 担当者・報告者の選択。今のユーザーを表示し、押すと表示名・メールアドレスで検索して選べる -->
  <div class="relative min-w-0">
    <input
      v-if="open"
      :id="id"
      ref="searchInput"
      v-model="query"
      type="text"
      placeholder="ユーザーを検索"
      role="combobox"
      :aria-expanded="candidates.length > 0"
      class="w-full px-3 py-1.5 border-2 border-accent rounded-lg text-sm text-fg bg-surface focus:outline-none"
      @keydown.down.prevent="moveActive(1)"
      @keydown.up.prevent="moveActive(-1)"
      @keydown.enter="handleEnter"
      @keydown.esc.prevent="close"
      @blur="close"
    />
    <button
      v-else
      :id="id"
      type="button"
      class="w-full -mx-2 px-2 py-1 flex items-center gap-2 bg-transparent border-0 rounded-md text-left cursor-pointer hover:bg-raised"
      title="クリックして変更"
      @click="start"
    >
      <template v-if="user">
        <UserAvatar :user="user" size="sm" />
        <span class="min-w-0 truncate text-sm text-fg">{{ user.DisplayName }}</span>
      </template>
      <template v-else>
        <!-- Jira と同じく、なしはグレーの人型のアイコン -->
        <span class="shrink-0 w-6 h-6 rounded-full bg-line text-muted flex items-center justify-center" aria-hidden="true">
          <svg class="w-4 h-4 block" viewBox="0 0 24 24" fill="currentColor">
            <circle cx="12" cy="8" r="4" />
            <path d="M4 20c0-4 3.6-6 8-6s8 2 8 6z" />
          </svg>
        </span>
        <span class="text-sm text-muted">{{ emptyLabel }}</span>
      </template>
    </button>

    <ul
      v-if="open && candidates.length"
      role="listbox"
      class="absolute left-0 right-0 top-full mt-1 m-0 p-1 list-none bg-surface border border-line rounded-lg shadow-lg z-10 max-h-64 overflow-y-auto"
    >
      <li
        v-for="(candidate, index) in candidates"
        :key="candidate.ID"
        role="option"
        :aria-selected="index === activeIndex"
        :class="['flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer', index === activeIndex ? 'bg-accent/10' : 'hover:bg-raised']"
        @mousedown.prevent="select(candidate)"
        @mouseenter="activeIndex = index"
      >
        <template v-if="candidate.ID">
          <UserAvatar :user="candidate" size="sm" />
          <span class="min-w-0 flex-1">
            <span class="block truncate text-sm text-fg">{{ candidate.DisplayName }}</span>
            <span class="block truncate text-xs text-muted">{{ candidate.Email }}</span>
          </span>
        </template>
        <span v-else class="text-sm text-muted">{{ emptyLabel }}</span>
      </li>
    </ul>
    <p v-else-if="open && loadError" class="absolute left-0 right-0 top-full mt-1 m-0 px-3 py-2 bg-surface border border-line rounded-lg shadow-lg z-10 text-sm text-danger">
      {{ loadError }}
    </p>
    <p v-else-if="open && query.trim()" class="absolute left-0 right-0 top-full mt-1 m-0 px-3 py-2 bg-surface border border-line rounded-lg shadow-lg z-10 text-sm text-muted">
      一致するユーザーがいません
    </p>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue';
import UserAvatar from './UserAvatar.vue';
import { usersApi } from '../api/users';

const props = defineProps({
  id: {
    type: String,
    required: true
  },
  // 今のユーザー（{ ID, DisplayName, AvatarColor }）。なしなら null
  user: {
    type: Object,
    default: null
  },
  // なしの表示（担当者は「未割り当て」）
  emptyLabel: {
    type: String,
    default: 'なし'
  }
});

// select: 選んだユーザーの ID（なしは 0）
const emit = defineEmits(['select']);

// ユーザーの一覧は画面全体で共有する。開くたびに取り直し（新しく登録したユーザーも出す）、取得中は前の一覧を出す
const users = useState('users', () => null);
const loadError = ref('');

const open = ref(false);
const query = ref('');
const activeIndex = ref(0);
const searchInput = ref(null);

const loadUsers = async () => {
  loadError.value = '';
  try {
    users.value = await usersApi.fetchAll();
  } catch (err) {
    loadError.value = 'ユーザーの取得に失敗しました';
    console.error('Failed to fetch users:', err);
  }
};

// 表示名・メールアドレスの部分一致（大文字・小文字は区別しない）。検索していないときは先頭に「なし」を出す
const candidates = computed(() => {
  const q = query.value.trim().toLowerCase();
  const matched = (users.value || []).filter(u =>
    !q || u.DisplayName.toLowerCase().includes(q) || u.Email.toLowerCase().includes(q)
  );
  return q ? matched : [{ ID: 0 }, ...matched];
});

const start = async () => {
  open.value = true;
  query.value = '';
  activeIndex.value = 0;
  await nextTick();
  searchInput.value?.focus();
  await loadUsers();
};

const close = () => {
  open.value = false;
};

watch(query, () => {
  activeIndex.value = 0;
});

const moveActive = (delta) => {
  const count = candidates.value.length;
  if (!count) return;
  activeIndex.value = (activeIndex.value + delta + count) % count;
};

const select = (candidate) => {
  close();
  if (candidate.ID !== (props.user?.ID ?? 0)) {
    emit('select', candidate.ID);
  }
};

// 日本語入力の変換確定の Enter では選ばない
const handleEnter = (event) => {
  if (event.isComposing || event.keyCode === 229) return;
  event.preventDefault();
  const candidate = candidates.value[activeIndex.value];
  if (candidate) select(candidate);
};
</script>
