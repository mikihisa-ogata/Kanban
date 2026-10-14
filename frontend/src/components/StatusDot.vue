<template>
  <!-- ステータスのカテゴリを表す小さな丸。未着手は灰の輪、進行中は黄の輪、完了は塗りつぶしたアクセント色 -->
  <span
    :class="[
      'shrink-0 inline-block w-3 h-3 rounded-full',
      category === 'Done' ? 'bg-accent' : category === 'InProgress' ? 'border-2 border-warning' : 'border-2 border-muted'
    ]"
    role="img"
    :aria-label="statusCategoryLabel(category)"
  />
</template>

<script setup>
import { statusCategoryLabel } from '../utils/statuses';

defineProps({
  // ToDo・InProgress・Done
  category: {
    type: String,
    required: true
  }
});
</script>
