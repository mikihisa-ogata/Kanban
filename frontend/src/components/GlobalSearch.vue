<template>
  <!-- 全体の検索（Jira のヘッダーの検索）。サイドバーの検索・`/`・⌘K で開く。
       空のときは最近表示した課題、入力中は課題（キー・要約）とスペースの候補を出す。課題キーをそのまま入れて Enter で開く -->
  <div
    v-if="open"
    class="fixed inset-0 bg-overlay flex items-start justify-center z-50 px-4 pt-[12vh]"
    @click.self="close"
  >
    <div
      role="dialog"
      aria-modal="true"
      aria-label="検索"
      class="w-full max-w-xl bg-surface border border-line rounded-xl shadow-2xl overflow-hidden text-fg"
    >
      <div class="flex items-center gap-2 px-4 border-b border-line">
        <svg class="shrink-0 w-4 h-4 text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <circle cx="11" cy="11" r="7" />
          <path d="M21 21l-4.35-4.35" />
        </svg>
        <input
          ref="input"
          v-model="query"
          type="text"
          placeholder="課題キー・要約・スペースで検索"
          role="combobox"
          aria-label="検索"
          aria-controls="global-search-results"
          :aria-expanded="items.length > 0"
          :aria-activedescendant="items.length ? `global-search-item-${activeIndex}` : undefined"
          class="flex-1 min-w-0 py-3.5 border-0 bg-transparent text-base text-fg focus:outline-none"
          @keydown.down.prevent="moveActive(1)"
          @keydown.up.prevent="moveActive(-1)"
          @keydown.enter="handleEnter"
          @keydown.esc.prevent="close"
        />
        <kbd class="shrink-0 px-1.5 rounded border border-line font-sans text-[11px] text-muted">Esc</kbd>
      </div>

      <div id="global-search-results" role="listbox" aria-label="検索の候補" class="max-h-[60vh] overflow-y-auto py-2">
        <p v-if="loadError" class="m-0 px-4 py-2 text-sm text-danger">{{ loadError }}</p>
        <template v-for="section in sections" :key="section.title">
          <h2 class="m-0 px-4 pt-2 pb-1 text-xs font-semibold text-muted">{{ section.title }}</h2>
          <button
            v-for="item in section.items"
            :id="`global-search-item-${item.index}`"
            :key="item.id"
            type="button"
            role="option"
            :aria-selected="item.index === activeIndex"
            :class="[
              'w-full flex items-center gap-3 px-4 py-2 border-0 text-left text-sm cursor-pointer',
              item.index === activeIndex ? 'bg-raised' : 'bg-transparent'
            ]"
            @mouseenter="activeIndex = item.index"
            @click="openItem(item)"
          >
            <template v-if="item.kind === 'issue'">
              <IssueTypeIcon :type="item.issue.IssueType" />
              <span class="shrink-0 font-mono text-xs text-muted">{{ item.issue.Key }}</span>
              <span :class="['min-w-0 flex-1 truncate', item.issue.Done ? 'line-through text-muted' : 'text-fg']">{{ item.issue.Title }}</span>
              <span v-if="item.issue.StatusName" class="shrink-0 text-xs text-muted">{{ item.issue.StatusName }}</span>
            </template>
            <template v-else-if="item.kind === 'space'">
              <span class="shrink-0 w-4 h-4 rounded-sm" :style="{ backgroundColor: spaceColor(item.space) }" aria-hidden="true" />
              <span class="min-w-0 flex-1 truncate text-fg">{{ item.space.Title }}</span>
              <span class="shrink-0 font-mono text-xs text-muted">{{ item.space.Key }}</span>
            </template>
            <template v-else-if="item.kind === 'key'">
              <span class="shrink-0 w-4 h-4 flex items-center justify-center text-muted" aria-hidden="true">→</span>
              <span class="min-w-0 flex-1 truncate text-fg"><span class="font-mono">{{ item.key }}</span> を開く</span>
            </template>
            <template v-else>
              <svg class="shrink-0 w-4 h-4 text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <circle cx="11" cy="11" r="7" />
                <path d="M21 21l-4.35-4.35" />
              </svg>
              <span class="min-w-0 flex-1 truncate text-fg">課題一覧で「{{ item.text }}」を検索</span>
            </template>
          </button>
        </template>
        <p v-if="!loading && !loadError && !items.length" class="m-0 px-4 py-2 text-sm text-muted">
          {{ query.trim() ? '一致する課題・スペースはありません' : '最近表示した課題はありません' }}
        </p>
        <p v-if="loading && !items.length" class="m-0 px-4 py-2 text-sm text-muted">読み込み中...</p>
      </div>

      <div class="flex items-center gap-3 px-4 py-2 border-t border-line text-xs text-muted">
        <span><kbd class="px-1 rounded border border-line font-sans">↑</kbd> <kbd class="px-1 rounded border border-line font-sans">↓</kbd> で選択</span>
        <span><kbd class="px-1 rounded border border-line font-sans">Enter</kbd> で開く</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue';
