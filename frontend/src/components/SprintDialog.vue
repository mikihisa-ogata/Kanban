<template>
  <!-- スプリントの開始・編集（Jira の「スプリントを開始」）。名前・期間・ゴールを入れる -->
  <div class="fixed inset-0 bg-overlay flex items-center justify-center z-50" @click.self="$emit('cancel')">
    <form
      role="dialog"
      aria-modal="true"
      aria-labelledby="sprint-dialog-title"
      class="bg-surface rounded-2xl px-8 pt-7 pb-6 shadow-2xl mx-4 max-w-lg w-full flex flex-col gap-4"
      @submit.prevent="submit"
    >
      <h3 id="sprint-dialog-title" class="m-0 text-lg font-bold text-fg">
        {{ mode === 'start' ? 'スプリントを開始' : 'スプリントを編集' }}
      </h3>
      <p v-if="mode === 'start'" class="m-0 text-sm text-muted">
        このスプリントの課題 {{ issueCount }} 件で開始します。開始すると、ボードにはこのスプリントの課題だけが表示されます。
      </p>

      <div>
        <label for="sprint-name" class="block mb-1 text-sm font-semibold text-fg">スプリント名 <span class="text-danger">*</span></label>
        <input
          id="sprint-name"
          v-model="form.name"
          type="text"
          maxlength="100"
          class="w-full px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface focus:outline-none focus:border-accent"
        />
      </div>
      <div class="flex gap-3">
        <div class="flex-1">
          <label for="sprint-start" class="block mb-1 text-sm font-semibold text-fg">開始日 <span v-if="needDates" class="text-danger">*</span></label>
          <input
            id="sprint-start"
            v-model="form.startDate"
            type="date"
            class="w-full px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface focus:outline-none focus:border-accent"
          />
        </div>
        <div class="flex-1">
          <label for="sprint-end" class="block mb-1 text-sm font-semibold text-fg">終了日 <span v-if="needDates" class="text-danger">*</span></label>
          <input
            id="sprint-end"
            v-model="form.endDate"
            type="date"
            class="w-full px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface focus:outline-none focus:border-accent"
          />
        </div>
      </div>
      <div>
        <label for="sprint-goal" class="block mb-1 text-sm font-semibold text-fg">スプリントのゴール</label>
        <textarea
          id="sprint-goal"
          v-model="form.goal"
          rows="3"
          maxlength="1000"
          class="w-full px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface resize-y focus:outline-none focus:border-accent"
        ></textarea>
      </div>
      <p v-if="validationError" class="m-0 text-sm text-danger">{{ validationError }}</p>

      <div class="flex justify-end gap-3">
        <button
          type="button"
          class="py-2 px-5 bg-surface text-fg border-2 border-line rounded-lg text-sm font-semibold cursor-pointer hover:bg-raised"
          @click="$emit('cancel')"
        >
          キャンセル
        </button>
        <button
          type="submit"
          :disabled="!!validationError || busy"
          class="py-2 px-5 bg-accent text-on-accent border-2 border-accent rounded-lg text-sm font-semibold cursor-pointer hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ mode === 'start' ? '開始' : '保存' }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import { SPRINT_ACTIVE, dateAfter } from '../utils/sprints';

const props = defineProps({
  sprint: {
    type: Object,
    required: true
  },
  // start: 未開始のスプリントを開始する、edit: 名前・期間・ゴールを変える
  mode: {
    type: String,
    default: 'start',
    validator: (value) => ['start', 'edit'].includes(value)
  },
  // スプリントの課題の件数（開始の説明に出す）
  issueCount: {
    type: Number,
    default: 0
  },
  busy: {
    type: Boolean,
    default: false
  }
});

const emit = defineEmits(['submit', 'cancel']);

// 開始するときは、期間が未設定なら今日から 2 週間（Jira の既定）にする
const form = ref({
  name: props.sprint.Name,
  goal: props.sprint.Goal,
  startDate: props.sprint.StartDate || (props.mode === 'start' ? dateAfter(0) : ''),
  endDate: props.sprint.EndDate || (props.mode === 'start' ? dateAfter(14) : '')
});

// 開始中・開始するスプリントには期間が要る
const needDates = computed(() => props.mode === 'start' || props.sprint.State === SPRINT_ACTIVE);

const validationError = computed(() => {
  const { name, startDate, endDate } = form.value;
  if (!name.trim()) return 'スプリント名を入力してください';
  if (needDates.value && (!startDate || !endDate)) return '開始日と終了日を入力してください';
  if (!!startDate !== !!endDate) return '開始日と終了日は両方入力してください';
  if (startDate && endDate && endDate < startDate) return '終了日は開始日以降にしてください';
  return '';
});

const submit = () => {
  if (validationError.value) return;
  emit('submit', { ...form.value, name: form.value.name.trim(), goal: form.value.goal.trim() });
};
</script>
