<template>
  <!-- ページ上部のツールバー（1 行）。左にパンくずと小さな情報（スプリントの期間など）、右にページの操作と「+ 新規課題」 -->
  <header class="sticky top-0 z-30 flex flex-wrap items-center gap-x-3 gap-y-2 min-h-12 px-4 md:px-6 py-2 bg-canvas/90 backdrop-blur border-b border-line">
    <nav class="min-w-0 flex items-center gap-1.5 text-sm" aria-label="パンくず">
      <template v-for="(crumb, index) in crumbs" :key="index">
        <span v-if="index > 0" class="text-muted" aria-hidden="true">›</span>
        <NuxtLink
          v-if="crumb.to && index < crumbs.length - 1"
          :to="crumb.to"
          class="shrink-0 font-medium text-muted no-underline hover:text-fg"
        >
          {{ crumb.label }}
        </NuxtLink>
        <span
          v-else
          :class="index === crumbs.length - 1 ? 'min-w-0 truncate font-semibold text-fg' : 'shrink-0 font-medium text-muted'"
          :aria-current="index === crumbs.length - 1 ? 'page' : undefined"
        >
          {{ crumb.label }}
        </span>
      </template>
    </nav>
    <slot name="chips" />
    <div class="ml-auto flex flex-wrap items-center gap-2">
      <slot />
      <button
        type="button"
        class="inline-flex items-center gap-1 px-2.5 py-1 border-0 rounded-md bg-accent text-on-accent text-xs font-semibold cursor-pointer whitespace-nowrap hover:bg-accent-hover"
        @click="createIssueOpen = true"
      >
        + 新規課題
      </button>
    </div>
  </header>
</template>

<script setup>
defineProps({
  // パンくず。{ label, to? } の配列で、最後が今のページ
  crumbs: {
    type: Array,
    required: true
  }
});

// 課題の作成モーダルの開閉状態。GlobalCreate と共有する
const createIssueOpen = useState('createIssueOpen', () => false);
</script>
