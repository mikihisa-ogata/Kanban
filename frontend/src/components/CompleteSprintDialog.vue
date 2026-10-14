<template>
  <!-- スプリントの完了（Jira の「スプリントを完了」）。完了していない課題の移し先を選ぶ -->
  <div class="fixed inset-0 bg-overlay flex items-center justify-center z-50" @click.self="$emit('cancel')">
    <form
      role="dialog"
      aria-modal="true"
      aria-labelledby="complete-sprint-title"
      class="bg-surface rounded-2xl px-8 pt-7 pb-6 shadow-2xl mx-4 max-w-md w-full flex flex-col gap-4"
      @submit.prevent="$emit('submit', moveTo)"
    >
      <h3 id="complete-sprint-title" class="m-0 text-lg font-bold text-fg">「{{ sprint.Name }}」を完了</h3>
      <ul class="m-0 pl-5 text-sm text-fg">
        <li>完了した課題: {{ doneCount }} 件</li>
        <li>完了していない課題: {{ openCount }} 件</li>
      </ul>
      <div v-if="openCount > 0">
        <label for="complete-sprint-move-to" class="block mb-1 text-sm font-semibold text-fg">完了していない課題の移し先</label>
        <select
          id="complete-sprint-move-to"
          v-model="moveTo"
          class="w-full px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface cursor-pointer focus:outline-none focus:border-accent"
        >
          <option :value="0">バックログ</option>
          <option v-for="target in targets" :key="target.ID" :value="target.ID">{{ target.Name }}</option>
        </select>
      </div>
      <p class="m-0 text-xs text-muted">完了した課題とポイントは、このスプリントの記録として残ります。</p>

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
          :disabled="busy"
          class="py-2 px-5 bg-accent text-on-accent border-2 border-accent rounded-lg text-sm font-semibold cursor-pointer hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed"
        >
          スプリントを完了
        </button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { ref } from 'vue';

const props = defineProps({
  sprint: {
    type: Object,
    required: true
  },
  doneCount: {
    type: Number,
    required: true
  },
  openCount: {
    type: Number,
    required: true
  },
  // 移し先にできる、同じスペースの未開始のスプリント
  targets: {
    type: Array,
    default: () => []
  },
  busy: {
    type: Boolean,
    default: false
  }
});

defineEmits(['submit', 'cancel']);

// 既定は次の未開始のスプリント（なければバックログ）。Jira と同じ
const moveTo = ref(props.targets[0]?.ID ?? 0);
</script>
