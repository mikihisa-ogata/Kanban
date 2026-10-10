<template>
  <!-- 課題タイプのアイコン。Jira と同じく、タスクは青のチェック、バグは赤の丸、ストーリーは緑のしおり、サブタスクは水色の重なった四角 -->
  <svg
    class="shrink-0 block"
    :width="size"
    :height="size"
    viewBox="0 0 16 16"
    role="img"
    :aria-label="label"
  >
    <title>{{ label }}</title>
    <rect width="16" height="16" rx="3" :fill="colors[kind]" />
    <path
      v-if="kind === 'Task'"
      d="M4.5 8.2l2.3 2.3 4.7-5"
      fill="none"
      stroke="#fff"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <circle v-else-if="kind === 'Bug'" cx="8" cy="8" r="3.5" fill="#fff" />
    <g v-else-if="kind === 'Subtask'" fill="none" stroke="#fff" stroke-width="1.6">
      <rect x="3.5" y="3.5" width="5.5" height="5.5" rx="0.8" fill="#fff" />
      <rect x="7" y="7" width="5.5" height="5.5" rx="0.8" />
    </g>
    <path v-else d="M5 3.5h6a.5.5 0 0 1 .5.5v8.6a.3.3 0 0 1-.48.24L8 10.6l-3.02 2.24a.3.3 0 0 1-.48-.24V4a.5.5 0 0 1 .5-.5z" fill="#fff" />
  </svg>
</template>

<script setup>
import { computed } from 'vue';
import { issueTypeLabel } from '../utils/issueTypes';

const props = defineProps({
  // Task / Bug / Story / Subtask。それ以外（未設定など）はタスクとして表示する
  type: {
    type: String,
    default: 'Task'
  },
  size: {
    type: Number,
    default: 16
  }
});

const colors = {
  Task: '#4BADE8',
  Bug: '#E5493A',
  Story: '#63BA3C',
  Subtask: '#4BAEE8'
};

const kind = computed(() => (colors[props.type] ? props.type : 'Task'));
const label = computed(() => issueTypeLabel(kind.value));
</script>
