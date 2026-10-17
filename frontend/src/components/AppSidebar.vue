<template>
  <!-- 画面左のサイドバー。上から スペースの切り替え・検索と作成・ナビ・スペース一覧・ユーザーと通知。
       画面が狭いときはアイコンだけにする。位置（画面の高さに張り付ける）は呼び出し側（SpaceShell）で決める -->
  <aside class="h-screen w-14 md:w-60 bg-sidebar border-r border-line flex flex-col text-fg">
    <!-- スペースの切り替え -->
    <div ref="switcherRoot" class="relative px-2 md:px-3 pt-3 pb-2">
      <button
        type="button"
        class="w-full flex items-center gap-2 px-1.5 py-1.5 border-0 rounded-md bg-transparent text-left text-fg cursor-pointer hover:bg-raised"
        :title="currentSpace ? `${currentSpace.Title}（${currentSpace.Key}）` : 'すべてのスペース'"
        aria-haspopup="listbox"
        :aria-expanded="switcherOpen"
        @click="switcherOpen = !switcherOpen"
      >
        <span
          class="shrink-0 w-6 h-6 rounded-md flex items-center justify-center text-xs font-bold"
          :style="currentSpace ? { backgroundColor: spaceColor(currentSpace), color: readableTextOn(spaceColor(currentSpace)) } : { backgroundColor: 'var(--kb-accent)', color: 'var(--kb-on-accent)' }"
          aria-hidden="true"
        >
          {{ currentSpace ? (currentSpace.Key || currentSpace.Title).slice(0, 1) : '全' }}
        </span>
        <span class="hidden md:block min-w-0 flex-1 truncate text-sm font-semibold">{{ currentSpace ? currentSpace.Title : 'すべてのスペース' }}</span>
        <svg class="hidden md:block shrink-0 w-4 h-4 text-muted" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
          <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
        </svg>
      </button>
      <div
        v-if="switcherOpen"
        class="absolute left-2 md:left-3 top-full mt-1 w-56 max-h-80 overflow-y-auto p-1 bg-surface border border-line rounded-lg shadow-lg z-50"
        role="listbox"
        aria-label="スペースを切り替え"
      >
        <button
          v-for="option in switcherOptions"
          :key="option.id ?? 'all'"
          type="button"
          role="option"
          :aria-selected="(currentSpace ? currentSpace.ID : null) === option.id"
          :class="[
            'w-full flex items-center gap-2 px-2 py-1.5 border-0 rounded text-left text-sm cursor-pointer',
            (currentSpace ? currentSpace.ID : null) === option.id ? 'bg-raised text-fg font-semibold' : 'bg-transparent text-fg hover:bg-raised'
          ]"
          @click="selectSpace(option.id)"
        >
          <span class="shrink-0 w-3 h-3 rounded-sm" :style="{ backgroundColor: option.color }" aria-hidden="true" />
          <span class="min-w-0 flex-1 truncate">{{ option.label }}</span>
          <span v-if="option.key" class="font-mono text-xs text-muted">{{ option.key }}</span>
        </button>
      </div>
    </div>

    <!-- 検索と作成。検索は `/`・⌘K でも開く（KeyboardShortcuts） -->
    <div class="flex flex-col md:flex-row items-center gap-1.5 px-2 md:px-3 pb-3">
      <button
        type="button"
        class="w-10 md:w-auto md:flex-1 min-w-0 flex items-center justify-center md:justify-start gap-2 px-2 py-1.5 border border-line rounded-md bg-surface text-sm text-muted cursor-pointer hover:text-fg"
        title="課題・スペースを検索（/ または ⌘K）"
        @click="openSearch"
      >
        <svg class="shrink-0 w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <circle cx="11" cy="11" r="7" />
          <path d="M21 21l-4.35-4.35" />
        </svg>
        <span class="hidden md:inline flex-1 text-left">検索</span>
        <kbd class="hidden md:inline px-1.5 rounded border border-line font-sans text-[11px] text-muted">⌘K</kbd>
      </button>
      <button
        type="button"
        class="shrink-0 w-10 md:w-8 h-8 flex items-center justify-center p-0 border-0 rounded-md bg-accent text-on-accent cursor-pointer hover:bg-accent-hover"
        title="課題を作成"
        aria-label="課題を作成"
        @click="createIssueOpen = true"
      >
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true">
          <path d="M12 5v14M5 12h14" />
        </svg>
      </button>
    </div>

    <div class="flex-1 min-h-0 overflow-y-auto px-2 md:px-3 pb-3 flex flex-col gap-4">
      <!-- ナビ -->
      <nav class="flex flex-col gap-0.5" aria-label="ページ">
        <template v-for="item in navItems" :key="item.to">
          <NuxtLink
            :to="item.to"
            :title="item.label"
            :aria-current="isActive(item.to) ? 'page' : undefined"
            :class="[
              'flex items-center justify-center md:justify-start gap-2.5 px-2 py-1.5 rounded-md text-sm font-medium no-underline transition-colors duration-150',
              isActive(item.to) ? 'bg-raised text-fg' : 'text-muted hover:bg-raised hover:text-fg'
            ]"
          >
            <svg class="shrink-0 w-4 h-4 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path v-for="(d, i) in item.icon" :key="i" :d="d" />
            </svg>
            <span class="hidden md:inline">{{ item.label }}</span>
          </NuxtLink>
          <!-- 課題一覧の下に、お気に入り（スター）のフィルター -->
          <ul v-if="item.filters && favouriteFilters.length" class="hidden md:flex flex-col gap-0.5 m-0 p-0 pl-6 list-none" aria-label="お気に入りのフィルター">
            <li v-for="filter in favouriteFilters" :key="filter.ID">
              <NuxtLink
                :to="filterLink(filter)"
                :title="filter.JQL ? `${filter.Name}: ${filter.JQL}` : filter.Name"
                :aria-current="isFilterActive(filter) ? 'page' : undefined"
                :class="[
                  'flex items-center gap-2 px-2 py-1 rounded-md text-xs no-underline transition-colors duration-150',
                  isFilterActive(filter) ? 'bg-raised text-fg font-semibold' : 'text-muted hover:bg-raised hover:text-fg'
                ]"
              >
                <svg class="shrink-0 w-3 h-3 text-warning" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                  <path :d="STAR_ICON" />
                </svg>
                <span class="min-w-0 truncate">{{ filter.Name }}</span>
              </NuxtLink>
            </li>
          </ul>
        </template>
      </nav>

      <!-- スペース一覧。課題やエピックをドラッグすると、そのスペースへ移せる -->
      <section aria-labelledby="sidebar-spaces-heading">
        <div class="flex items-center px-2 mb-1">
          <h2 id="sidebar-spaces-heading" class="hidden md:block m-0 flex-1 text-xs font-semibold text-muted">スペース</h2>
          <button
            type="button"
            class="hidden md:flex w-5 h-5 items-center justify-center p-0 border-0 rounded bg-transparent text-muted cursor-pointer hover:bg-raised hover:text-fg"
            :title="creating ? 'スペースの作成をやめる' : 'スペースを作成'"
            :aria-expanded="creating"
            @click="creating = !creating"
          >
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true">
              <path v-if="creating" d="M6 6l12 12M18 6L6 18" />
              <path v-else d="M12 5v14M5 12h14" />
            </svg>
          </button>
        </div>

        <!-- スペースの作成。キーは名前から候補を入れ、手で変えることもできる -->
        <form v-if="creating" class="hidden md:flex flex-col gap-1.5 px-1 mb-2" @submit.prevent="handleCreate">
          <input
            v-model="newTitle"
            type="text"
            placeholder="新しいスペースの名前"
            aria-label="新しいスペースの名前"
            class="w-full px-2 py-1 border border-line rounded-md text-sm text-fg bg-surface focus:outline-none focus:border-accent"
          />
          <div v-if="newTitle.trim()" class="flex items-center gap-2">
            <label for="new-space-key" class="text-xs font-semibold text-muted">キー</label>
            <input
              id="new-space-key"
              :value="newKey"
              type="text"
              maxlength="10"
              placeholder="KAN"
              class="w-20 px-2 py-1 border border-line rounded-md font-mono text-xs uppercase text-fg bg-surface focus:outline-none focus:border-accent"
              @input="handleKeyInput"
            />
            <span v-if="newKey && !keyError" class="text-xs text-muted font-mono">{{ newKey }}-1</span>
          </div>
          <!-- 候補の取得中（キーが空）はエラーを出さず、追加ボタンを押せないだけにする -->
          <p v-if="newTitle.trim() && newKey && keyError" class="m-0 text-xs text-danger">{{ keyError }}</p>
          <button
            type="submit"
            :disabled="!canCreate"
            class="self-end px-2.5 py-1 border-0 rounded-md bg-accent text-on-accent text-xs font-semibold cursor-pointer hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed"
          >
            作成
          </button>
        </form>

        <ul class="m-0 p-0 list-none flex flex-col gap-0.5">
          <li
            v-for="space in spaces"
            :key="space.ID"
            :class="[
              'group flex items-center justify-center md:justify-start gap-2.5 px-2 py-1.5 rounded-md text-sm cursor-pointer transition-colors duration-150',
              currentSpace && currentSpace.ID === space.ID ? 'bg-raised text-fg' : 'text-muted hover:bg-raised hover:text-fg',
              dragOverSpaceId === space.ID ? 'ring-2 ring-accent' : ''
            ]"
            :title="`${space.Title}（${space.Key}）: 課題 ${issueCount(space.ID)} 件。課題やエピックをドラッグすると、このスペースへ移動できます`"
            @click="selectSpace(space.ID)"
            @dragover="handleDragOver($event, space.ID)"
            @dragleave="handleDragLeave(space.ID)"
            @drop="handleDrop($event, space.ID)"
          >
            <span class="shrink-0 w-3 h-3 rounded-sm" :style="{ backgroundColor: spaceColor(space) }" aria-hidden="true" />
            <span class="hidden md:inline min-w-0 flex-1 truncate">{{ space.Title }}</span>
            <span class="hidden md:inline text-xs text-muted group-hover:hidden">{{ issueCount(space.ID) }}</span>
            <span class="hidden md:group-hover:flex items-center gap-1.5">
              <NuxtLink
                :to="settingsPath(space)"
                class="leading-none text-muted hover:text-fg"
                title="スペースの設定"
                @click.stop
              >
                <svg class="w-3.5 h-3.5 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <path v-for="(d, i) in SETTINGS_ICON" :key="i" :d="d" />
                </svg>
              </NuxtLink>
              <button
                type="button"
                class="bg-transparent border-0 p-0 leading-none text-sm text-muted cursor-pointer hover:text-danger"
                title="スペースを削除"
                @click.stop="$emit('delete', space.ID)"
              >
                ×
              </button>
            </span>
          </li>
        </ul>
        <p v-if="!spaces.length" class="hidden md:block m-0 px-2 py-1 text-xs text-muted">スペースはまだありません</p>
      </section>
    </div>

    <!-- ユーザーと通知。ログイン中のユーザーはブラウザ側でだけ取得するので、サーバーの描画には含めない -->
    <ClientOnly>
      <div v-if="currentUser" class="flex flex-col md:flex-row items-center gap-1 px-2 md:px-3 py-2.5 border-t border-line">
        <UserMenu :user="currentUser" class="md:flex-1 min-w-0" />
        <NotificationBell />
      </div>
    </ClientOnly>
  </aside>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue';
