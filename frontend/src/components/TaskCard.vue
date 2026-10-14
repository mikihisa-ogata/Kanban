<template>
  <!-- ボードのカード。1 行目は課題キーと担当者、2 行目は要約、3 行目はタイプ・優先度・ラベルと、右にサブタスク・期限・ストーリーポイント。
       削除は課題の詳細から行う -->
  <div
    class="bg-surface border border-line rounded-md px-2.5 py-2 cursor-grab active:cursor-grabbing transition-colors duration-150 hover:border-muted/60"
    draggable="true"
    @dragstart="handleDragStart"
    @click="$emit('open', task)"
  >
    <div class="flex items-center gap-1.5 min-h-5">
      <span v-if="task.Key" class="font-mono text-xs text-muted">{{ task.Key }}</span>
      <!-- 完了していない課題にブロックされている -->
      <span
        v-if="blockers.length"
        class="flex items-center text-danger"
        :title="`ブロックされています: ${blockers.map(issue => issue.Key).join(', ')}`"
        role="img"
        :aria-label="`ブロックされています: ${blockers.map(issue => issue.Key).join(', ')}`"
      >
        <svg class="w-3.5 h-3.5 block" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
          <circle cx="8" cy="8" r="6" />
          <path d="M3.8 12.2l8.4-8.4" />
        </svg>
      </span>
      <UserAvatar
        v-if="task.Assignee"
        class="ml-auto"
        :user="task.Assignee"
        size="xs"
        :title="`担当者: ${task.Assignee.DisplayName}`"
      />
    </div>

    <h3 :class="['mt-1 mb-0 text-sm font-medium leading-snug break-words', done ? 'text-muted' : 'text-fg']">{{ task.Title }}</h3>

    <div class="mt-2 flex flex-wrap items-center gap-1.5">
      <IssueTypeIcon :type="task.IssueType" :size="14" />
      <span class="inline-flex items-center gap-1 px-1.5 py-px rounded border border-line text-[11px] text-muted" :title="`優先度: ${priorityLabel(task.Priority)}`">
        <PriorityIcon :priority="task.Priority" :size="12" />
        {{ priorityLabel(task.Priority) }}
      </span>
      <span
        v-for="label in task.Labels || []"
        :key="label"
        class="inline-flex max-w-32 items-center gap-1 px-1.5 py-px rounded border border-line text-[11px] text-muted"
        :title="`ラベル: ${label}`"
      >
        <span class="shrink-0" :style="{ color: labelColor(label) }" aria-hidden="true">●</span>
        <span class="truncate">{{ label }}</span>
      </span>
      <div class="ml-auto flex items-center gap-2 text-[11px] text-muted">
        <!-- サブタスクの件数（完了 / 全体） -->
        <span
          v-if="subtasks"
          :class="['inline-flex items-center gap-1 tabular-nums', subtasks.done === subtasks.total ? 'text-success' : '']"
          :title="`サブタスク ${subtasks.done}/${subtasks.total} 完了`"
        >
          <IssueTypeIcon type="Subtask" :size="12" />
          {{ subtasks.done }}/{{ subtasks.total }}
        </span>
        <!-- 期限。完了していない課題の期限切れと今日が期限のものは赤 -->
        <span
          v-if="task.Deadline"
          :class="['inline-flex items-center gap-1 tabular-nums', dueAlert ? 'text-danger font-semibold' : '']"
          :title="`期限: ${task.Deadline}${dueAlert === 'overdue' ? '（期限切れ）' : dueAlert === 'today' ? '（今日）' : ''}`"
        >
          <svg class="w-3 h-3 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="3" y="4" width="18" height="18" rx="2" />
            <path d="M16 2v4M8 2v4M3 10h18" />
          </svg>
          {{ shortDate(task.Deadline) }}
        </span>
        <span
          v-if="task.StoryPoints != null"
          class="min-w-5 h-5 px-1 rounded-full bg-raised text-fg font-semibold flex items-center justify-center tabular-nums"
          :title="`ストーリーポイント: ${formatStoryPoints(task.StoryPoints)}`"
        >
          {{ formatStoryPoints(task.StoryPoints) }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { defineProps, defineEmits, computed, inject } from 'vue';
import IssueTypeIcon from './IssueTypeIcon.vue';
import PriorityIcon from './PriorityIcon.vue';
import UserAvatar from './UserAvatar.vue';
import { formatStoryPoints } from '../utils/storyPoints';
import { openBlockers } from '../utils/issueLinks';
import { priorityLabel } from '../utils/priorities';
import { isDoneCategory } from '../utils/statuses';
import { labelColor } from '../utils/spaceColors';

const props = defineProps({
  task: {
    type: Object,
    required: true
  }
});

const emit = defineEmits(['dragstart', 'open']);

// 親課題ごとのサブタスクの件数（ボードが provide する。サブタスクがなければ出さない）
const subtaskProgress = inject('subtaskProgress', null);
const subtasks = computed(() => subtaskProgress?.value?.[props.task.ID] ?? null);

const blockers = computed(() => openBlockers(props.task));

// 完了カテゴリの課題（要約を補助色にする）
const done = computed(() => isDoneCategory(props.task.StatusCategory));

// 期限の注意。overdue は期限切れ、today は今日が期限。完了した課題には出さない
const localToday = () => {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
};
const dueAlert = computed(() => {
  if (!props.task.Deadline || done.value) return null;
  const today = localToday();
  if (props.task.Deadline < today) return 'overdue';
  if (props.task.Deadline === today) return 'today';
  return null;
});

// 期限（YYYY-MM-DD）を「10/20」の形にする
const shortDate = (date) => {
  const [, month, day] = date.split('-');
  return `${Number(month)}/${Number(day)}`;
};

const handleDragStart = (event) => {
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('taskId', props.task.ID);
  emit('dragstart', props.task);
};
</script>
