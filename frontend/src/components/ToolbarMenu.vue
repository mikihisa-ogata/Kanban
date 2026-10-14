<template>
  <!-- ツールバーのボタンと、押すと開くパネル（フィルター・表示など）。count があればボタンに件数を出す -->
  <div ref="root" class="relative">
    <button
      type="button"
      :class="toolbarButtonClass(count > 0)"
      aria-haspopup="dialog"
      :aria-expanded="open"
      @click="open = !open"
    >
      <slot name="icon" />
      {{ label }}
      <span v-if="count > 0" class="px-1.5 rounded-full bg-accent text-on-accent text-[10px] leading-4">{{ count }}</span>
    </button>
    <div
      v-if="open"
      :class="['absolute top-full mt-1 p-3 bg-surface border border-line rounded-lg shadow-lg z-40', align === 'right' ? 'right-0' : 'left-0']"
      role="dialog"
      :aria-label="label"
    >
      <slot />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue';
import { toolbarButtonClass } from '../utils/toolbar';

defineProps({
  label: {
    type: String,
    required: true
  },
  // ボタンに出す件数（効いている条件の数など）。0 なら出さない
  count: {
    type: Number,
    default: 0
  },
  // パネルをボタンの左端・右端のどちらに揃えるか
  align: {
    type: String,
    default: 'right'
  }
});

const root = ref(null);
const open = ref(false);

// パネルの外を押すか Esc で閉じる（パネルの中のメニューを押しても閉じない）
const closeOnOutsideClick = (event) => {
  if (open.value && root.value && !root.value.contains(event.target)) {
    open.value = false;
  }
};
const closeOnEscape = (event) => {
  if (event.key === 'Escape') open.value = false;
};
onMounted(() => {
  document.addEventListener('click', closeOnOutsideClick);
  document.addEventListener('keydown', closeOnEscape);
});
onBeforeUnmount(() => {
  document.removeEventListener('click', closeOnOutsideClick);
  document.removeEventListener('keydown', closeOnEscape);
});
</script>