import UserMenu from './UserMenu.vue';
import NotificationBell from './NotificationBell.vue';
import { spacesApi } from '../api/spaces';
import { todosApi } from '../api/todos';
import { normalizeSpaceKey, spaceKeyError } from '../utils/spaceKey';
import { boardPath, timelinePath, backlogPath, issuesPath, releasesPath, reportsPath, settingsPath } from '../utils/spaceRoutes';
import { spaceColor, readableTextOn } from '../utils/spaceColors';
import { useCurrentUser } from '../composables/useCurrentUser';
import { useSavedFilters } from '../composables/useSavedFilters';
import { filterLink, STAR_ICON } from '../utils/filters';

const props = defineProps({
  spaces: {
    type: Array,
    required: true
  },
  // 各スペースの課題数を出すための課題。null ならサイドバーで読む
  todos: {
    type: Array,
    default: null
  },
  // 表示中のスペース。null の場合は「すべてのスペース」
  currentSpace: {
    type: Object,
    default: null
  }
});

// select: スペースを選んだ（null は「すべて」）、create・delete: スペースの作成・削除、move: 課題やエピックをスペースへドロップした
const emit = defineEmits(['select', 'create', 'delete', 'move']);

const route = useRoute();
const { user: currentUser } = useCurrentUser();
// 課題の作成モーダルの開閉状態。GlobalCreate と共有する
const createIssueOpen = useState('createIssueOpen', () => false);
// 課題を作成したら増える。課題数を読み直すのに使う
const issuesVersion = useState('issuesVersion', () => 0);

