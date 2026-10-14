<template>
  <!-- ページの共通の枠。左にサイドバー（スペースの切り替え・ナビ・スペース一覧・ユーザー）、右にページ本体 -->
  <div class="flex min-h-screen">
    <div class="sticky top-0 self-start shrink-0 z-40">
      <AppSidebar
        :spaces="spaces"
        :todos="todos"
        :current-space="currentSpace"
        @select="handleSelect"
        @create="handleCreateSpace"
        @delete="handleDeleteSpace"
        @move="$emit('move', $event)"
      />
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

    <div v-if="error" class="fixed bottom-8 right-8 bg-danger/10 text-danger px-6 py-4 rounded-xl shadow-xl max-w-96 text-sm font-semibold border-l-4 border-danger animate-slideIn">
      {{ error }}
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import AppSidebar from './AppSidebar.vue';
import ConfirmDialog from './ConfirmDialog.vue';
import { spacesApi } from '../api/spaces';
import { boardPath } from '../utils/spaceRoutes';

const props = defineProps({
  spaces: {
    type: Array,
    required: true
  },
  // サイドバーに各スペースの課題数を出すための課題。null ならサイドバーで読む
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
