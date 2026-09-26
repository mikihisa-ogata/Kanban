<template>
  <div
    class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50"
    @click.self="$emit('cancel')"
  >
    <div
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
      aria-describedby="confirm-dialog-message"
      class="bg-white rounded-2xl px-8 pt-8 pb-6 shadow-2xl mx-4 max-w-md w-full"
    >
      <div class="flex items-start gap-4">
        <div class="shrink-0 w-10 h-10 rounded-full bg-red-100 text-red-600 flex items-center justify-center text-xl font-bold">
          !
        </div>
        <div class="min-w-0">
          <h3 id="confirm-dialog-title" class="m-0 text-lg font-bold text-gray-900">{{ title }}</h3>
          <p id="confirm-dialog-message" class="mt-2 text-sm text-gray-600 whitespace-pre-line break-words">{{ message }}</p>
        </div>
      </div>

      <div class="flex justify-end gap-3 mt-6">
        <button
          ref="cancelButton"
          type="button"
          class="py-2 px-5 bg-white text-gray-700 border-2 border-gray-300 rounded-lg text-sm font-semibold hover:bg-gray-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
          @click="$emit('cancel')"
        >
          キャンセル
        </button>
        <button
          type="button"
          class="py-2 px-5 bg-red-500 text-white border-2 border-red-500 rounded-lg text-sm font-semibold hover:bg-red-600 hover:border-red-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-300"
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
