<template>
  <!-- バージョンの進捗（Jira のリリースの進捗バー）。完了は緑、進行中は青、未着手は灰 -->
  <div class="min-w-0">
    <div
      class="flex h-2 w-full overflow-hidden rounded-full bg-line"
      role="img"
      :aria-label="summary"
      :title="summary"
    >
      <div class="bg-success" :style="{ width: percent(progress.done) }"></div>
      <div class="bg-accent" :style="{ width: percent(progress.inProgress) }"></div>
    </div>
    <p v-if="showCounts" class="m-0 mt-1 text-xs text-muted">{{ summary }}</p>
  </div>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  // versionProgress の結果（done・inProgress・toDo・total）
  progress: {
    type: Object,
    required: true
  },
  // バーの下に件数を出すか
  showCounts: {
    type: Boolean,
    default: false
  }
});

const percent = (count) => (props.progress.total ? `${(count / props.progress.total) * 100}%` : '0%');

const summary = computed(() => {
  const { done, inProgress, toDo, total } = props.progress;
  return total ? `完了 ${done} / 進行中 ${inProgress} / 未着手 ${toDo}（全 ${total} 件）` : '課題なし';
});
</script>
