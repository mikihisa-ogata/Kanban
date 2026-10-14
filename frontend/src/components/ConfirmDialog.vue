<template>
  <div
    class="fixed inset-0 bg-overlay flex items-center justify-center z-50"
    @click.self="$emit('cancel')"
  >
    <div
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
      aria-describedby="confirm-dialog-message"
      class="bg-surface rounded-2xl px-8 pt-8 pb-6 shadow-2xl mx-4 max-w-md w-full"
    >
      <!-- アイコンとタイトルを横に並べ、本文はその下に全幅で置く -->
      <div class="flex items-center gap-4">
        <div class="shrink-0 w-10 h-10 rounded-full bg-danger/10 text-danger flex items-center justify-center text-xl font-bold">
          !
        </div>
        <h3 id="confirm-dialog-title" class="m-0 min-w-0 text-lg font-bold text-fg">{{ title }}</h3>
      </div>
      <p id="confirm-dialog-message" class="mt-4 text-sm text-muted whitespace-pre-line break-words">{{ message }}</p>

      <div class="flex justify-center gap-3 mt-6">
        <button
          ref="cancelButton"
          type="button"
          class="py-2 px-5 bg-surface text-fg border-2 border-line rounded-lg text-sm font-semibold hover:bg-raised focus:outline-none focus-visible:ring-2 focus-visible:ring-line"
          @click="$emit('cancel')"
        >
          キャンセル
        </button>
        <button
          type="button"
          class="py-2 px-5 bg-danger text-on-danger border-2 border-danger rounded-lg text-sm font-semibold hover:bg-danger/90 hover:border-danger focus:outline-none focus-visible:ring-2 focus-visible:ring-danger/40"
          @click="$emit('confirm')"
        >
          {{ confirmLabel }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue';

defineProps({
  title: {
    type: String,
    required: true
  },
  message: {
    type: String,
    default: ''
  },
  confirmLabel: {
    type: String,
    default: '削除'
  }
});

const emit = defineEmits(['confirm', 'cancel']);

// Esc キーでキャンセルする
const handleKeydown = (event) => {
  if (event.key === 'Escape') {
    emit('cancel');
  }
};

// 誤って削除しないよう、開いたときはキャンセルにフォーカスを置く
const cancelButton = ref(null);
onMounted(() => {
  cancelButton.value?.focus();
  window.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown);
});
</script>
