<template>
  <!-- Jira のアバターと同じく、表示名の頭文字をユーザーの色の丸に入れる -->
  <span
    :class="['shrink-0 inline-flex items-center justify-center rounded-full font-semibold text-white select-none', sizeClass]"
    :style="{ backgroundColor: user.AvatarColor || '#6B778C' }"
    :title="user.DisplayName"
    aria-hidden="true"
  >
    {{ initial }}
  </span>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  // { DisplayName, AvatarColor }
  user: {
    type: Object,
    required: true
  },
  size: {
    type: String,
    default: 'md',
    validator: (value) => ['sm', 'md', 'lg'].includes(value)
  }
});

const initial = computed(() => Array.from(props.user.DisplayName || '?')[0].toUpperCase());

const sizeClass = computed(() => ({
  sm: 'w-6 h-6 text-xs',
  md: 'w-8 h-8 text-sm',
  lg: 'w-10 h-10 text-base'
})[props.size]);
</script>
