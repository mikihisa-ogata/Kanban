<template>
  <!-- 課題の詳細。全画面のページ（/browse/:key）と詳細モーダルの両方で使う。
       Jira と同じく左に要約・説明、右にステータスなどの詳細欄を置き、各項目はその場で編集する -->
  <div class="text-gray-900">
    <div class="flex items-center gap-2 min-h-8">
      <nav class="flex items-center gap-1.5 min-w-0 text-sm text-gray-500">
        <span v-if="spaceTitle" class="truncate">{{ spaceTitle }}</span>
        <span v-if="spaceTitle" aria-hidden="true">/</span>
        <!-- サブタスクは Jira と同じく親課題をパンくずに出す -->
        <template v-if="parentTask">
          <IssueTypeIcon :type="parentTask.IssueType" />
          <button
            type="button"
            class="bg-transparent border-0 p-0 font-mono font-semibold text-gray-600 cursor-pointer hover:text-blue-600 hover:underline"
            :title="`親課題 ${parentTask.Key}「${parentTask.Title}」を開く`"
            @click="$emit('open', parentTask.Key)"
          >
            {{ parentTask.Key }}
          </button>
          <span aria-hidden="true">/</span>
        </template>
        <IssueTypeIcon :type="task.IssueType" />
        <NuxtLink :to="issuePath(task.Key)" class="font-mono font-semibold text-gray-600 no-underline hover:text-blue-600 hover:underline">
          {{ task.Key }}
        </NuxtLink>
      </nav>
      <button
        type="button"
        class="shrink-0 bg-transparent border-0 p-1 rounded text-gray-400 cursor-pointer hover:bg-gray-100 hover:text-gray-600"
        :title="copied ? 'コピーしました' : '課題キーをコピー'"
        @click="copyKey"
      >
        <svg v-if="!copied" class="w-4 h-4 block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <rect x="9" y="9" width="12" height="12" rx="2" />
          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
        </svg>
        <svg v-else class="w-4 h-4 block text-emerald-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M5 13l4 4L19 7" />
        </svg>
      </button>
      <span v-if="copied" class="text-xs font-semibold text-emerald-600">コピーしました</span>
      <div class="ml-auto flex items-center gap-1">
        <slot name="actions" />
      </div>
    </div>

    <div class="mt-4 flex flex-col md:flex-row gap-8">
      <!-- 左: 要約・説明・添付ファイル・子課題・リンク・アクティビティ（コメントと変更履歴） -->
      <div class="flex-1 min-w-0">
        <input
          v-if="editing === 'title'"
          ref="titleInput"
          v-model="draft"
          type="text"
          aria-label="要約"
          class="w-full -mx-2 px-2 py-1 border-2 border-blue-500 rounded-md text-2xl font-semibold text-gray-900 bg-white focus:outline-none"
          @keydown.enter="handleTitleEnter"
          @keydown.esc="handleEsc"
          @blur="saveTitle"
        />
        <h1
          v-else
          class="m-0 -mx-2 px-2 py-1 border-2 border-transparent rounded-md text-2xl font-semibold leading-snug break-words cursor-text hover:bg-gray-100"
          title="クリックして編集"
          @click="startEdit('title', task.Title)"
        >
          {{ task.Title }}
        </h1>

        <h2 class="mt-6 mb-2 text-sm font-semibold text-gray-700">説明</h2>
        <div v-if="editing === 'description'">
          <MarkdownEditor
            ref="descriptionInput"
            v-model="draft"
            :rows="8"
            aria-label="説明"
            placeholder="説明を入力（Markdown が使えます）"
            @keydown="handleDescriptionKeydown"
          />
          <div class="flex gap-2 mt-2">
            <button
              type="button"
              class="px-4 py-1.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700"
              @click="saveDescription"
            >
              保存
            </button>
            <button
              type="button"
              class="px-4 py-1.5 bg-transparent text-gray-700 border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-gray-100"
              @click="cancelEdit"
            >
              キャンセル
            </button>
          </div>
        </div>
        <!-- Markdown を整形して表示する。リンクやチェックボックスを押したときは編集に入らない -->
        <div
          v-else
          class="-mx-2 px-2 py-1.5 rounded-md cursor-text hover:bg-gray-100"
          title="クリックして編集"
          @click="handleDescriptionClick"
        >
          <MarkdownView v-if="task.Description" :source="task.Description" />
          <span v-else class="text-sm text-gray-400">説明を追加...</span>
        </div>

        <AttachmentSection class="mt-6" :task="task" />

        <!-- サブタスクの下にはサブタスクを作れないので、子課題はサブタスク以外の課題にだけ出す -->
        <SubtaskSection
          v-if="!isSubtask"
          class="mt-6"
          :parent="task"
          :subtasks="subtasks"
          :statuses="statuses"
          @changed="$emit('subtasks-changed')"
          @open="$emit('open', $event)"
        />

        <LinkSection
          class="mt-6"
          :task="task"
          :todos="allTodos"
          @changed="$emit('links-changed')"
          @open="$emit('open', $event)"
        />

        <ActivitySection
          class="mt-8"
          :task="task"
          @changed="$emit('comments-changed')"
          @worklogs-changed="$emit('worklogs-changed')"
        />
      </div>

      <!-- 右: 詳細欄 -->
      <aside class="md:w-80 shrink-0">
        <section class="border border-gray-200 rounded-lg">
          <h2 class="m-0 px-4 py-3 border-b border-gray-200 text-sm font-semibold text-gray-700">詳細</h2>
          <dl class="m-0 px-4 py-3 grid grid-cols-[6rem_minmax(0,1fr)] gap-x-3 gap-y-3 items-center text-sm">
            <dt class="text-gray-600 font-semibold"><label :for="fieldId('issueType')">課題タイプ</label></dt>
            <dd class="m-0 min-w-0 flex items-center gap-2">
              <IssueTypeIcon :type="task.IssueType" />
              <!-- サブタスクは他の課題タイプに変えられない -->
              <span v-if="isSubtask" :id="fieldId('issueType')" class="text-sm text-gray-800">サブタスク</span>
              <OptionSelect
                v-else
                :id="fieldId('issueType')"
                :model-value="task.IssueType"
                :options="ISSUE_TYPE_OPTIONS"
                @update:model-value="save({ issueType: $event })"
              />
            </dd>

            <dt class="text-gray-600 font-semibold"><label :for="fieldId('status')">ステータス</label></dt>
            <dd class="m-0 min-w-0">
              <!-- 課題のスペースのワークフローのステータスから選ぶ -->
              <OptionSelect
                :id="fieldId('status')"
                :model-value="task.StatusID"
                :options="statusOptions"
                @update:model-value="save({ statusId: $event })"
              />
            </dd>

            <dt class="text-gray-600 font-semibold"><label :for="fieldId('priority')">優先度</label></dt>
            <dd class="m-0 min-w-0 flex items-center gap-2">
              <PriorityIcon :priority="task.Priority" />
              <OptionSelect
                :id="fieldId('priority')"
                :model-value="task.Priority"
                :options="PRIORITY_OPTIONS"
                @update:model-value="save({ priority: $event })"
              />
            </dd>

            <dt class="text-gray-600 font-semibold"><label :for="fieldId('storyPoints')">ストーリーポイント</label></dt>
            <dd class="m-0 min-w-0">
              <template v-if="editing === 'storyPoints'">
                <input
                  :id="fieldId('storyPoints')"
                  ref="storyPointsInput"
                  v-model="draft"
                  type="text"
                  inputmode="decimal"
                  placeholder="未設定"
                  class="w-28 px-3 py-1.5 border-2 border-blue-500 rounded-lg text-sm text-gray-900 bg-white focus:outline-none"
                  @keydown.enter="handleStoryPointsEnter"
                  @keydown.esc="handleEsc"
                  @blur="saveStoryPoints"
                />
                <p v-if="storyPointsError" class="mt-1 mb-0 text-xs text-red-600">{{ storyPointsError }}</p>
              </template>
              <button
                v-else
                type="button"
                class="-mx-2 px-2 py-1.5 bg-transparent border-0 rounded-md text-left cursor-pointer hover:bg-gray-100"
                title="クリックして編集"
                @click="startEdit('storyPoints', task.StoryPoints == null ? '' : formatStoryPoints(task.StoryPoints))"
              >
                <span
                  v-if="task.StoryPoints != null"
                  class="inline-flex min-w-6 h-6 px-1.5 rounded-full bg-gray-200 text-xs font-semibold text-gray-700 items-center justify-center"
                >
                  {{ formatStoryPoints(task.StoryPoints) }}
                </span>
                <span v-else class="text-sm text-gray-400">なし</span>
              </button>
            </dd>

            <dt class="self-start pt-2 text-gray-600 font-semibold"><label :for="fieldId('labels')">ラベル</label></dt>
            <dd class="m-0 min-w-0">
              <div v-if="editing === 'labels'">
                <LabelInput :id="fieldId('labels')" v-model="labelsDraft" />
                <div class="flex gap-2 mt-2">
                  <button
                    type="button"
                    class="px-3 py-1 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700"
                    @click="saveLabels"
                  >
                    保存
                  </button>
                  <button
                    type="button"
                    class="px-3 py-1 bg-transparent text-gray-700 border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-gray-100"
                    @click="cancelEdit"
                  >
                    キャンセル
                  </button>
                </div>
              </div>
              <button
                v-else
                type="button"
                class="w-full -mx-2 px-2 py-1.5 flex flex-wrap gap-1 bg-transparent border-0 rounded-md text-left cursor-pointer hover:bg-gray-100"
                title="クリックして編集"
                @click="startEditLabels"
              >
                <span
                  v-for="label in task.Labels"
                  :key="label"
                  class="max-w-full truncate px-1.5 py-0.5 rounded bg-gray-100 border border-gray-300 text-xs text-gray-800"
                >
                  {{ label }}
                </span>
                <span v-if="!task.Labels || !task.Labels.length" class="text-sm text-gray-400">なし</span>
              </button>
            </dd>

            <template v-if="spaces.length">
              <dt class="text-gray-600 font-semibold"><label :for="fieldId('space')">スペース</label></dt>
              <dd class="m-0 min-w-0">
                <!-- サブタスクのスペース・エピックは親課題と同じ -->
                <span v-if="isSubtask" :id="fieldId('space')" class="text-sm text-gray-800" title="親課題と同じ">{{ spaceTitle || 'なし' }}</span>
                <OptionSelect
                  v-else
                  :id="fieldId('space')"
                  :model-value="task.SpaceID"
                  :options="spaceOptions"
                  :empty-value="0"
                  @update:model-value="changeSpace"
                />
              </dd>
            </template>

            <dt class="text-gray-600 font-semibold"><label :for="fieldId('epic')">エピック</label></dt>
            <dd class="m-0 min-w-0">
              <span v-if="isSubtask" :id="fieldId('epic')" class="text-sm text-gray-800" title="親課題と同じ">{{ epicTitle || 'なし' }}</span>
              <OptionSelect
                v-else
                :id="fieldId('epic')"
                :model-value="task.EpicID"
                :options="epicOptions"
                :empty-value="0"
                @update:model-value="save({ epicId: $event })"
              />
            </dd>

            <dt class="text-gray-600 font-semibold"><label :for="fieldId('deadline')">期限</label></dt>
            <dd class="m-0 min-w-0">
              <input
                :id="fieldId('deadline')"
                :value="task.Deadline"
                type="date"
                class="px-3 py-2 border-2 border-gray-300 rounded-lg text-sm text-gray-900 bg-white hover:border-gray-400 focus:outline-none focus:border-blue-500"
                @change="save({ deadline: $event.target.value })"
              />
            </dd>
          </dl>
        </section>

        <TimeTrackingSection
          class="mt-3"
          :task="task"
          @update="save"
          @worklogs-changed="$emit('worklogs-changed')"
        />

        <dl class="mt-3 mb-0 px-1 flex flex-col gap-1 text-xs text-gray-500">
          <div v-if="task.CreatedAt" class="flex gap-1">
            <dt>作成</dt>
            <dd class="m-0">{{ formatDateTime(task.CreatedAt) }}</dd>
          </div>
          <div v-if="task.UpdatedAt" class="flex gap-1">
            <dt>更新</dt>
            <dd class="m-0">{{ formatDateTime(task.UpdatedAt) }}</dd>
          </div>
        </dl>
      </aside>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onUnmounted } from 'vue';
