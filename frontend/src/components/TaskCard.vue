<template>
  <div class="bg-white rounded-lg p-4 mb-4 shadow hover:shadow-lg cursor-grab active:cursor-grabbing transition-all duration-200 hover:-translate-y-1" draggable="true" @dragstart="handleDragStart" @click="$emit('open', task)">
    <div class="flex justify-between items-start mb-3">
      <h3 class="m-0 text-base font-semibold text-gray-900 flex-1 leading-relaxed">{{ task.Title }}</h3>
      <button class="bg-transparent border-0 text-gray-400 text-2xl cursor-pointer p-0 w-6 h-6 flex items-center justify-center rounded hover:bg-red-100 hover:text-red-500 transition-all duration-200 leading-none" @click.stop="$emit('delete', task.ID)" title="削除">
        ×
      </button>
    </div>
    <p v-if="task.Description" class="m-0 mb-3 text-sm text-gray-600 whitespace-pre-wrap break-words line-clamp-3" :title="task.Description">{{ task.Description }}</p>
    <div v-if="task.Deadline" class="flex items-center gap-1.5 text-sm text-gray-600">
      <span class="text-sm">📅</span>
      <span>{{ formatDate(task.Deadline) }}</span>
    </div>
    <!-- ラベル -->
    <div v-if="task.Labels && task.Labels.length" class="flex flex-wrap gap-1 mb-1">
      <span
        v-for="label in task.Labels"
        :key="label"
        class="max-w-full truncate px-1.5 py-0.5 rounded bg-gray-100 border border-gray-200 text-xs text-gray-700"
      >
        {{ label }}
      </span>
    </div>
    <!-- 課題タイプのアイコンと課題キー（Jira と同じくカードの左下）。ストーリーポイントと優先度は右下 -->
    <div class="mt-3 flex items-center gap-1.5">
      <IssueTypeIcon :type="task.IssueType" />
      <span v-if="task.Key" class="font-mono text-xs font-semibold text-gray-500">{{ task.Key }}</span>
      <!-- ストーリーポイントは Jira と同じく丸いバッジ -->
      <span
        v-if="task.StoryPoints != null"
        class="ml-auto min-w-6 h-6 px-1.5 rounded-full bg-gray-200 text-xs font-semibold text-gray-700 flex items-center justify-center"
        :title="`ストーリーポイント: ${formatStoryPoints(task.StoryPoints)}`"
      >
        {{ formatStoryPoints(task.StoryPoints) }}
      </span>
      <PriorityIcon :priority="task.Priority" :class="task.StoryPoints != null ? '' : 'ml-auto'" />
    </div>
    <div v-if="task.Done" class="mt-3 pt-3 border-t border-gray-200 flex items-center gap-1.5 text-sm text-emerald-600 font-semibold">
      <span class="text-sm">✓</span>
      <span>完了</span>
    </div>
  </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue';
import IssueTypeIcon from './IssueTypeIcon.vue';
import PriorityIcon from './PriorityIcon.vue';
import { formatStoryPoints } from '../utils/storyPoints';

const props = defineProps({
  task: {
    type: Object,
    required: true
  }
});

const emit = defineEmits(['dragstart', 'delete', 'open']);

const handleDragStart = (event) => {
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('taskId', props.task.ID);
  emit('dragstart', props.task);
};

const formatDate = (dateString) => {
  const date = new Date(dateString);
  return date.toLocaleDateString('ja-JP', { year: 'numeric', month: 'short', day: 'numeric' });
};

const getStatusLabel = (status) => {
  const labels = {
    'Open': 'オープン',
    'InProgress': '進行中',
    'Waiting': '待ち状況',
    'Closed': 'クローズ'
  };
  return labels[status] || status;
};
</script>