// スペースの切り替え
const switcherRoot = ref(null);
const switcherOpen = ref(false);
const switcherOptions = computed(() => [
  { id: null, label: 'すべてのスペース', key: '', color: 'var(--kb-accent)' },
  ...props.spaces.map(space => ({ id: space.ID, label: space.Title, key: space.Key, color: spaceColor(space) }))
]);
const selectSpace = (spaceId) => {
  switcherOpen.value = false;
  emit('select', spaceId);
};

// 検索（GlobalSearch）を開く
const searchOpen = useState('globalSearchOpen', () => false);
const openSearch = () => {
  searchOpen.value = true;
};
const handleKeydown = (event) => {
  if (event.key === 'Escape') switcherOpen.value = false;
};
const closeOnOutsideClick = (event) => {
  if (switcherOpen.value && switcherRoot.value && !switcherRoot.value.contains(event.target)) {
    switcherOpen.value = false;
  }
};
onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
  document.addEventListener('click', closeOnOutsideClick);
});
onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleKeydown);
  document.removeEventListener('click', closeOnOutsideClick);
});

const SETTINGS_ICON = [
  'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z',
  'M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.01a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.01a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z'
];

// ナビ。ダッシュボードはスペースによらず 1 つ。タイムライン・バックログ・リリース・設定はスペースごと（Jira のプロジェクトと同じ）。課題一覧はスペースの中から開くと、そのスペースに絞り込む
const navItems = computed(() => {
  const space = props.currentSpace;
  const list = [
    { label: 'ダッシュボード', to: '/dashboard', icon: ['M3 3h8v10H3z', 'M13 3h8v6h-8z', 'M13 11h8v10h-8z', 'M3 15h8v6H3z'] },
    { label: 'ボード', to: boardPath(space), icon: ['M3 4h18v16H3z', 'M9 4v16', 'M15 4v16'] }
  ];
  if (space) {
    list.push({ label: 'タイムライン', to: timelinePath(space), icon: ['M3 6h10', 'M7 12h12', 'M5 18h8', 'M3 3v18'] });
    list.push({ label: 'バックログ', to: backlogPath(space), icon: ['M8 6h13', 'M8 12h13', 'M8 18h13', 'M3 6h.01', 'M3 12h.01', 'M3 18h.01'] });
  }
  list.push({ label: '課題一覧', to: issuesPath(space), filters: true, icon: ['M9 11l3 3L22 4', 'M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11'] });
  if (space) {
    list.push({ label: 'レポート', to: reportsPath(space), icon: ['M3 3v18h18', 'M7 14l4-4 3 3 5-6'] });
    list.push({ label: 'リリース', to: releasesPath(space), icon: ['M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z', 'M7 7h.01'] });
    list.push({ label: '設定', to: settingsPath(space), icon: SETTINGS_ICON });
  }
  return list;
});

