<template>
  <!-- 画面全体のキーボードショートカット（Jira と同じキー）と、`?` で開く一覧。
       課題を開いているときのショートカット（e・a・i・m）は IssueDetail で扱う -->
  <div
    v-if="helpOpen"
    class="fixed inset-0 bg-overlay flex items-center justify-center z-50 px-4"
    @click.self="helpOpen = false"
  >
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="shortcut-help-title"
      class="w-full max-w-lg bg-surface border border-line rounded-xl shadow-2xl px-6 py-5 text-fg"
    >
      <div class="flex items-center">
        <h2 id="shortcut-help-title" class="m-0 flex-1 text-lg font-bold">キーボードショートカット</h2>
        <button
          ref="closeButton"
          type="button"
          class="w-8 h-8 flex items-center justify-center p-0 border-0 rounded-md bg-transparent text-muted cursor-pointer hover:bg-raised hover:text-fg"
          aria-label="閉じる"
          @click="helpOpen = false"
        >
          ×
        </button>
      </div>
      <section v-for="group in SHORTCUT_GROUPS" :key="group.title" class="mt-4">
        <h3 class="m-0 mb-2 text-xs font-semibold text-muted">{{ group.title }}</h3>
        <dl class="m-0 flex flex-col gap-1.5">
          <div v-for="item in group.items" :key="item.label" class="flex items-center gap-3 text-sm">
            <dt class="w-24 shrink-0 flex items-center gap-1">
              <template v-for="(key, i) in item.keys" :key="i">
                <span v-if="i > 0" class="text-xs text-muted">{{ item.sequence ? 'の後に' : '+' }}</span>
                <kbd class="min-w-6 px-1.5 py-0.5 rounded border border-line bg-raised font-sans text-xs text-center">{{ key }}</kbd>
              </template>
            </dt>
            <dd class="m-0 text-fg">{{ item.label }}</dd>
          </div>
        </dl>
      </section>
      <p class="mt-5 mb-0 text-xs text-muted">入力欄にカーソルがあるときは、ショートカットは使えません。</p>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick, onMounted, onBeforeUnmount } from 'vue';
import { SHORTCUT_GROUPS, shouldIgnoreShortcut } from '../utils/shortcuts';
import { boardPath, backlogPath } from '../utils/spaceRoutes';

const searchOpen = useState('globalSearchOpen', () => false);
const createIssueOpen = useState('createIssueOpen', () => false);
// 表示中のスペース（SpaceShell が設定する。null は「すべてのスペース」）
const currentSpace = useState('currentSpace', () => null);

const helpOpen = ref(false);
const closeButton = ref(null);
watch(helpOpen, async (isOpen) => {
  if (!isOpen) return;
  await nextTick();
  closeButton.value?.focus();
});

// `g` を押してから次のキーを待つ時間
const SEQUENCE_TIMEOUT_MS = 1500;
let pendingG = false;
let pendingTimer = null;
const clearPending = () => {
  pendingG = false;
  clearTimeout(pendingTimer);
};

const handleKeydown = (event) => {
  // ⌘K / Ctrl+K は入力中でも検索を開く（ほかのダイアログを開いているときは除く）
  if ((event.metaKey || event.ctrlKey) && !event.altKey && event.key.toLowerCase() === 'k') {
    if (helpOpen.value || !document.querySelector('[aria-modal="true"]')) {
      event.preventDefault();
      helpOpen.value = false;
      searchOpen.value = true;
    }
    return;
  }
  // 一覧はもう一度 `?` か Esc で閉じる
  if (helpOpen.value) {
    if (event.key === 'Escape' || event.key === '?') {
      event.preventDefault();
      helpOpen.value = false;
    }
    return;
  }
  if (createIssueOpen.value || shouldIgnoreShortcut(event)) {
    clearPending();
    return;
  }

  if (pendingG) {
    clearPending();
    if (event.key === 'b') {
      event.preventDefault();
      navigateTo(boardPath(currentSpace.value));
    } else if (event.key === 'k' && currentSpace.value) {
      // バックログはスペースごと。「すべてのスペース」では開かない
      event.preventDefault();
      navigateTo(backlogPath(currentSpace.value));
    }
    return;
  }

  switch (event.key) {
    case '/':
      event.preventDefault();
      searchOpen.value = true;
      break;
    case 'c':
      event.preventDefault();
      createIssueOpen.value = true;
      break;
    case 'g':
      pendingG = true;
      pendingTimer = setTimeout(clearPending, SEQUENCE_TIMEOUT_MS);
      break;
    case '?':
      event.preventDefault();
      helpOpen.value = true;
      break;
  }
};

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});
onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleKeydown);
  clearPending();
});
</script>