import OptionSelect from './OptionSelect.vue';
import IssueTypeIcon from './IssueTypeIcon.vue';
import PriorityIcon from './PriorityIcon.vue';
import LabelInput from './LabelInput.vue';
import MarkdownView from './MarkdownView.vue';
import MarkdownEditor from './MarkdownEditor.vue';
import SubtaskSection from './SubtaskSection.vue';
import LinkSection from './LinkSection.vue';
import ActivitySection from './ActivitySection.vue';
import AttachmentSection from './AttachmentSection.vue';
import TimeTrackingSection from './TimeTrackingSection.vue';
import { formatStoryPoints, parseStoryPoints } from '../utils/storyPoints';
import { ISSUE_TYPE_OPTIONS, SUBTASK_ISSUE_TYPE } from '../utils/issueTypes';
import { PRIORITY_OPTIONS } from '../utils/priorities';
import { formatDateTime } from '../utils/datetime';
import { statusesOfSpace, statusOptionsOf } from '../utils/statuses';
import { issuePath } from '../utils/spaceRoutes';

const props = defineProps({
  task: {
    type: Object,
    required: true
  },
  epics: {
    type: Array,
    default: () => []
  },
  spaces: {
    type: Array,
    default: () => []
  },
  // この課題のサブタスク
  subtasks: {
    type: Array,
    default: () => []
  },
  // サブタスクの場合は親課題（パンくずに出す）
  parentTask: {
    type: Object,
    default: null
  },
  // 全課題（リンクする課題の検索に使う）
  allTodos: {
    type: Array,
    default: () => []
  },
  // すべてのワークフローのステータス（ステータスの選択肢にする）
  statuses: {
    type: Array,
    default: () => []
  }
});

