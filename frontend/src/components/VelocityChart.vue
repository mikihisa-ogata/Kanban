<template>
  <!-- ベロシティ（Jira のベロシティチャート）。スプリントごとに、灰色の棒がコミットメント（開始時のポイント）、
       緑の棒が完了したポイント。青の破線が平均ベロシティ。棒にカーソルを合わせると値が出る -->
  <figure class="m-0">
    <div class="overflow-x-auto">
      <svg
        :viewBox="`0 0 ${WIDTH} ${HEIGHT}`"
        class="block w-full min-w-[600px] h-auto"
        role="img"
        :aria-label="`ベロシティチャート。直近 ${velocity.Sprints.length} スプリントの平均ベロシティ ${formatStoryPoints(velocity.Average)} ポイント`"
      >
        <!-- 縦軸の目盛りと横線 -->
        <g v-for="tick in geometry.yTicks" :key="`y${tick.label}`">
          <line :x1="geometry.plot.left" :x2="geometry.plot.right" :y1="tick.y" :y2="tick.y" class="stroke-line" stroke-width="1" />
          <text :x="geometry.plot.left - 8" :y="tick.y" text-anchor="end" dominant-baseline="middle" class="fill-muted text-[11px]">{{ tick.label }}</text>
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

        <g v-for="group in geometry.groups" :key="group.sprint.SprintID">
          <rect
            v-for="bar in [group.commitment, group.completed]"
            :key="bar === group.commitment ? 'c' : 'd'"
            :x="bar.x"
            :y="bar.y"
            :width="bar.width"
            :height="bar.height"
            :class="bar === group.commitment ? 'fill-muted/50' : 'fill-success'"
            rx="2"
          >
            <title>{{ barTitle(group, bar) }}</title>
          </rect>
          <text
            v-for="bar in [group.commitment, group.completed]"
            :key="bar === group.commitment ? 'cl' : 'dl'"
            :x="bar.x + bar.width / 2"
            :y="bar.y - 4"
            text-anchor="middle"
            class="fill-fg text-[11px]"
          >
            {{ bar.label }}
          </text>
          <!-- スプリント名（長い名前は省略し、カーソルで全体を出す） -->
          <text :x="group.center" :y="geometry.plot.bottom + 18" text-anchor="middle" class="fill-muted text-[11px]">
            {{ shortName(group.sprint.Name) }}
            <title>{{ group.sprint.Name }}</title>
          </text>
        </g>

        <!-- 平均ベロシティ -->
        <line
          v-if="geometry.average !== null"
          :x1="geometry.plot.left"
          :x2="geometry.plot.right"
          :y1="geometry.average"
          :y2="geometry.average"
          class="stroke-accent"
          stroke-width="1.5"
          stroke-dasharray="6 4"
        >
          <title>平均ベロシティ {{ formatStoryPoints(velocity.Average) }} ポイント</title>
        </line>
      </svg>
    </div>
    <figcaption class="flex flex-wrap items-center gap-x-5 gap-y-1 mt-2 text-xs text-muted">
      <span class="inline-flex items-center gap-1.5">
        <span class="w-3 h-3 rounded-sm bg-muted/50" aria-hidden="true" />
        コミットメント（開始時）
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="w-3 h-3 rounded-sm bg-success" aria-hidden="true" />
        完了
      </span>
      <span class="inline-flex items-center gap-1.5">
        <svg class="w-6 h-2" viewBox="0 0 24 8" aria-hidden="true"><line x1="0" y1="4" x2="24" y2="4" class="stroke-accent" stroke-width="1.5" stroke-dasharray="6 4" /></svg>
        平均ベロシティ
      </span>
    </figcaption>
  </figure>
</template>

<script setup>
import { computed } from 'vue';
import { velocityGeometry } from '../utils/velocity';
import { formatStoryPoints } from '../utils/storyPoints';

const props = defineProps({
  // GET /spaces/:id/velocity の結果（Sprints が 1 つ以上）
  velocity: {
    type: Object,
    required: true
  }
});

const WIDTH = 760;
const HEIGHT = 320;
const PADDING = { top: 20, right: 20, bottom: 32, left: 56 };

const geometry = computed(() => velocityGeometry(props.velocity, { width: WIDTH, height: HEIGHT, padding: PADDING }));

const barTitle = (group, bar) =>
  `${group.sprint.Name}\n${bar === group.commitment ? 'コミットメント' : '完了'} ${bar.label} ポイント`;

// 7 本並べても重ならない長さに切る
const shortName = (name) => (name.length > 12 ? `${name.slice(0, 11)}…` : name);
</script>
