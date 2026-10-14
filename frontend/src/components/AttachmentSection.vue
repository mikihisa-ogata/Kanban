<template>
  <!-- 添付ファイルのセクション。課題の詳細（ページとモーダル）で使う。
       ファイル選択とドラッグ＆ドロップで添付でき、画像はサムネイルを出してクリックで拡大する -->
  <section
    @dragenter="handleDragEnter"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <div class="flex items-center gap-2 mb-2">
      <h2 class="m-0 text-sm font-semibold text-fg">添付ファイル</h2>
      <span v-if="attachments.length" class="text-xs text-muted">{{ attachments.length }} 件</span>
      <button
        type="button"
        class="ml-auto bg-transparent border-0 p-0 w-6 h-6 flex items-center justify-center rounded text-muted text-lg leading-none cursor-pointer hover:bg-raised hover:text-fg disabled:opacity-50"
        title="ファイルを添付"
        :disabled="uploading > 0"
        @click="fileInput?.click()"
      >
        +
      </button>
      <input ref="fileInput" type="file" multiple class="hidden" aria-label="添付するファイル" @change="handleFileSelect" />
    </div>

    <!-- ドロップ先。ファイルをドラッグしている間は強調する -->
    <div
      :class="[
        'rounded-lg border-2 border-dashed px-3 py-3 text-center text-sm transition-colors',
        dragging ? 'border-accent bg-accent/10 text-accent' : 'border-line text-muted'
      ]"
    >
      <template v-if="uploading > 0">アップロード中...（残り {{ uploading }} 件）</template>
      <template v-else>
        ファイルをここにドロップ、または
        <button
          type="button"
          class="bg-transparent border-0 p-0 font-semibold text-accent cursor-pointer hover:underline"
          @click="fileInput?.click()"
        >
          選択
        </button>
        （1 ファイル 10MB まで）
      </template>
    </div>
    <p v-for="message in errors" :key="message" class="mt-1 mb-0 text-xs text-danger">{{ message }}</p>

    <ul v-if="attachments.length" class="mt-3 mb-0 p-0 list-none grid grid-cols-[repeat(auto-fill,minmax(9rem,1fr))] gap-3">
      <li
        v-for="attachment in attachments"
        :key="attachment.ID"
        class="min-w-0 border border-line rounded-lg overflow-hidden bg-surface"
      >
        <!-- 画像はサムネイル（押すと拡大）、それ以外はファイルのアイコン -->
        <button
          v-if="isImage(attachment)"
          type="button"
          class="block w-full h-24 p-0 border-0 bg-raised cursor-zoom-in"
          :title="`${attachment.FileName} を拡大`"
          @click="previewing = attachment"
        >
          <img :src="attachmentsApi.fileUrl(attachment.ID, { inline: true })" :alt="attachment.FileName" class="block w-full h-full object-cover" loading="lazy" />
        </button>
        <a
          v-else
          :href="attachmentsApi.fileUrl(attachment.ID)"
          class="flex w-full h-24 items-center justify-center bg-raised text-muted no-underline hover:text-muted"
          :title="`${attachment.FileName} をダウンロード`"
        >
          <svg class="w-8 h-8 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
            <path d="M14 2v6h6" />
          </svg>
        </a>
        <div class="px-2 py-1.5">
          <p class="m-0 truncate text-xs font-semibold text-fg" :title="attachment.FileName">{{ attachment.FileName }}</p>
          <div class="flex items-center gap-1 text-xs text-muted">
            <span :title="formatDateTime(attachment.CreatedAt)">{{ formatSize(attachment.Size) }}</span>
            <a
              :href="attachmentsApi.fileUrl(attachment.ID)"
              class="ml-auto p-0.5 rounded text-muted hover:bg-raised hover:text-fg"
              :title="`${attachment.FileName} をダウンロード`"
              :aria-label="`${attachment.FileName} をダウンロード`"
            >
              <svg class="w-4 h-4 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <path d="M7 10l5 5 5-5" />
                <path d="M12 15V3" />
              </svg>
            </a>
            <button
              type="button"
              class="bg-transparent border-0 p-0.5 rounded text-muted cursor-pointer hover:bg-danger/10 hover:text-danger"
              :title="`${attachment.FileName} を削除`"
              :aria-label="`${attachment.FileName} を削除`"
              @click="confirmingDelete = attachment"
            >
              <svg class="w-4 h-4 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M3 6h18" />
                <path d="M8 6V4h8v2" />
                <path d="M19 6l-1 14H6L5 6" />
              </svg>
            </button>
          </div>
        </div>
      </li>
    </ul>

    <!-- 画像の拡大表示。背景・× ・Esc で閉じる -->
    <div
      v-if="previewing"
      class="fixed inset-0 z-[60] bg-black/85 flex flex-col items-center justify-center p-6"
      role="dialog"
      aria-modal="true"
      :aria-label="previewing.FileName"
      @click.self="previewing = null"
    >
      <div class="w-full max-w-5xl flex items-center gap-3 mb-3 text-white">
        <p class="m-0 flex-1 min-w-0 truncate text-sm font-semibold">{{ previewing.FileName }}</p>
        <a :href="attachmentsApi.fileUrl(previewing.ID)" class="text-sm font-semibold text-white hover:underline">ダウンロード</a>
        <button
          type="button"
          class="bg-transparent border-0 text-white text-2xl leading-none cursor-pointer p-1 rounded hover:bg-white/10"
          title="閉じる"
          @click="previewing = null"
        >
          ×
        </button>
      </div>
      <img
        :src="attachmentsApi.fileUrl(previewing.ID, { inline: true })"
        :alt="previewing.FileName"
        class="max-w-full max-h-[80vh] object-contain rounded shadow-2xl bg-surface"
        @click.stop
      />
    </div>

    <ConfirmDialog
      v-if="confirmingDelete"
      title="添付ファイルを削除しますか？"
      :message="`「${confirmingDelete.FileName}」を削除します。この操作は取り消せません。`"
      @confirm="deleteAttachment"
      @cancel="confirmingDelete = null"
    />
  </section>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted } from 'vue';