// update: 変更後の内容（PUT /todos/:id のリクエストの形）。保存は呼び出し側で行う
// subtasks-changed: 子課題を追加・変更・削除した（呼び出し側でサブタスクを取り直す）
// links-changed: 課題リンクを追加・削除した（呼び出し側でこの課題とリンク相手を取り直す）
// comments-changed: コメントを投稿・削除した（呼び出し側でコメント数を取り直す）
// worklogs-changed: 作業ログを記録・編集・削除した（呼び出し側で記録済み・残り時間を取り直す）
// open: 別の課題（親課題・子課題・リンク相手）を開く（課題キー）
const emit = defineEmits(['update', 'subtasks-changed', 'links-changed', 'comments-changed', 'worklogs-changed', 'open']);

const isSubtask = computed(() => props.task.IssueType === SUBTASK_ISSUE_TYPE);
const epicTitle = computed(() => props.epics.find(epic => epic.ID === props.task.EpicID)?.Title ?? '');

// 課題のスペースのステータス。ステータスを読み込む前でも今の値は表示する
const statusOptions = computed(() => {
  const options = statusOptionsOf(statusesOfSpace(props.statuses, props.task.SpaceID));
  if (!options.some(option => option.value === props.task.StatusID)) {
    options.unshift({ value: props.task.StatusID, label: props.task.StatusName || '—' });
  }
  return options;
});

