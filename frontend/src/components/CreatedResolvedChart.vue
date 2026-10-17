<template>
  <!-- 作成 vs 解決（Jira の Created vs Resolved）。日ごとに作成された課題（アクセント）と解決された課題（緑）の数の折れ線。
       カーソルを合わせた日の縦線と、その日の件数を出す -->
  <figure class="m-0">
    <div class="relative" @mouseleave="hovered = null">
      <svg
        :viewBox="`0 0 ${WIDTH} ${HEIGHT}`"
        class="block w-full h-auto"
        role="img"
        :aria-label="`作成 vs 解決。直近 ${trend.Days.length} 日で作成 ${trend.Created} 件、解決 ${trend.Resolved} 件`"
      >
        <g v-for="tick in geometry.yTicks" :key="`y${tick.label}`">
          <line :x1="geometry.plot.left" :x2="geometry.plot.right" :y1="tick.y" :y2="tick.y" class="stroke-line" stroke-width="1" />
          <text :x="geometry.plot.left - 6" :y="tick.y" text-anchor="end" dominant-baseline="middle" class="fill-muted text-[11px]">{{ tick.label }}</text>
        </g>
        <!-- 端の日付は枠からはみ出さないよう、内側に寄せる -->
        <text
          v-for="tick in geometry.xTicks"
          :key="`x${tick.i}`"
          :x="tick.x" :y="geometry.plot.bottom + 16"
          :text-anchor="tick.x > WIDTH - 24 ? 'end' : tick.x < 24 + PADDING.left ? 'start' : 'middle'"
          class="fill-muted text-[11px]"
        >
          {{ tick.label }}
        </text>

        <line
          v-if="hoveredPoint"
          :x1="hoveredPoint.x" :x2="hoveredPoint.x" :y1="geometry.plot.top" :y2="geometry.plot.bottom"
          class="stroke-muted" stroke-width="1" stroke-dasharray="3 3"
        />
        <path :d="geometry.createdPath" fill="none" class="stroke-accent" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
        <path :d="geometry.resolvedPath" fill="none" class="stroke-success" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
        <template v-if="hoveredPoint">
          <circle :cx="hoveredPoint.x" :cy="hoveredPoint.created" r="4" class="fill-accent" stroke="var(--kb-surface)" stroke-width="2" />
          <circle :cx="hoveredPoint.x" :cy="hoveredPoint.resolved" r="4" class="fill-success" stroke="var(--kb-surface)" stroke-width="2" />
        </template>

        <!-- カーソルを合わせる範囲（日ごと。隣の日との中間まで） -->
        <rect
          v-for="(point, index) in geometry.points"
          :key="point.Date"
          :x="point.x - geometry.half" :y="geometry.plot.top"
          :width="geometry.half * 2" :height="geometry.plot.bottom - geometry.plot.top"
          fill="transparent"
          @mouseenter="hovered = index"
        />
      </svg>
      <div
        v-if="hoveredPoint"
        class="absolute top-0 pointer-events-none px-2 py-1.5 bg-surface border border-line rounded-md shadow-lg text-xs text-fg whitespace-nowrap"
        :style="tooltipStyle"
      >
        <div class="font-semibold mb-0.5">{{ hoveredPoint.Date.replaceAll('-', '/') }}</div>
        <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-accent" aria-hidden="true" />作成 <span class="ml-auto pl-3 tabular-nums">{{ hoveredPoint.Created }}</span></div>
        <div class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-success" aria-hidden="true" />解決 <span class="ml-auto pl-3 tabular-nums">{{ hoveredPoint.Resolved }}</span></div>
      </div>
    </div>
    <figcaption class="flex flex-wrap items-center gap-x-5 gap-y-1 mt-2 text-xs text-muted">
      <span class="inline-flex items-center gap-1.5">
        <span class="w-4 h-0.5 rounded bg-accent" aria-hidden="true" />
        作成 <span class="font-semibold text-fg tabular-nums">{{ trend.Created }}</span>
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="w-4 h-0.5 rounded bg-success" aria-hidden="true" />
        解決 <span class="font-semibold text-fg tabular-nums">{{ trend.Resolved }}</span>
      </span>
      <span>直近 {{ trend.Days.length }} 日</span>
    </figcaption>
  </figure>
</template>

<script setup>
import { ref, computed } from 'vue';
import { trendGeometry } from '../utils/dashboard';

const props = defineProps({
  // GET /dashboard/created-vs-resolved の結果
  trend: {
    type: Object,
    required: true
  }
});

const WIDTH = 520;
const HEIGHT = 200;
const PADDING = { top: 10, right: 12, bottom: 24, left: 32 };

const geometry = computed(() => trendGeometry(props.trend, { width: WIDTH, height: HEIGHT, padding: PADDING }));
const hovered = ref(null);
const hoveredPoint = computed(() => (hovered.value === null ? null : geometry.value.points[hovered.value]));

// ツールチップはカーソルの日の横に出す（右半分では左側）
const tooltipStyle = computed(() => {
  const ratio = hoveredPoint.value.x / WIDTH;
  return ratio > 0.5
    ? { right: `${(1 - ratio) * 100 + 2}%` }
    : { left: `${ratio * 100 + 2}%` };
});
</script>
