<template>
  <!-- 「作業を記録」と作業ログの編集のダイアログ。作業時間・開始日時・メモと、残り時間の調整方法を入力する -->
  <div
    class="fixed inset-0 bg-overlay flex items-center justify-center z-[60]"
    @click.self="$emit('cancel')"
  >
    <form
      role="dialog"
      aria-modal="true"
      aria-labelledby="worklog-dialog-title"
      class="bg-surface rounded-2xl px-8 pt-7 pb-6 shadow-2xl mx-4 max-w-md w-full text-fg"
      @submit.prevent="submit"
    >
      <h3 id="worklog-dialog-title" class="m-0 text-lg font-bold">{{ worklog ? '作業ログを編集' : '作業を記録' }}</h3>
      <p class="mt-1 mb-0 text-xs text-muted truncate">{{ task.Key }} {{ task.Title }}</p>

      <label class="block mt-5 text-sm font-semibold text-fg" for="worklog-time-spent">作業時間</label>
      <input
        id="worklog-time-spent"
        ref="timeSpentInput"
        v-model="timeSpent"
        type="text"
        placeholder="例: 2h 30m"
        class="mt-1 w-full px-3 py-2 border-2 border-line rounded-lg text-sm bg-surface focus:outline-none focus:border-accent"
      />
      <p class="mt-1 mb-0 text-xs text-muted">{{ DURATION_FORMAT_HINT }}</p>

      <label class="block mt-4 text-sm font-semibold text-fg" for="worklog-started-at">開始日時</label>
      <input
        id="worklog-started-at"
        v-model="startedAt"
        type="datetime-local"
        class="mt-1 w-full px-3 py-2 border-2 border-line rounded-lg text-sm bg-surface focus:outline-none focus:border-accent"
      />

      <fieldset class="mt-4 m-0 p-0 border-0">
        <legend class="p-0 text-sm font-semibold text-fg">残り時間</legend>
        <label class="mt-1 flex items-center gap-2 text-sm text-fg cursor-pointer">
          <input v-model="remainingMode" type="radio" value="auto" />
          自動で調整する
          <span v-if="autoRemainingText" class="text-xs text-muted">（{{ autoRemainingText }}）</span>
        </label>
        <label class="mt-1 flex items-center gap-2 text-sm text-fg cursor-pointer">
          <input v-model="remainingMode" type="radio" value="set" />
          指定する
          <input
            v-model="remaining"
            type="text"
            placeholder="例: 1d"
            aria-label="新しい残り時間"
            :disabled="remainingMode !== 'set'"
            class="w-28 px-2 py-1 border-2 border-line rounded-md text-sm bg-surface focus:outline-none focus:border-accent disabled:bg-raised"
          />
        </label>
      </fieldset>

      <label class="block mt-4 text-sm font-semibold text-fg" for="worklog-note">メモ</label>
      <textarea
        id="worklog-note"
        v-model="note"
        rows="3"
        placeholder="何をしたか（任意）"
        class="mt-1 w-full px-3 py-2 border-2 border-line rounded-lg text-sm bg-surface resize-y focus:outline-none focus:border-accent"
      ></textarea>

      <p v-if="error" class="mt-2 mb-0 text-xs text-danger">{{ error }}</p>

      <div class="flex justify-end gap-3 mt-5">
        <button
          type="button"
          class="py-2 px-5 bg-surface text-fg border-2 border-line rounded-lg text-sm font-semibold cursor-pointer hover:bg-raised"
          @click="$emit('cancel')"
        >
          キャンセル
        </button>
        <button
          type="submit"
          :disabled="busy"
          class="py-2 px-5 bg-accent text-on-accent border-2 border-accent rounded-lg text-sm font-semibold cursor-pointer hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ worklog ? '保存' : '記録' }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { worklogsApi } from '../api/worklogs';
import { formatDuration, parseDuration, DURATION_FORMAT_HINT } from '../utils/duration';

const props = defineProps({
  // 作業を記録する課題
  task: {
    type: Object,
    required: true
  },
  // 編集する作業ログ。null なら新しく記録する
  worklog: {
    type: Object,
    default: null
  }
});

// saved: 記録・編集した（呼び出し側で課題と作業ログを取り直す）
const emit = defineEmits(['saved', 'cancel']);

// datetime-local の値（ブラウザのタイムゾーンの「2026-10-11T10:00」）
const toLocalInput = (date) => {
  const pad = (n) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
};

const timeSpent = ref(props.worklog ? formatDuration(props.worklog.Minutes) : '');
const startedAt = ref(toLocalInput(props.worklog ? new Date(props.worklog.StartedAt) : new Date()));
const note = ref(props.worklog?.Note ?? '');
const remainingMode = ref('auto');
const remaining = ref(props.task.RemainingEstimate != null ? formatDuration(props.task.RemainingEstimate) : '');
const busy = ref(false);
const error = ref('');
const timeSpentInput = ref(null);

// 「自動で調整する」を選んだときの残り時間の見込み（bff の自動調整と同じ計算）
const autoRemainingText = computed(() => {
  const { value } = parseDuration(timeSpent.value);
  if (value == null) return '';
  const delta = value - (props.worklog?.Minutes ?? 0);
  const { RemainingEstimate: current, OriginalEstimate: original, TimeSpent: spent } = props.task;
  if (current != null) return `残り ${formatDuration(Math.max(current - delta, 0))} になります`;
  if (original != null) return `残り ${formatDuration(Math.max(original - (spent + delta), 0))} になります`;
  return '見積もりがないため変わりません';
});

const submit = async () => {
  if (busy.value) return;
  const spent = parseDuration(timeSpent.value);
  if (spent.error || !spent.value) {
    error.value = spent.error || '作業時間を 1 分以上で入力してください';
    return;
  }
  if (!startedAt.value) {
    error.value = '開始日時を入力してください';
    return;
  }
  const payload = {
    timeSpent: timeSpent.value,
    startedAt: new Date(startedAt.value).toISOString(),
    note: note.value
  };
  if (remainingMode.value === 'set') {
    const left = parseDuration(remaining.value);
    if (left.error) {
      error.value = '残り時間: ' + left.error;
      return;
    }
    payload.remainingEstimate = remaining.value;
  }
  busy.value = true;
  try {
    error.value = '';
    if (props.worklog) {
      await worklogsApi.update(props.worklog.ID, payload);
    } else {
      await worklogsApi.create(props.task.ID, payload);
    }
    emit('saved');
  } catch (err) {
    error.value = (props.worklog ? '作業ログの更新' : '作業の記録') + 'に失敗しました: ' + err.message;
    console.error('Failed to save worklog:', err);
  } finally {
    busy.value = false;
  }
};

const handleKeydown = (event) => {
  if (event.key === 'Escape') {
    emit('cancel');
  }
};

onMounted(() => {
  timeSpentInput.value?.focus();
  window.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown);
});
</script>