const spaceTitle = computed(() => props.spaces.find(space => space.ID === props.task.SpaceID)?.Title ?? '');

const spaceOptions = computed(() => [
  { value: 0, label: 'なし' },
  ...props.spaces.map(space => ({ value: space.ID, label: space.Title }))
]);

// 課題と同じスペースのエピックだけを選べる
const epicOptions = computed(() => [
  { value: 0, label: 'なし' },
  ...props.epics
    .filter(epic => epic.SpaceID === props.task.SpaceID)
    .map(epic => ({ value: epic.ID, label: epic.Title }))
]);

// ページとモーダルが同時にあっても label の for が重ならないよう、課題ごとに ID を分ける
const fieldId = (name) => `issue-${props.task.ID}-${name}`;

// 今の内容に changes を重ねて保存を依頼する
const save = (changes) => {
  emit('update', {
    title: props.task.Title,
    description: props.task.Description,
    deadline: props.task.Deadline,
    // スペースを変えるとき（changeSpace）は、bff が移動先のワークフローの同じ名前かカテゴリのステータスに合わせる
    statusId: props.task.StatusID,
    issueType: props.task.IssueType,
    priority: props.task.Priority,
    labels: props.task.Labels || [],
    storyPoints: props.task.StoryPoints ?? null,
    epicId: props.task.EpicID,
    spaceId: props.task.SpaceID,
    ...changes
  });
};

