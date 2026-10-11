<template>
  <!-- リンクされた課題のセクション。課題の詳細（ページとモーダル）で使い、課題キーか要約で検索して追加・削除できる -->
  <section>
    <div class="flex items-center gap-2 mb-2">
      <h2 class="m-0 text-sm font-semibold text-gray-700">リンクされた課題</h2>
      <button
        v-if="!adding"
        type="button"
        class="ml-auto bg-transparent border-0 p-0 w-6 h-6 flex items-center justify-center rounded text-gray-500 text-lg leading-none cursor-pointer hover:bg-gray-100 hover:text-gray-700"
        title="課題をリンク"
        @click="startAdding"
      >
        +
      </button>
    </div>

    <!-- 名前（ブロックする・関連する…）ごとにまとめる。並びは bff が返す順 -->
    <div v-for="group in groups" :key="group.relation" class="mb-2">
      <h3 class="m-0 mb-1 text-xs font-semibold text-gray-500">{{ linkRelationLabel(group.relation) }}</h3>
      <ul class="m-0 p-0 list-none border border-gray-200 rounded-lg divide-y divide-gray-200">
        <li v-for="link in group.links" :key="link.ID" class="flex items-center gap-2 px-3 py-2">
          <IssueTypeIcon :type="link.Issue.IssueType" />
          <button
            type="button"
            class="shrink-0 bg-transparent border-0 p-0 font-mono text-xs font-semibold text-gray-500 cursor-pointer hover:text-blue-600 hover:underline"
            :title="`${link.Issue.Key} を開く`"
            @click="$emit('open', link.Issue.Key)"
          >
            {{ link.Issue.Key }}
          </button>
          <span :class="['flex-1 min-w-0 truncate text-sm', isDoneCategory(link.Issue.StatusCategory) ? 'text-gray-400 line-through' : 'text-gray-900']">
            {{ link.Issue.Title }}
          </span>
          <PriorityIcon :priority="link.Issue.Priority" />
          <span :class="['shrink-0 max-w-32 truncate px-1.5 py-0.5 rounded text-xs font-semibold', STATUS_CATEGORY_BADGE_CLASS[link.Issue.StatusCategory] || 'bg-gray-100 text-gray-600']">{{ link.Issue.StatusName }}</span>
          <button
            type="button"
            class="shrink-0 bg-transparent border-0 p-0 w-6 h-6 flex items-center justify-center rounded text-gray-400 text-lg leading-none cursor-pointer hover:bg-red-100 hover:text-red-500"
            :title="`${link.Issue.Key} とのリンクを削除`"
            @click="deleteLink(link)"
          >
            ×
          </button>
        </li>
      </ul>
    </div>
    <p v-if="!groups.length && !adding" class="m-0 text-sm text-gray-400">リンクされた課題はありません</p>

    <form v-if="adding" class="mt-2 flex flex-col gap-2" @submit.prevent="addLink">
      <div class="flex items-start gap-2">
        <select
          v-model="relation"
          aria-label="リンクの種類"
          class="shrink-0 px-2 py-1.5 border-2 border-gray-300 rounded-lg text-sm text-gray-800 bg-white cursor-pointer hover:border-gray-400 focus:outline-none focus:border-blue-500"
        >
          <option v-for="option in LINK_RELATION_OPTIONS" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
        <div class="relative flex-1 min-w-0">
          <!-- 選んだ課題。押すと選び直せる -->
          <button
            v-if="selected"
            type="button"
            class="w-full px-3 py-1.5 flex items-center gap-2 border-2 border-blue-500 rounded-lg bg-white text-left cursor-pointer"
            title="別の課題を選ぶ"
            @click="clearSelection"
          >
            <IssueTypeIcon :type="selected.IssueType" />
            <span class="shrink-0 font-mono text-xs font-semibold text-gray-500">{{ selected.Key }}</span>
            <span class="flex-1 min-w-0 truncate text-sm text-gray-900">{{ selected.Title }}</span>
          </button>
          <template v-else>
            <input
              ref="searchInput"
              v-model="query"
              type="text"
              placeholder="課題キーまたは要約で検索"
              aria-label="リンクする課題を検索"
              role="combobox"
              :aria-expanded="candidates.length > 0"
              class="w-full px-3 py-1.5 border-2 border-gray-300 rounded-lg text-sm text-gray-900 bg-white focus:outline-none focus:border-blue-500"
              @keydown.down.prevent="moveActive(1)"
              @keydown.up.prevent="moveActive(-1)"
              @keydown.enter="handleSearchEnter"
              @keydown.esc="handleEsc"
            />
            <ul
              v-if="candidates.length"
              role="listbox"
              class="absolute left-0 right-0 top-full mt-1 m-0 p-1 list-none bg-white border border-gray-200 rounded-lg shadow-lg z-10 max-h-64 overflow-y-auto"
            >
              <li
                v-for="(todo, index) in candidates"
                :key="todo.ID"
                role="option"
                :aria-selected="index === activeIndex"
                :class="['flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer', index === activeIndex ? 'bg-blue-50' : 'hover:bg-gray-100']"
                @mousedown.prevent="select(todo)"
                @mouseenter="activeIndex = index"
              >
                <IssueTypeIcon :type="todo.IssueType" />
                <span class="shrink-0 font-mono text-xs font-semibold text-gray-500">{{ todo.Key }}</span>
                <span class="flex-1 min-w-0 truncate text-sm text-gray-900">{{ todo.Title }}</span>
              </li>
            </ul>
            <p v-else-if="query.trim()" class="absolute left-0 right-0 top-full mt-1 m-0 px-3 py-2 bg-white border border-gray-200 rounded-lg shadow-lg z-10 text-sm text-gray-500">
              一致する課題がありません
            </p>
          </template>
        </div>
      </div>
      <div class="flex gap-2">
        <button
          type="submit"
          :disabled="!selected || busy"
          class="px-3 py-1.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          リンク
        </button>
        <button
          type="button"
          class="px-3 py-1.5 bg-transparent text-gray-700 border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-gray-100"
          @click="stopAdding"
        >
          キャンセル
        </button>
      </div>
    </form>
    <p v-if="error" class="mt-1 mb-0 text-xs text-red-600">{{ error }}</p>
  </section>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue';
