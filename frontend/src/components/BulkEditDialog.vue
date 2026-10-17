<template>
  <!-- 課題の一括変更（Jira の一括変更）。変える項目にチェックを入れて値を選ぶ。チェックしなかった項目は今のまま -->
  <div class="fixed inset-0 bg-overlay flex items-center justify-center z-50" @click.self="$emit('cancel')">
    <form
      role="dialog"
      aria-modal="true"
      aria-labelledby="bulk-edit-title"
      class="bg-surface rounded-2xl px-8 pt-7 pb-6 shadow-2xl mx-4 max-w-2xl w-full max-h-[90vh] overflow-y-auto flex flex-col gap-4 text-fg"
      @submit.prevent="submit"
    >
      <h3 id="bulk-edit-title" class="m-0 text-lg font-bold">{{ issues.length }} 件の課題を一括変更</h3>
      <p class="m-0 text-sm text-muted">
        変える項目にチェックを入れてください。1 件でも変更できない課題があれば、どの課題も変わりません。
      </p>

      <div class="grid grid-cols-[auto_8rem_minmax(0,1fr)] gap-x-3 gap-y-3 items-center text-sm">
        <template v-for="field in fields" :key="field.id">
          <input
            :id="`bulk-${field.id}-enabled`"
            v-model="enabled[field.id]"
            type="checkbox"
            :disabled="!!field.disabled"
            class="cursor-pointer disabled:cursor-not-allowed"
          />
          <label :for="`bulk-${field.id}-enabled`" :class="['font-semibold', field.disabled ? 'text-muted' : 'text-fg cursor-pointer']">{{ field.label }}</label>
          <div class="min-w-0">
            <p v-if="field.disabled" class="m-0 text-xs text-muted">{{ field.disabled }}</p>
            <div v-else :class="enabled[field.id] ? '' : 'opacity-50 pointer-events-none'" :aria-disabled="!enabled[field.id]">
              <OptionSelect v-if="field.id === 'status'" id="bulk-status" v-model="form.statusId" :options="statusOptions" />
              <OptionSelect v-else-if="field.id === 'assignee'" id="bulk-assignee" v-model="form.assigneeId" :options="assigneeOptions" :empty-value="0" />
              <OptionSelect v-else-if="field.id === 'priority'" id="bulk-priority" v-model="form.priority" :options="PRIORITY_OPTIONS" />
              <LabelInput v-else-if="field.id === 'addLabels'" id="bulk-add-labels" v-model="form.addLabels" />
              <LabelInput v-else-if="field.id === 'removeLabels'" id="bulk-remove-labels" v-model="form.removeLabels" />
              <OptionSelect v-else-if="field.id === 'sprint'" id="bulk-sprint" v-model="form.sprintId" :options="sprintOptions" :empty-value="0" />
              <OptionSelect v-else-if="field.id === 'epic'" id="bulk-epic" v-model="form.epicId" :options="epicOptions" :empty-value="0" />
              <FilterMenu
                v-else-if="field.id === 'addFixVersions'"
                label="付けるバージョン"
                :options="addVersionOptions"
                :model-value="form.addFixVersionIds"
                empty-text="付けられるバージョンがありません"
                @update:model-value="form.addFixVersionIds = $event"
              />
              <FilterMenu
                v-else-if="field.id === 'removeFixVersions'"
                label="外すバージョン"
                :options="removeVersionOptions"
                :model-value="form.removeFixVersionIds"
                empty-text="このスペースにバージョンがありません"
                @update:model-value="form.removeFixVersionIds = $event"
              />
            </div>
          </div>
        </template>
      </div>
      <p class="m-0 text-xs text-muted">サブタスクのスプリント・エピックは親課題に合わせます（親課題を変えるとサブタスクも移ります）。</p>

      <p v-if="error" class="m-0 text-sm text-danger">{{ error }}</p>
      <div class="flex justify-end gap-2">
        <button
          type="button"
          class="px-4 py-2 bg-transparent border-0 rounded-lg text-sm font-semibold text-fg cursor-pointer hover:bg-raised"
          @click="$emit('cancel')"
        >
          キャンセル
        </button>
        <button
          type="submit"
          :disabled="!changes || busy"
          class="px-4 py-2 bg-accent text-on-accent border-0 rounded-lg text-sm font-semibold cursor-pointer hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ busy ? '変更中...' : `${issues.length} 件を変更` }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { reactive, computed } from 'vue';
import OptionSelect from './OptionSelect.vue';
import LabelInput from './LabelInput.vue';
import FilterMenu from './FilterMenu.vue';
import { PRIORITY_OPTIONS, DEFAULT_PRIORITY } from '../utils/priorities';
import { SPRINT_CLOSED, SPRINT_ACTIVE } from '../utils/sprints';
import { VERSION_ARCHIVED } from '../utils/versions';