// エピックは移動元のスペースに属するので、スペースを変えたら外す
const changeSpace = (spaceId) => {
  save({ spaceId, epicId: 0 });
};

// インライン編集中の項目（'title' | 'description' | 'storyPoints' | 'labels' | null）と入力中の値
const editing = ref(null);
const draft = ref('');
const titleInput = ref(null);
const descriptionInput = ref(null);
const storyPointsInput = ref(null);
const inputs = { title: titleInput, description: descriptionInput, storyPoints: storyPointsInput };

const startEdit = async (field, value) => {
  editing.value = field;
  draft.value = value;
  storyPointsError.value = '';
  await nextTick();
  inputs[field]?.value?.focus();
};

const cancelEdit = () => {
  editing.value = null;
};

// 日本語入力の変換確定の Enter や Esc では保存・取り消しをしない
const isComposing = (event) => event.isComposing || event.keyCode === 229;

const handleEsc = (event) => {
  if (isComposing(event)) return;
  event.preventDefault();
  cancelEdit();
};

const handleTitleEnter = (event) => {
  if (isComposing(event)) return;
  event.preventDefault();
  saveTitle();
};

// 説明は改行できるよう、⌘+Enter / Ctrl+Enter で保存し、Esc で取り消す
const handleDescriptionKeydown = (event) => {
  if (event.key === 'Escape') {
    handleEsc(event);
  } else if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
    if (isComposing(event)) return;
    event.preventDefault();
    saveDescription();
  }
};

const handleDescriptionClick = (event) => {
  if (event.target.closest('a, input')) return;
  startEdit('description', props.task.Description);
};

// 要約は空にできない。空や変更なしなら元に戻すだけ
const saveTitle = () => {
  if (editing.value !== 'title') return;
  const title = draft.value.trim();
  editing.value = null;
  if (title && title !== props.task.Title) {
    save({ title });
  }
};

// ラベルは入力欄で付け外ししてから「保存」でまとめて保存する
const labelsDraft = ref([]);

const startEditLabels = () => {
  labelsDraft.value = [...(props.task.Labels || [])];
  editing.value = 'labels';
};

const saveLabels = () => {
  if (editing.value !== 'labels') return;
  editing.value = null;
  const current = props.task.Labels || [];
  if (labelsDraft.value.length !== current.length || labelsDraft.value.some(label => !current.includes(label))) {
    save({ labels: labelsDraft.value });
  }
};

// ストーリーポイントは空で未設定に戻す。不正な値なら編集を続けてエラーを出す
const storyPointsError = ref('');

const saveStoryPoints = () => {
  if (editing.value !== 'storyPoints') return;
  const { value, error } = parseStoryPoints(draft.value);
  if (error) {
    storyPointsError.value = error;
    return;
  }
  editing.value = null;
  if (value !== (props.task.StoryPoints ?? null)) {
    save({ storyPoints: value });
  }
};

const handleStoryPointsEnter = (event) => {
  if (isComposing(event)) return;
  event.preventDefault();
  saveStoryPoints();
};

const saveDescription = () => {
  if (editing.value !== 'description') return;
  editing.value = null;
  if (draft.value !== props.task.Description) {
    save({ description: draft.value });
  }
};

// 別の課題に切り替わったら編集をやめる
watch(() => props.task.ID, () => {
  editing.value = null;
});

const copied = ref(false);
let copiedTimer = null;

const copyKey = async () => {
  try {
    await navigator.clipboard.writeText(props.task.Key);
    copied.value = true;
    clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      copied.value = false;
    }, 2000);
  } catch (err) {
    console.error('Failed to copy issue key:', err);
  }
};

onUnmounted(() => {
  clearTimeout(copiedTimer);
});
</script>
