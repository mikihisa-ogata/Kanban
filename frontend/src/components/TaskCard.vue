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
      <!-- 完了していない課題にブロックされている -->
      <span
        v-if="blockers.length"
        class="flex items-center text-red-600"
        :title="`ブロックされています: ${blockers.map(issue => issue.Key).join(', ')}`"
        role="img"
        :aria-label="`ブロックされています: ${blockers.map(issue => issue.Key).join(', ')}`"
      >
        <svg class="w-4 h-4 block" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
          <circle cx="8" cy="8" r="6" />
          <path d="M3.8 12.2l8.4-8.4" />
        </svg>
      </span>
      <!-- サブタスクの件数（完了 / 全体） -->
      <span
        v-if="subtasks"
        :class="[
          'flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-semibold',
          subtasks.done === subtasks.total ? 'bg-emerald-50 text-emerald-700' : 'bg-gray-100 text-gray-600'
        ]"
        :title="`サブタスク ${subtasks.done}/${subtasks.total} 完了`"
      >
        <IssueTypeIcon type="Subtask" :size="12" />
        {{ subtasks.done }}/{{ subtasks.total }}
      </span>
      <!-- コメント数 -->
      <span
        v-if="task.CommentCount"
        class="flex items-center gap-0.5 text-xs font-semibold text-gray-500"
        :title="`コメント ${task.CommentCount} 件`"
      >
        <svg class="w-3.5 h-3.5 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
        {{ task.CommentCount }}
      </span>
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
import { defineProps, defineEmits, computed, inject } from 'vue';
import IssueTypeIcon from './IssueTypeIcon.vue';
import PriorityIcon from './PriorityIcon.vue';
import { formatStoryPoints } from '../utils/storyPoints';
import { openBlockers } from '../utils/issueLinks';

const props = defineProps({
  task: {
    type: Object,
    required: true
  }
});

const emit = defineEmits(['dragstart', 'delete', 'open']);

// 親課題ごとのサブタスクの件数（ボードが provide する。サブタスクがなければ出さない）
const subtaskProgress = inject('subtaskProgress', null);
const subtasks = computed(() => subtaskProgress?.value?.[props.task.ID] ?? null);

const blockers = computed(() => openBlockers(props.task));

const handleDragStart = (event) => {
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('taskId', props.task.ID);
  emit('dragstart', props.task);
};

const formatDate = (dateString) => {
  const date = new Date(dateString);
  return date.toLocaleDateString('ja-JP', { year: 'numeric', month: 'short', day: 'numeric' });
};
</script>
