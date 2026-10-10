<template>
  <!-- スペース単位のページの共通の枠。左からスペース一覧（ハンバーガーで開閉）、スペース内の左ナビ、ページ本体 -->
  <div class="flex min-h-screen">
    <!-- 開閉時は幅を広げながらサイドバーを画面左端からスライドさせ、右側の位置もなめらかに動かす -->
    <div
      :class="[
        'sticky top-17 self-start shrink-0 overflow-hidden transition-[width] duration-300 ease-out motion-reduce:transition-none',
        sidebarOpen ? 'w-64' : 'w-0'
      ]"
      :inert="!sidebarOpen"
    >
      <SpaceSidebar
        :class="[
          'transition-transform duration-300 ease-out motion-reduce:transition-none',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        ]"
        :spaces="spaces"
        :todos="todos"
        :selected-space-id="currentSpace ? currentSpace.ID : null"
        @select="handleSelect"
        @create="handleCreateSpace"
        @delete="handleDeleteSpace"
        @move="$emit('move', $event)"
      />
    </div>

    <div class="sticky top-17 self-start shrink-0">
      <SpaceNav :space="currentSpace" />
    </div>

    <div class="flex-1 min-w-0">
      <slot />
    </div>

    <ConfirmDialog
      v-if="confirmDialog"
      :title="confirmDialog.title"
      :message="confirmDialog.message"
      @confirm="closeConfirmDialog(true)"
      @cancel="closeConfirmDialog(false)"
    />

    <div v-if="error" class="fixed bottom-8 right-8 bg-red-100 text-red-800 px-6 py-4 rounded-xl shadow-xl max-w-96 text-sm font-semibold border-l-4 border-red-500 animate-slideIn">
      {{ error }}
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import SpaceSidebar from './SpaceSidebar.vue';
import SpaceNav from './SpaceNav.vue';
import ConfirmDialog from './ConfirmDialog.vue';
import { spacesApi } from '../api/spaces';
import { boardPath } from '../utils/spaceRoutes';

const props = defineProps({
  spaces: {
    type: Array,
    required: true
  },
  // サイドバーに各スペースの進み具合を出すためのタスク。null なら出さない
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

// changed: スペースを作成・削除した（ページ側で一覧などを取り直す）
// move: タスクやエピックをサイドバーのスペースへドロップした
const emit = defineEmits(['changed', 'move']);

// サイドバー（スペース一覧）の開閉状態。ヘッダーのハンバーガーメニューで切り替える
const sidebarOpen = useState('sidebarOpen', () => false);
const error = ref('');
// 削除の確認モーダルの内容と、選んだ結果を返す関数（null の場合は閉じている）
const confirmDialog = ref(null);

const askConfirm = (title, message) => new Promise((resolve) => {
  confirmDialog.value = { title, message, resolve };
});

const closeConfirmDialog = (confirmed) => {
  const dialog = confirmDialog.value;
  if (!dialog) return;
  confirmDialog.value = null;
  dialog.resolve(confirmed);
};

// スペースを切り替えると、そのスペースのボードを開く（null は「すべて」）
const handleSelect = (spaceId) => {
  const space = props.spaces.find(s => s.ID === spaceId) || null;
  navigateTo(boardPath(space));
};

const handleCreateSpace = async ({ title, key }) => {
  try {
    error.value = '';
    await spacesApi.create({ title, key });
    emit('changed');
  } catch (err) {
    error.value = 'スペースの作成に失敗しました: ' + err.message;
    console.error('Failed to create space:', err);
  }
};

// スペースを削除（エピックとタスクは削除されず、スペース未割り当てになる）。
// 表示中のスペースを消した場合は「すべて」のボードへ移る
const handleDeleteSpace = async (spaceId) => {
  const space = props.spaces.find(s => s.ID === spaceId);
  const confirmed = await askConfirm(
    'スペースを削除しますか？',
    `「${space ? space.Title : ''}」を削除します。\nエピックとタスクは削除されず、スペース未割り当てになります。`
  );
  if (!confirmed) {
    return;
  }

  try {
    error.value = '';
    await spacesApi.delete(spaceId);
    if (props.currentSpace && props.currentSpace.ID === spaceId) {
      await navigateTo(boardPath(null));
    }
    emit('changed');
  } catch (err) {
    error.value = 'スペースの削除に失敗しました: ' + err.message;
    console.error('Failed to delete space:', err);
  }
};
</script>