// お気に入りのフィルター。ログインしてから読む
const { filters: savedFilters, reload: reloadFilters } = useSavedFilters();
const favouriteFilters = computed(() => savedFilters.value.filter(filter => filter.Favourite));
onMounted(reloadFilters);
const isFilterActive = (filter) => route.path === '/issues' && route.query.filter === String(filter.ID);

// リリースはバージョンの詳細（/releases/:id）でも選択中にする。課題一覧は絞り込みのクエリを除いたパスで比べる
const isActive = (to) => {
  const path = to.split('?')[0];
  return route.path === path || (path.endsWith('/releases') && route.path.startsWith(`${path}/`));
};

// 各スペースの課題数。ページから課題を受け取らなければ、ここで読む
const loadedTodos = ref([]);
const fetchTodos = async () => {
  if (props.todos) return;
  try {
    loadedTodos.value = await todosApi.fetchAll();
  } catch (err) {
    console.error('Failed to fetch todos for sidebar:', err);
  }
};
onMounted(fetchTodos);
watch(issuesVersion, fetchTodos);
const issueCount = (spaceId) => (props.todos || loadedTodos.value).filter(todo => todo.SpaceID === spaceId).length;

// スペースの作成
const creating = ref(false);
const newTitle = ref('');
const newKey = ref('');
// キーを手で入力したら、名前からの候補で上書きしない（空にすると候補に戻る）
const keyEdited = ref(false);
const keyError = computed(() => spaceKeyError(newKey.value, props.spaces));
const canCreate = computed(() => newTitle.value.trim() !== '' && !keyError.value);

