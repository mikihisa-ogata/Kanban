<template>
  <!-- 優先度のアイコン。Jira と同じく、高いほど赤の上向き矢印、低いほど青の下向き矢印。中はオレンジの二本線 -->
  <svg
    class="shrink-0 block"
    :width="size"
    :height="size"
    viewBox="0 0 16 16"
    role="img"
    :aria-label="label"
  >
    <title>{{ label }}</title>
    <g
      fill="none"
      :stroke="colors[kind]"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path v-for="(d, i) in paths[kind]" :key="i" :d="d" />
    </g>
  </svg>
</template>

<script setup>
import { computed } from 'vue';
import { priorityLabel } from '../utils/priorities';

const props = defineProps({
  // Highest / High / Medium / Low / Lowest。それ以外（未設定など）は中として表示する
  priority: {
    type: String,
    default: 'Medium'
  },
  size: {
    type: Number,
    default: 16
  }
});

const colors = {
  Highest: '#CD1316',
  High: '#E9494A',
  Medium: '#E97F33',
  Low: '#2684FF',
  Lowest: '#0065FF'
};

const paths = {
  Highest: ['M3.5 8l4.5-4 4.5 4', 'M3.5 12.5l4.5-4 4.5 4'],
  High: ['M3.5 10.5l4.5-4.5 4.5 4.5'],
  Medium: ['M3 6h10', 'M3 10h10'],
  Low: ['M3.5 5.5l4.5 4.5 4.5-4.5'],
  Lowest: ['M3.5 3.5l4.5 4 4.5-4', 'M3.5 8l4.5 4 4.5-4']
};

const kind = computed(() => (colors[props.priority] ? props.priority : 'Medium'));
const label = computed(() => `優先度: ${priorityLabel(kind.value)}`);
</script>