const props = defineProps({
  // 変える課題（SpaceID・FixVersionIDs を見る）
  issues: {
    type: Array,
    required: true
  },
  statuses: {
    type: Array,
    default: () => []
  },
  users: {
    type: Array,
    default: () => []
  },
  epics: {
    type: Array,
    default: () => []
  },
  sprints: {
    type: Array,
    default: () => []
  },
  versions: {
    type: Array,
    default: () => []
  },
  busy: {
    type: Boolean,
    default: false
  },
  // 変更に失敗したときのメッセージ
  error: {
    type: String,
    default: ''
  }
});

// submit: 変える項目（POST /issues/bulk-update の ids 以外）
const emit = defineEmits(['submit', 'cancel']);

// 課題のスペース。スプリント・エピック・修正バージョンはスペースごとなので、1 つのスペースの課題だけのときに選べる
const spaceIds = computed(() => [...new Set(props.issues.map(issue => issue.SpaceID))]);
const spaceId = computed(() => (spaceIds.value.length === 1 ? spaceIds.value[0] : null));
const planningDisabled = computed(() => {
  if (spaceIds.value.length > 1) return '複数のスペースの課題を選んでいるため変更できません';
  if (spaceId.value === 0) return 'スペース未割り当ての課題には設定できません';
  return '';
});

// ステータスは課題のスペースのものを名前でまとめる。別のスペースの課題には、bff が同じ名前（なければカテゴリ）のステータスを当てる
const statusOptions = computed(() => {
  const options = [];
  for (const status of props.statuses) {
    if (!spaceIds.value.includes(status.SpaceID) || options.some(option => option.label === status.Name)) continue;
    options.push({ value: status.ID, label: status.Name });
  }
  return options;
});
const assigneeOptions = computed(() => [
  { value: 0, label: '未割り当て' },
  ...props.users.map(user => ({ value: user.ID, label: user.DisplayName }))
]);
const sprintOptions = computed(() => [
  { value: 0, label: 'バックログ' },
  ...props.sprints
    .filter(sprint => sprint.SpaceID === spaceId.value && sprint.State !== SPRINT_CLOSED)
    .map(sprint => ({ value: sprint.ID, label: sprint.State === SPRINT_ACTIVE ? `${sprint.Name}（開始中）` : sprint.Name }))
]);
const epicOptions = computed(() => [
  { value: 0, label: 'なし' },
  ...props.epics.filter(epic => epic.SpaceID === spaceId.value).map(epic => ({ value: epic.ID, label: epic.Title }))
]);
const spaceVersions = computed(() => props.versions.filter(version => version.SpaceID === spaceId.value));
const addVersionOptions = computed(() =>
  spaceVersions.value.filter(version => version.Status !== VERSION_ARCHIVED).map(version => ({ value: version.ID, label: version.Name }))
);
const removeVersionOptions = computed(() => spaceVersions.value.map(version => ({ value: version.ID, label: version.Name })));

const fields = computed(() => [
  { id: 'status', label: 'ステータス', disabled: statusOptions.value.length ? '' : '選べるステータスがありません' },
  { id: 'assignee', label: '担当者', disabled: '' },
  { id: 'priority', label: '優先度', disabled: '' },
  { id: 'addLabels', label: 'ラベルを追加', disabled: '' },
  { id: 'removeLabels', label: 'ラベルを外す', disabled: '' },
  { id: 'sprint', label: 'スプリント', disabled: planningDisabled.value },
  { id: 'epic', label: 'エピック', disabled: planningDisabled.value },
  { id: 'addFixVersions', label: '修正バージョンを追加', disabled: planningDisabled.value },
  { id: 'removeFixVersions', label: '修正バージョンを外す', disabled: planningDisabled.value }
]);

const enabled = reactive({});
const form = reactive({
  statusId: statusOptions.value[0]?.value ?? 0,
  assigneeId: 0,
  priority: DEFAULT_PRIORITY,
  addLabels: [],
  removeLabels: [],
  sprintId: 0,
  epicId: 0,
  addFixVersionIds: [],
  removeFixVersionIds: []
});

// チェックした項目だけを送る。値のない項目（空のラベルなど）しかなければ null
const changes = computed(() => {
  const on = (id) => enabled[id] && !fields.value.find(field => field.id === id)?.disabled;
  const result = {};
  if (on('status') && form.statusId) result.statusId = form.statusId;
  if (on('assignee')) result.assigneeId = form.assigneeId || null;
  if (on('priority')) result.priority = form.priority;
  if (on('addLabels') && form.addLabels.length) result.addLabels = form.addLabels;
  if (on('removeLabels') && form.removeLabels.length) result.removeLabels = form.removeLabels;
  if (on('sprint')) result.sprintId = form.sprintId || null;
  if (on('epic')) result.epicId = form.epicId || null;
  if (on('addFixVersions') && form.addFixVersionIds.length) result.addFixVersionIds = form.addFixVersionIds;
  if (on('removeFixVersions') && form.removeFixVersionIds.length) result.removeFixVersionIds = form.removeFixVersionIds;
  return Object.keys(result).length ? result : null;
});

const submit = () => {
  if (!changes.value || props.busy) return;
  emit('submit', changes.value);
};
</script>