import IssueTypeIcon from './IssueTypeIcon.vue';
import { todosApi } from '../api/todos';
import { spacesApi } from '../api/spaces';
import { boardPath, issuePath } from '../utils/spaceRoutes';
import { spaceColor } from '../utils/spaceColors';
import { looksLikeIssueKey } from '../utils/shortcuts';

// 検索の開閉。サイドバーの検索ボタンとキーボードショートカットと共有する
const open = useState('globalSearchOpen', () => false);

// 候補の件数
const MAX_RECENT = 8;
const MAX_ISSUES = 8;
const MAX_SPACES = 5;
const RECENT_JQL = 'issuekey IN issueHistory() ORDER BY lastViewed DESC';

const input = ref(null);
const query = ref('');
const activeIndex = ref(0);
const todos = ref([]);
const spaces = ref([]);
const recent = ref([]);
const loading = ref(false);
const loadError = ref('');

// 開くたびに最新の課題・スペース・最近表示した課題を読む（他の画面で作成・変更されていることがある）
let loadRequest = 0;
const load = async () => {
  const request = ++loadRequest;
  loading.value = true;
  loadError.value = '';
  try {
    const [allTodos, allSpaces, history] = await Promise.all([
      todosApi.fetchAll(),
      spacesApi.fetchAll(),
      todosApi.searchJql(RECENT_JQL, 1, MAX_RECENT)
    ]);
    if (request !== loadRequest) return;
    todos.value = allTodos;
    spaces.value = allSpaces;
    recent.value = history.Issues || [];
  } catch (err) {
    if (request !== loadRequest) return;
    loadError.value = '課題の取得に失敗しました: ' + err.message;
    console.error('Failed to load global search:', err);
  } finally {
    if (request === loadRequest) loading.value = false;
  }
};

watch(open, async (isOpen) => {
  if (!isOpen) return;
  query.value = '';
  activeIndex.value = 0;
  await nextTick();
  input.value?.focus();
  await load();
});

// 課題の候補。キーの完全一致 → キーの前方一致 → 要約の部分一致の順（大文字・小文字は区別しない）
const matchedIssues = (q) => {
  const ranked = [];
  for (const todo of todos.value) {
    const key = (todo.Key || '').toLowerCase();
    let rank = -1;
    if (key === q) rank = 0;
    else if (key.startsWith(q)) rank = 1;
    else if (todo.Title.toLowerCase().includes(q)) rank = 2;
    if (rank >= 0) ranked.push({ rank, todo });
  }
  // 同じ順位の中では新しい課題（ID の大きい順）を先に出す
  ranked.sort((a, b) => a.rank - b.rank || b.todo.ID - a.todo.ID);
  return ranked.slice(0, MAX_ISSUES).map(r => r.todo);
};

const matchedSpaces = (q) =>
  spaces.value
    .filter(space => space.Title.toLowerCase().includes(q) || space.Key.toLowerCase().includes(q))
    .slice(0, MAX_SPACES);

// 見出しごとの候補。index はキーボードで選ぶときの通し番号
const sections = computed(() => {
  const text = query.value.trim();
  const q = text.toLowerCase();
  const list = [];
  if (!q) {
    list.push({ title: '最近表示した課題', items: recent.value.map(issue => ({ kind: 'issue', id: `recent-${issue.ID}`, issue })) });
  } else {
    const issues = matchedIssues(q);
    const issueItems = issues.map(issue => ({ kind: 'issue', id: `issue-${issue.ID}`, issue }));
    // 課題キーの形で、読み込んだ課題に一致するものがなければ、そのキーのページを開く候補を先頭に出す
    if (looksLikeIssueKey(text) && !issues.some(issue => issue.Key.toLowerCase() === q)) {
      issueItems.unshift({ kind: 'key', id: 'key', key: text.toUpperCase() });
    }
    list.push({ title: '課題', items: issueItems });
    list.push({ title: 'スペース', items: matchedSpaces(q).map(space => ({ kind: 'space', id: `space-${space.ID}`, space })) });
    list.push({ title: '課題一覧', items: [{ kind: 'search', id: 'search', text }] });
  }
  let index = 0;
  return list
    .filter(section => section.items.length)
    .map(section => ({ ...section, items: section.items.map(item => ({ ...item, index: index++ })) }));
});

const items = computed(() => sections.value.flatMap(section => section.items));

watch(query, () => {
  activeIndex.value = 0;
});

const moveActive = (delta) => {
  const count = items.value.length;
  if (!count) return;
  activeIndex.value = (activeIndex.value + delta + count) % count;
};

const close = () => {
  open.value = false;
};

const openItem = (item) => {
  close();
  switch (item.kind) {
    case 'issue':
      navigateTo(issuePath(item.issue.Key));
      break;
    case 'key':
      navigateTo(issuePath(item.key));
      break;
    case 'space':
      navigateTo(boardPath(item.space));
      break;
    default:
      navigateTo({ path: '/issues', query: { text: item.text } });
  }
};

// 日本語入力の変換確定の Enter では開かない
const handleEnter = (event) => {
  if (event.isComposing || event.keyCode === 229) return;
  event.preventDefault();
  const item = items.value[activeIndex.value];
  if (item) openItem(item);
};
</script>