// 名前の入力が止まったら、サーバーからキーの候補を取得する（古い応答は捨てる）
let suggestTimer = null;
let suggestRequest = 0;
const requestKeySuggestion = (title) => {
  clearTimeout(suggestTimer);
  if (keyEdited.value) return;
  const trimmed = title.trim();
  if (!trimmed) {
    newKey.value = '';
    return;
  }
  suggestTimer = setTimeout(async () => {
    const request = ++suggestRequest;
    try {
      const key = await spacesApi.suggestKey(trimmed);
      if (request === suggestRequest && !keyEdited.value) {
        newKey.value = key;
      }
    } catch (err) {
      console.error('Failed to suggest space key:', err);
    }
  }, 300);
};
watch(newTitle, requestKeySuggestion);

const handleKeyInput = (event) => {
  newKey.value = normalizeSpaceKey(event.target.value);
  event.target.value = newKey.value;
  keyEdited.value = newKey.value !== '';
  if (!keyEdited.value) {
    requestKeySuggestion(newTitle.value);
  }
};

const handleCreate = () => {
  if (!canCreate.value) return;
  emit('create', { title: newTitle.value.trim(), key: newKey.value });
  newTitle.value = '';
  newKey.value = '';
  keyEdited.value = false;
  creating.value = false;
};

// 課題カード（taskId）かエピック（epicId）のドラッグだけを受け付ける
const dragOverSpaceId = ref(null);
const isMovableDrag = (event) => {
  const types = Array.from(event.dataTransfer.types);
  return types.includes('taskid') || types.includes('epicid');
};
const handleDragOver = (event, spaceId) => {
  if (!isMovableDrag(event)) return;
  event.preventDefault();
  event.dataTransfer.dropEffect = 'move';
  dragOverSpaceId.value = spaceId;
};
const handleDragLeave = (spaceId) => {
  if (dragOverSpaceId.value === spaceId) {
    dragOverSpaceId.value = null;
  }
};
const handleDrop = (event, spaceId) => {
  event.preventDefault();
  dragOverSpaceId.value = null;
  const taskId = parseInt(event.dataTransfer.getData('taskId'));
  if (!Number.isNaN(taskId)) {
    emit('move', { type: 'task', id: taskId, spaceId });
    return;
  }
  const epicId = parseInt(event.dataTransfer.getData('epicId'));
  if (!Number.isNaN(epicId)) {
    emit('move', { type: 'epic', id: epicId, spaceId });
  }
};
</script>