import ConfirmDialog from './ConfirmDialog.vue';
import { attachmentsApi, ATTACHMENT_MAX_SIZE } from '../api/attachments';
import { formatDateTime } from '../utils/datetime';

const props = defineProps({
  // ファイルを添付する課題
  task: {
    type: Object,
    required: true
  }
});

// ブラウザでそのまま表示する画像の種類（bff が inline で返すものと同じ。SVG は含めない）
const INLINE_IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/gif', 'image/webp', 'image/bmp'];
const isImage = (attachment) => INLINE_IMAGE_TYPES.includes(attachment.ContentType);

const formatSize = (bytes) => {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
};

const attachments = ref([]);
// 失敗したファイルごとのメッセージ
const errors = ref([]);
const uploading = ref(0);
const fileInput = ref(null);
const previewing = ref(null);
const confirmingDelete = ref(null);

const fetchAttachments = async () => {
  const todoId = props.task.ID;
  try {
    const result = await attachmentsApi.fetchByTodo(todoId);
    // 読み込み中に別の課題へ切り替わった場合は捨てる
    if (todoId === props.task.ID) {
      attachments.value = result;
    }
  } catch (err) {
    errors.value = ['添付ファイルの取得に失敗しました: ' + err.message];
    console.error('Failed to fetch attachments:', err);
  }
};

// 選んだ・ドロップしたファイルを 1 つずつアップロードする。上限を超えるファイルは送らずにエラーにする
const uploadFiles = async (files) => {
  const todoId = props.task.ID;
  errors.value = [];
  const targets = [];
  for (const file of files) {
    if (file.size > ATTACHMENT_MAX_SIZE) {
      errors.value.push(`「${file.name}」は 10MB を超えているため添付できません`);
    } else if (file.size === 0) {
      errors.value.push(`「${file.name}」は空のファイルのため添付できません`);
    } else {
      targets.push(file);
    }
  }
  uploading.value += targets.length;
  for (const file of targets) {
    try {
      const result = await attachmentsApi.upload(todoId, file);
      if (todoId === props.task.ID) {
        attachments.value = [...attachments.value, result.attachment];
      }
    } catch (err) {
      errors.value.push(`「${file.name}」の添付に失敗しました: ${err.message}`);
      console.error('Failed to upload attachment:', err);
    } finally {
      uploading.value -= 1;
    }
  }
};

const handleFileSelect = async (event) => {
  const files = [...event.target.files];
  // 同じファイルをもう一度選べるよう、選択をクリアする
  event.target.value = '';
  await uploadFiles(files);
};

// ファイルのドラッグだけを受け付ける（カードのドラッグなどは無視する）
const dragging = ref(false);
let dragDepth = 0;
const hasFiles = (event) => [...(event.dataTransfer?.types || [])].includes('Files');

const handleDragEnter = (event) => {
  if (!hasFiles(event)) return;
  event.preventDefault();
  dragDepth += 1;
  dragging.value = true;
};

const handleDragOver = (event) => {
  if (!hasFiles(event)) return;
  event.preventDefault();
  event.dataTransfer.dropEffect = 'copy';
};

const handleDragLeave = (event) => {
  if (!hasFiles(event)) return;
  dragDepth = Math.max(0, dragDepth - 1);
  if (dragDepth === 0) dragging.value = false;
};

const handleDrop = async (event) => {
  if (!hasFiles(event)) return;
  event.preventDefault();
  dragDepth = 0;
  dragging.value = false;
  await uploadFiles([...event.dataTransfer.files]);
};

const deleteAttachment = async () => {
  const attachment = confirmingDelete.value;
  confirmingDelete.value = null;
  try {
    errors.value = [];
    await attachmentsApi.delete(attachment.ID);
    attachments.value = attachments.value.filter(a => a.ID !== attachment.ID);
  } catch (err) {
    errors.value = ['添付ファイルの削除に失敗しました: ' + err.message];
    console.error('Failed to delete attachment:', err);
    await fetchAttachments();
  }
};

// 拡大表示は Esc で閉じる
const handleKeydown = (event) => {
  if (event.key === 'Escape' && previewing.value) {
    previewing.value = null;
  }
};

onMounted(() => {
  window.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown);
});

// 別の課題に切り替わったら読み直す
watch(() => props.task.ID, async () => {
  attachments.value = [];
  errors.value = [];
  previewing.value = null;
  confirmingDelete.value = null;
  await fetchAttachments();
}, { immediate: true });
</script>
