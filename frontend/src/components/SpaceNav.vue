<template>
  <!-- スペース内のページへの左ナビ。画面が狭いときはアイコンだけにする -->
  <nav class="h-[calc(100vh-4.25rem)] overflow-y-auto w-14 md:w-56 bg-white/90 border-r border-gray-200 px-2 md:px-3 py-4 flex flex-col gap-1">
    <div class="flex items-center gap-2 px-1 md:px-2 pb-4 mb-2 border-b border-gray-200" :title="headerTitle">
      <span
        :class="[
          'shrink-0 w-8 h-8 rounded-lg flex items-center justify-center text-sm font-bold text-white',
          space ? 'bg-teal-500' : 'bg-blue-500'
        ]"
        aria-hidden="true"
      >
        {{ iconLabel }}
      </span>
      <div class="hidden md:block min-w-0">
        <p class="m-0 text-sm font-bold text-gray-900 truncate">{{ space ? space.Title : 'すべてのスペース' }}</p>
        <p v-if="space && space.Key" class="m-0 font-mono text-xs text-gray-500">{{ space.Key }}</p>
      </div>
    </div>

    <NuxtLink
      v-for="item in items"
      :key="item.to"
      :to="item.to"
      :title="item.label"
      :class="[
        'flex items-center gap-3 px-2 py-2 rounded-lg text-sm font-semibold no-underline transition-colors duration-150',
        isActive(item.to) ? 'bg-blue-50 text-blue-700' : 'text-gray-700 hover:bg-gray-100'
      ]"
    >
      <svg class="shrink-0 w-5 h-5 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path v-for="(d, i) in item.icon" :key="i" :d="d" />
      </svg>
      <span class="hidden md:inline">{{ item.label }}</span>
    </NuxtLink>
  </nav>
</template>

<script setup>
import { computed } from 'vue';
import { boardPath, backlogPath, releasesPath, settingsPath } from '../utils/spaceRoutes';

const props = defineProps({
  // 表示中のスペース。null の場合は「すべてのスペース」
  space: {
    type: Object,
    default: null
  }
});

const route = useRoute();

const iconLabel = computed(() => (props.space ? (props.space.Key || props.space.Title).slice(0, 1) : '全'));
const headerTitle = computed(() => (props.space ? `${props.space.Title}（${props.space.Key}）` : 'すべてのスペース'));

// 左ナビのリンク。タイムラインなどは、それぞれのチケットで追加する。バックログはスペースごと（Jira のプロジェクトと同じ）
const items = computed(() => {
  const list = [];
  if (props.space) {
    list.push({
      label: 'バックログ',
      to: backlogPath(props.space),
      icon: ['M8 6h13', 'M8 12h13', 'M8 18h13', 'M3 6h.01', 'M3 12h.01', 'M3 18h.01']
    });
  }
  list.push({
    label: 'ボード',
    to: boardPath(props.space),
    icon: ['M3 4h18v16H3z', 'M9 4v16', 'M15 4v16']
  });
  if (props.space) {
    list.push({
      label: 'リリース',
      to: releasesPath(props.space),
      icon: ['M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z', 'M7 7h.01']
    });
    list.push({
      label: '設定',
      to: settingsPath(props.space),
      icon: ['M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z', 'M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.01a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.01a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z']
    });
  }
  return list;
});

// リリースはバージョンの詳細（/releases/:id）でも選択中にする
const isActive = (to) => route.path === to || (to.endsWith('/releases') && route.path.startsWith(`${to}/`));
</script>
