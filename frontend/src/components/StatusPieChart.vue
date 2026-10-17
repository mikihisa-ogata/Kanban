<template>
  <!-- ステータス別の円グラフ（ドーナツ）。色はカテゴリ（To Do は灰色・進行中はアクセント・完了は緑）で、同じカテゴリの
       2 つ目以降のステータスは薄くする。扇形か凡例にカーソルを合わせると、ほかの扇形を薄くして中央にそのステータスの件数と割合を出す -->
  <figure class="m-0 flex flex-wrap items-center justify-center gap-4">
    <svg :viewBox="`0 0 ${SIZE} ${SIZE}`" class="block w-40 h-40 shrink-0" role="img" :aria-label="ariaLabel">
      <template v-for="(slice, index) in slices" :key="index">
        <circle
          v-if="slice.full"
          :cx="CENTER" :cy="CENTER" :r="(OUTER + INNER) / 2"
          fill="none"
          :stroke="slice.color" :stroke-opacity="shownOpacity(slice, index)" :stroke-width="OUTER - INNER"
          @mouseenter="hovered = index" @mouseleave="hovered = null"
        >
          <title>{{ sliceTitle(slice) }}</title>
        </circle>
        <path
          v-else
          :d="slice.path"
          :fill="slice.color" :fill-opacity="shownOpacity(slice, index)"
          stroke="var(--kb-surface)" stroke-width="2" stroke-linejoin="round"
          class="transition-[fill-opacity] duration-150"
          @mouseenter="hovered = index" @mouseleave="hovered = null"
        >
          <title>{{ sliceTitle(slice) }}</title>
        </path>
      </template>
      <text :x="CENTER" :y="CENTER - 4" text-anchor="middle" class="fill-fg text-[20px] font-semibold">{{ centerValue }}</text>
      <text :x="CENTER" :y="CENTER + 14" text-anchor="middle" class="fill-muted text-[10px]">{{ centerLabel }}</text>
    </svg>
    <figcaption class="min-w-48 flex-1">
      <table class="w-full text-sm border-collapse">
        <caption class="sr-only">ステータスごとの課題数</caption>
        <tbody>
          <tr
            v-for="(slice, index) in slices"
            :key="index"
            :class="['cursor-default', hovered === index ? 'bg-raised' : '']"
            @mouseenter="hovered = index" @mouseleave="hovered = null"
          >
            <td class="py-1 pl-1 pr-2 w-4">
              <span class="block w-3 h-3 rounded-sm" :style="{ backgroundColor: slice.color, opacity: slice.opacity }" aria-hidden="true" />
            </td>
            <th scope="row" class="py-1 pr-2 text-left font-normal text-fg max-w-0 w-full truncate" :title="`${slice.name}（${statusCategoryLabel(slice.category)}）`">{{ slice.name }}</th>
            <td class="py-1 pr-2 text-right tabular-nums text-fg">{{ slice.count }}</td>
            <td class="py-1 pr-1 text-right tabular-nums text-muted">{{ slice.percent }}%</td>
          </tr>
        </tbody>
      </table>
    </figcaption>
  </figure>
</template>

<script setup>
import { ref, computed } from 'vue';
import { pieSlices } from '../utils/dashboard';
import { statusCategoryLabel } from '../utils/statuses';

const props = defineProps({
  // GET /dashboard/status-counts の結果（1 件以上）
  counts: {
    type: Array,
    required: true
  }
});

const SIZE = 160;
const CENTER = SIZE / 2;
const OUTER = 76;
const INNER = 50;

const hovered = ref(null);
const slices = computed(() => pieSlices(props.counts, { cx: CENTER, cy: CENTER, outer: OUTER, inner: INNER }));
const total = computed(() => props.counts.reduce((sum, c) => sum + c.Count, 0));

const centerValue = computed(() => (hovered.value === null ? total.value : slices.value[hovered.value].count));
const centerLabel = computed(() => {
  if (hovered.value === null) return '課題';
  const slice = slices.value[hovered.value];
  const name = slice.name.length > 8 ? `${slice.name.slice(0, 7)}…` : slice.name;
  return `${name} ${slice.percent}%`;
});

// カーソルを合わせている間は、ほかの扇形を薄くする
const shownOpacity = (slice, index) => (hovered.value === null || hovered.value === index ? slice.opacity : slice.opacity * 0.35);

const sliceTitle = (slice) => `${slice.name}（${statusCategoryLabel(slice.category)}）: ${slice.count} 件（${slice.percent}%）`;
const ariaLabel = computed(() => `ステータス別の円グラフ。全 ${total.value} 件: ` + slices.value.map(s => `${s.name} ${s.count} 件`).join('、'));
</script>
