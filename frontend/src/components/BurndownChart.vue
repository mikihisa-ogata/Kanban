<template>
  <!-- スプリントのバーンダウン（Jira のスプリントバーンダウンチャート）。縦軸はストーリーポイント、横軸は日付。
       灰色の破線が理想線、赤の階段が残りの作業。点にカーソルを合わせると、その出来事の課題が出る -->
  <figure class="m-0">
    <div class="overflow-x-auto">
      <svg
        :viewBox="`0 0 ${WIDTH} ${HEIGHT}`"
        class="block w-full min-w-[600px] h-auto"
        role="img"
        :aria-label="`バーンダウンチャート。開始時 ${formatStoryPoints(burndown.StartPoints)} ポイント、現在 ${formatStoryPoints(currentRemaining)} ポイント`"
      >
        <!-- 土日 -->
        <rect
          v-for="(band, index) in geometry.weekends"
          :key="`w${index}`"
          :x="band.x"
          :y="geometry.plot.top"
          :width="band.width"
          :height="geometry.plot.bottom - geometry.plot.top"
          class="fill-raised"
        />
        <!-- 縦軸の目盛りと横線 -->
        <g v-for="tick in geometry.yTicks" :key="`y${tick.label}`">
          <line :x1="geometry.plot.left" :x2="geometry.plot.right" :y1="tick.y" :y2="tick.y" class="stroke-line" stroke-width="1" />
          <text :x="geometry.plot.left - 8" :y="tick.y" text-anchor="end" dominant-baseline="middle" class="fill-muted text-[11px]">{{ tick.label }}</text>
        </g>
        <!-- 横軸の日付 -->
        <g v-for="tick in geometry.xTicks" :key="`x${tick.label}`">
          <line :x1="tick.x" :x2="tick.x" :y1="geometry.plot.bottom" :y2="geometry.plot.bottom + 4" class="stroke-line" stroke-width="1" />
          <text :x="tick.x" :y="geometry.plot.bottom + 18" text-anchor="start" class="fill-muted text-[11px]">{{ tick.label }}</text>
        </g>
        <line :x1="geometry.plot.left" :x2="geometry.plot.left" :y1="geometry.plot.top" :y2="geometry.plot.bottom" class="stroke-line" stroke-width="1" />
        <text
          :x="14"
          :y="(geometry.plot.top + geometry.plot.bottom) / 2"
          text-anchor="middle"
          :transform="`rotate(-90 14 ${(geometry.plot.top + geometry.plot.bottom) / 2})`"
          class="fill-muted text-[11px]"
        >
          ストーリーポイント
        </text>

        <!-- 理想線 -->
        <line
          :x1="geometry.ideal.x1"
          :y1="geometry.ideal.y1"
          :x2="geometry.ideal.x2"
          :y2="geometry.ideal.y2"
          class="stroke-muted"
          stroke-width="1.5"
          stroke-dasharray="6 4"
        />
        <!-- 開始中なら、今の位置に縦線 -->
        <line
          v-if="active"
          :x1="geometry.untilX"
          :x2="geometry.untilX"
          :y1="geometry.plot.top"
          :y2="geometry.plot.bottom"
          class="stroke-accent"
          stroke-width="1"
          stroke-dasharray="3 3"
        />
        <!-- 残りの作業 -->
        <path :d="geometry.path" fill="none" class="stroke-danger" stroke-width="2" stroke-linejoin="round" />
        <circle
          v-for="(marker, index) in geometry.markers"
          :key="`m${index}`"
          :cx="marker.x"
          :cy="marker.y"
          r="3.5"
          class="fill-surface stroke-danger"
          stroke-width="2"
        >
          <title>{{ markerTitle(marker.event) }}</title>
        </circle>
      </svg>
    </div>
    <figcaption class="flex flex-wrap items-center gap-x-5 gap-y-1 mt-2 text-xs text-muted">
      <span class="inline-flex items-center gap-1.5">
        <svg class="w-6 h-2" viewBox="0 0 24 8" aria-hidden="true"><line x1="0" y1="4" x2="24" y2="4" class="stroke-danger" stroke-width="2" /></svg>
        残りの作業
      </span>
      <span class="inline-flex items-center gap-1.5">
        <svg class="w-6 h-2" viewBox="0 0 24 8" aria-hidden="true"><line x1="0" y1="4" x2="24" y2="4" class="stroke-muted" stroke-width="1.5" stroke-dasharray="6 4" /></svg>
        理想線
      </span>
      <span v-if="active" class="inline-flex items-center gap-1.5">
        <svg class="w-6 h-2" viewBox="0 0 24 8" aria-hidden="true"><line x1="0" y1="4" x2="24" y2="4" class="stroke-accent" stroke-width="1.5" stroke-dasharray="3 3" /></svg>
        現在
      </span>
    </figcaption>
  </figure>
</template>

<script setup>
import { computed } from 'vue';
import { burndownGeometry, formatChange, BURNDOWN_EVENT_LABELS } from '../utils/burndown';
import { formatStoryPoints } from '../utils/storyPoints';

const props = defineProps({
  // GET /sprints/:id/burndown の結果
  burndown: {
    type: Object,
    required: true
  },
  // 開始中のスプリントか（今の位置に線を引く）
  active: {
    type: Boolean,
    default: false
  }
});

const WIDTH = 760;
const HEIGHT = 320;
const PADDING = { top: 16, right: 20, bottom: 32, left: 56 };

const geometry = computed(() => burndownGeometry(props.burndown, { width: WIDTH, height: HEIGHT, padding: PADDING }));

const currentRemaining = computed(() => {
  const events = props.burndown.Events;
  return events.length ? events[events.length - 1].Remaining : props.burndown.StartPoints;
});

const markerTitle = (event) => {
  const change = formatChange(event.Change);
  return `${event.Key} ${event.Title}\n${BURNDOWN_EVENT_LABELS[event.Kind] ?? event.Kind}${change ? ` ${change}` : ''}（残り ${formatStoryPoints(event.Remaining)}）`;
};
</script>
