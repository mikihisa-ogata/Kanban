<template>
  <!-- Jira のアバターと同じく、表示名の頭文字をユーザーの色の丸に入れる -->
  <span
    :class="['shrink-0 inline-flex items-center justify-center rounded-full font-semibold select-none', sizeClass]"
    :style="{ backgroundColor: color, color: readableTextOn(color) }"
    :title="user.DisplayName"
    aria-hidden="true"
  >
    {{ initial }}
  </span>
</template>

<script setup>
import { computed } from 'vue';
import { readableTextOn } from '../utils/spaceColors';

const props = defineProps({
  // { DisplayName, AvatarColor }
  user: {
    type: Object,
    required: true
  },
  size: {
    type: String,
    default: 'md',
    validator: (value) => ['xs', 'sm', 'md', 'lg'].includes(value)
  }
});

// 文字の色は、背景のユーザーの色に合わせて読みやすい方（白か濃い色）にする
const color = computed(() => props.user.AvatarColor || '#6B778C');

const initial = computed(() => Array.from(props.user.DisplayName || '?')[0].toUpperCase());

const sizeClass = computed(() => ({
  xs: 'w-5 h-5 text-[10px]',
  sm: 'w-6 h-6 text-xs',
  md: 'w-8 h-8 text-sm',
  lg: 'w-10 h-10 text-base'
})[props.size]);
</script>