import IssueTypeIcon from './IssueTypeIcon.vue';
import PriorityIcon from './PriorityIcon.vue';
import { todosApi } from '../api/todos';
import { isDoneCategory, STATUS_CATEGORY_BADGE_CLASS } from '../utils/statuses';
import { LINK_RELATION_OPTIONS, DEFAULT_LINK_RELATION, linkRelationLabel } from '../utils/issueLinks';

const props = defineProps({
  // リンクを表示・追加する課題
  task: {
    type: Object,
    required: true
  },
  // リンク相手の検索対象（全課題）
  todos: {
    type: Array,
    default: () => []
  }
});

// changed: リンクを追加・削除した（相手の課題のリンクも変わるので、呼び出し側で課題を取り直す）
// open: リンク相手の課題を開く（課題キー）
const emit = defineEmits(['changed', 'open']);

// 検索候補の最大件数
const MAX_CANDIDATES = 8;

const groups = computed(() => {
  const result = [];
  for (const link of props.task.Links || []) {
    const last = result[result.length - 1];
    if (last && last.relation === link.Relation) {
      last.links.push(link);
    } else {
      result.push({ relation: link.Relation, links: [link] });
    }
  }
  return result;
});

const adding = ref(false);
const relation = ref(DEFAULT_LINK_RELATION);
const query = ref('');
const selected = ref(null);
const activeIndex = ref(0);
const busy = ref(false);
const error = ref('');
const searchInput = ref(null);

// 自分以外の課題から、課題キーか要約に検索語を含むものを出す。キーが一致するものを先にする
const candidates = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return [];
  const matches = props.todos.filter(todo =>
    todo.ID !== props.task.ID && (todo.Key.toLowerCase().includes(q) || todo.Title.toLowerCase().includes(q))
  );
  const keyMatches = matches.filter(todo => todo.Key.toLowerCase().includes(q));
  const titleMatches = matches.filter(todo => !todo.Key.toLowerCase().includes(q));
  return [...keyMatches, ...titleMatches].slice(0, MAX_CANDIDATES);
});

watch(query, () => {
  activeIndex.value = 0;
});

const focusSearch = async () => {
  await nextTick();
  searchInput.value?.focus();
};

const startAdding = () => {
  adding.value = true;
  error.value = '';
  focusSearch();
};

const stopAdding = () => {
  adding.value = false;
  query.value = '';
  selected.value = null;
  relation.value = DEFAULT_LINK_RELATION;
};

const select = (todo) => {
  selected.value = todo;
  query.value = '';
};

const clearSelection = () => {
  selected.value = null;
  focusSearch();
};

const moveActive = (delta) => {
  if (!candidates.value.length) return;
  activeIndex.value = (activeIndex.value + delta + candidates.value.length) % candidates.value.length;
};

// 日本語入力の変換確定の Enter や Esc では選択・取り消しをしない
const isComposing = (event) => event.isComposing || event.keyCode === 229;

// 検索中の Enter は候補を選ぶ（フォームは送信しない）
const handleSearchEnter = (event) => {
  event.preventDefault();
  if (isComposing(event)) return;
  const todo = candidates.value[activeIndex.value];
  if (todo) select(todo);
};

const handleEsc = (event) => {
  if (isComposing(event)) return;
  event.preventDefault();
  if (query.value) {
    query.value = '';
  } else {
    stopAdding();
  }
};

const addLink = async () => {
  if (!selected.value || busy.value) return;
  busy.value = true;
  try {
    error.value = '';
    await todosApi.addLink(props.task.ID, relation.value, selected.value.ID);
    stopAdding();
    emit('changed');
  } catch (err) {
    error.value = '課題のリンクに失敗しました: ' + err.message;
    console.error('Failed to add link:', err);
  } finally {
    busy.value = false;
  }
};

const deleteLink = async (link) => {
  try {
    error.value = '';
    await todosApi.deleteLink(link.ID);
    emit('changed');
  } catch (err) {
    error.value = 'リンクの削除に失敗しました: ' + err.message;
    console.error('Failed to delete link:', err);
    emit('changed');
  }
};

// 別の課題に切り替わったら追加をやめる
watch(() => props.task.ID, () => {
  stopAdding();
  error.value = '';
});
</script>
