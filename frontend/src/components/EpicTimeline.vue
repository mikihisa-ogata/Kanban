<template>
  <!-- エピックのタイムライン（Jira のロードマップ）。左にエピック（展開すると子の課題）、右に日付軸のバー。
       左の列と見出しはスクロールしても張り付ける -->
  <div ref="scroller" class="relative overflow-auto bg-surface border border-line rounded-xl" :style="{ maxHeight }">
    <div class="relative" :style="{ width: `${LEFT_WIDTH + chartWidth}px` }">
      <!-- 見出し -->
      <div class="sticky top-0 z-30 flex bg-surface border-b border-line" :style="{ height: `${HEADER_HEIGHT}px` }">
        <div
          class="sticky left-0 z-10 shrink-0 flex items-end px-4 pb-1.5 bg-surface border-r border-line text-xs font-semibold text-muted"
          :style="{ width: `${LEFT_WIDTH}px` }"
        >
          エピック
        </div>
        <div class="relative shrink-0" :style="{ width: `${chartWidth}px` }">
          <div class="absolute inset-x-0 top-0 h-1/2">
            <div
              v-for="segment in headers.top"
              :key="segment.start"
              class="absolute top-0 bottom-0 flex items-center border-l border-line text-xs font-semibold text-fg whitespace-nowrap"
              :style="segmentStyle(segment)"
            >
              <!-- 横にスクロールしても、区間の見出しを左端に残す -->
              <span class="sticky px-2" :style="{ left: `${LEFT_WIDTH}px` }">{{ segment.label }}</span>
            </div>
          </div>
          <div class="absolute inset-x-0 bottom-0 h-1/2">
            <div
              v-for="segment in headers.bottom"
              :key="segment.start"
              :class="[
                'absolute top-0 bottom-0 flex items-center border-l border-line text-[11px] whitespace-nowrap overflow-hidden',
                scale.value === 'week' ? 'justify-center' : 'px-2',
                segment.start === today ? 'text-accent font-bold' : 'text-muted'
              ]"
              :style="segmentStyle(segment)"
            >
              {{ segment.label }}
            </div>
          </div>
          <span
            v-if="todayInRange"
            class="absolute bottom-0.5 -translate-x-1/2 px-1 rounded bg-accent text-on-accent text-[10px] font-semibold whitespace-nowrap"
            :style="{ left: `${todayX}px` }"
          >
            今日
          </span>
        </div>
      </div>

      <!-- 区切り線（週の表示は土日の背景も）と今日の線。行の後ろに敷く -->
      <div class="absolute bottom-0 pointer-events-none" :style="{ top: `${HEADER_HEIGHT}px`, left: `${LEFT_WIDTH}px`, width: `${chartWidth}px` }" aria-hidden="true">
        <template v-if="scale.value === 'week'">
          <div
            v-for="segment in weekendSegments"
            :key="segment.start"
            class="absolute top-0 bottom-0 bg-raised/60"
            :style="segmentStyle(segment)"
          />
        </template>
        <div
          v-for="segment in headers.bottom"
          :key="segment.start"
          class="absolute top-0 bottom-0 border-l border-line/60"
          :style="{ left: `${offsetOf(segment.start)}px` }"
        />
        <div v-if="todayInRange" class="absolute top-0 bottom-0 w-0.5 -ml-px bg-accent z-20" :style="{ left: `${todayX}px` }" />
      </div>

      <!-- 行 -->
      <template v-for="row in rows" :key="row.key">
        <div class="relative flex border-b border-line" :style="{ height: `${ROW_HEIGHT}px` }">
          <!-- 左の列: エピックか子の課題 -->
          <div
            class="sticky left-0 z-20 shrink-0 flex items-center gap-2 pr-2 bg-surface border-r border-line text-sm"
            :style="{ width: `${LEFT_WIDTH}px`, paddingLeft: row.type === 'epic' ? '8px' : '36px' }"
          >
            <template v-if="row.type === 'epic'">
              <button
                type="button"
                class="shrink-0 w-6 h-6 flex items-center justify-center p-0 border-0 rounded bg-transparent text-muted cursor-pointer hover:bg-raised hover:text-fg disabled:opacity-30 disabled:cursor-default"
                :aria-label="expanded.has(row.epic.ID) ? `${row.epic.Title} の子の課題を隠す` : `${row.epic.Title} の子の課題を表示`"
                :aria-expanded="expanded.has(row.epic.ID)"
                :disabled="!row.children.length"
                @click="toggle(row.epic.ID)"
              >
                <svg :class="['w-3.5 h-3.5 transition-transform', expanded.has(row.epic.ID) ? 'rotate-90' : '']" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
                </svg>
              </button>
              <span class="shrink-0 w-4 h-4 rounded" :style="{ backgroundColor: epicColor(row.epic) }" aria-hidden="true" />
              <div class="min-w-0 flex-1 flex flex-col gap-1">
                <span class="truncate font-semibold text-fg" :title="row.epic.Title">{{ row.epic.Title }}</span>
                <!-- 進捗（完了した子の課題の割合） -->
                <span
                  v-if="row.progress.total"
                  class="block h-1 rounded-full bg-raised overflow-hidden"
                  role="img"
                  :aria-label="`完了 ${row.progress.done} / ${row.progress.total} 件`"
                  :title="`完了 ${row.progress.done} / ${row.progress.total} 件（${row.progress.percent}%）`"
                >
                  <span class="block h-full bg-success" :style="{ width: `${row.progress.percent}%` }" />
                </span>
              </div>
              <span class="shrink-0 text-xs text-muted" :title="`子の課題 ${row.children.length} 件`">{{ row.progress.total ? `${row.progress.percent}%` : '' }}</span>
              <button
                type="button"
                class="shrink-0 w-6 h-6 flex items-center justify-center p-0 border-0 rounded bg-transparent text-muted cursor-pointer hover:bg-raised hover:text-fg"
                :title="`${row.epic.Title} の期間と色`"
                :aria-label="`${row.epic.Title} の期間と色を変更`"
                :aria-expanded="editing?.ID === row.epic.ID"
                data-epic-editor-toggle
                @click="openEditor(row.epic, $event)"
              >
                <svg class="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path d="M3 10a1.5 1.5 0 113 0 1.5 1.5 0 01-3 0zM8.5 10a1.5 1.5 0 113 0 1.5 1.5 0 01-3 0zM15.5 8.5a1.5 1.5 0 100 3 1.5 1.5 0 000-3z" />
                </svg>
              </button>
            </template>
            <template v-else>
              <IssueTypeIcon :type="row.todo.IssueType" />
              <NuxtLink :to="issuePath(row.todo.Key)" class="shrink-0 font-mono text-xs text-muted no-underline hover:text-accent hover:underline">{{ row.todo.Key }}</NuxtLink>
              <span :class="['min-w-0 flex-1 truncate', row.todo.StatusCategory === 'Done' ? 'text-muted line-through' : 'text-fg']" :title="row.todo.Title">{{ row.todo.Title }}</span>
              <StatusDot :category="row.todo.StatusCategory" :title="row.todo.StatusName" />
            </template>
          </div>

          <!-- 右: 日付軸 -->
          <div
            class="relative shrink-0"
            :style="{ width: `${chartWidth}px` }"
            @pointermove="row.type === 'epic' && !row.span ? handleGhostMove($event, row.epic) : undefined"
            @pointerleave="ghost = null"
            @click="row.type === 'epic' && !row.span ? handleSetSpan($event, row.epic) : undefined"
          >
            <!-- エピックのバー。全体をドラッグで移動、両端のつまみで期間を変える -->
            <div
              v-if="row.type === 'epic' && row.span"
              :class="[
                'group absolute top-1.5 bottom-1.5 rounded-md shadow-sm select-none touch-none',
                drag?.epicId === row.epic.ID ? 'cursor-grabbing ring-2 ring-accent' : 'cursor-grab'
              ]"
              :style="barStyle(row.span, epicColor(row.epic))"
              :title="`${row.epic.Title}: ${spanLabel(row.span)}${row.progress.total ? `（完了 ${row.progress.done} / ${row.progress.total} 件）` : ''}`"
              role="group"
              :aria-label="`${row.epic.Title} の期間 ${spanLabel(row.span)}`"
              @pointerdown="startDrag($event, row.epic, 'move')"
              @pointermove="moveDrag"
              @pointerup="endDrag"
              @pointercancel="drag = null"
            >
              <span
                class="absolute inset-y-0 left-0 w-2 rounded-l-md cursor-ew-resize bg-black/0 group-hover:bg-black/20"
                aria-hidden="true"
                @pointerdown="startDrag($event, row.epic, 'start')"
              />
              <span
                v-if="barWidth(row.span) >= LABEL_MIN_WIDTH"
                class="absolute inset-y-0 left-2.5 right-2.5 flex items-center text-xs font-semibold truncate pointer-events-none"
                :style="{ color: readableTextOn(epicColor(row.epic)) }"
              >
                <span class="truncate">{{ row.epic.Title }}</span>
              </span>
              <!-- 進捗。バーの下端に完了の割合を出す -->
              <span v-if="row.progress.total" class="absolute left-1 right-1 bottom-0.5 h-1 rounded-full bg-white/40 overflow-hidden pointer-events-none" aria-hidden="true">
                <span class="block h-full bg-white" :style="{ width: `${row.progress.percent}%` }" />
              </span>
              <span
                class="absolute inset-y-0 right-0 w-2 rounded-r-md cursor-ew-resize bg-black/0 group-hover:bg-black/20"
                aria-hidden="true"
                @pointerdown="startDrag($event, row.epic, 'end')"
              />
              <!-- 狭いバーの名前はバーの右に出す -->
              <span
                v-if="barWidth(row.span) < LABEL_MIN_WIDTH && drag?.epicId !== row.epic.ID"
                class="absolute left-full top-1/2 -translate-y-1/2 ml-1.5 text-xs font-semibold text-fg whitespace-nowrap pointer-events-none"
              >
                {{ row.epic.Title }}
              </span>
              <!-- ドラッグ中は期間をバーの右に出す -->
              <span
                v-if="drag?.epicId === row.epic.ID"
                class="absolute left-full top-1/2 -translate-y-1/2 ml-2 px-1.5 py-0.5 rounded bg-fg text-canvas text-[11px] whitespace-nowrap pointer-events-none"
              >
                {{ spanLabel(row.span) }}
              </span>
            </div>

            <!-- 期間のないエピック: ポインターの位置に、クリックで作るバーを薄く出す -->
            <div
              v-if="row.type === 'epic' && !row.span && ghost?.epicId === row.epic.ID"
              class="absolute top-1.5 bottom-1.5 rounded-md border-2 border-dashed flex items-center px-2 text-[11px] text-muted whitespace-nowrap overflow-hidden pointer-events-none"
              :style="{ ...barStyle(ghost.span, 'transparent'), borderColor: epicColor(row.epic) }"
            >
              クリックで期間を設定
            </div>

            <!-- 子の課題: スプリントに入っていればスプリントの期間、なければ期限日の印 -->
            <template v-if="row.type === 'todo'">
              <div
                v-if="row.sprintSpan"
                class="absolute top-3 bottom-3 rounded opacity-70"
                :style="barStyle(row.sprintSpan, row.color)"
                :title="`${row.todo.Key}: ${row.sprintName}（${spanLabel(row.sprintSpan)}）`"
              />
              <div
                v-if="row.todo.Deadline && inRange(row.todo.Deadline)"
                class="absolute top-1/2 w-2.5 h-2.5 -mt-[5px] -ml-[5px] rotate-45 border-2 border-surface"
                :class="row.todo.StatusCategory !== 'Done' && row.todo.Deadline < today ? 'bg-danger' : 'bg-fg'"
                :style="{ left: `${offsetOf(row.todo.Deadline) + scale.dayWidth / 2}px` }"
                :title="`${row.todo.Key} の期限日: ${spanLabel({ start: row.todo.Deadline, end: row.todo.Deadline })}`"
              />
            </template>
          </div>
        </div>
      </template>

      <!-- エピックの作成 -->
      <div class="flex" :style="{ minHeight: `${ROW_HEIGHT}px` }">
        <form
          class="sticky left-0 z-20 shrink-0 flex items-center gap-2 px-2 bg-surface border-r border-line"
          :style="{ width: `${LEFT_WIDTH}px` }"
          @submit.prevent="submitCreate"
        >
          <input
            v-model="newTitle"
            type="text"
            maxlength="255"
            placeholder="+ エピックを作成"
            aria-label="新しいエピックの名前"
            class="min-w-0 flex-1 px-2 py-1 border border-transparent rounded-md bg-transparent text-sm text-fg placeholder:text-muted hover:border-line focus:outline-none focus:border-accent focus:bg-surface"
          />
          <button
            v-if="newTitle.trim()"
            type="submit"
            class="shrink-0 px-2.5 py-1 border-0 rounded-md bg-accent text-on-accent text-xs font-semibold cursor-pointer hover:bg-accent-hover"
          >
            作成
          </button>
        </form>
        <div class="shrink-0" :style="{ width: `${chartWidth}px` }" />
      </div>
    </div>
  </div>

  <!-- エピックの期間と色の編集（ドラッグの代わりに日付で入力・未設定に戻す） -->
  <div
    v-if="editing"
    ref="editorRoot"
    class="fixed z-50 w-72 p-4 bg-surface border border-line rounded-xl shadow-xl flex flex-col gap-3 text-sm text-fg"
    :style="{ top: `${editorPosition.top}px`, left: `${editorPosition.left}px` }"
    role="dialog"
    :aria-label="`${editing.Title} の期間と色`"
  >
    <h3 class="m-0 text-sm font-bold truncate">{{ editing.Title }}</h3>
    <div class="grid grid-cols-2 gap-2">
      <label class="flex flex-col gap-1 text-xs font-semibold text-muted">
        開始日
        <input
          type="date"
          :value="editing.StartDate"
          :max="editing.DueDate || undefined"
          class="px-2 py-1 border border-line rounded-md bg-surface text-sm text-fg focus:outline-none focus:border-accent"
          @change="changeDate('StartDate', $event.target.value)"
        />
      </label>
      <label class="flex flex-col gap-1 text-xs font-semibold text-muted">
        期限日
        <input
          type="date"
          :value="editing.DueDate"
          :min="editing.StartDate || undefined"
          class="px-2 py-1 border border-line rounded-md bg-surface text-sm text-fg focus:outline-none focus:border-accent"
          @change="changeDate('DueDate', $event.target.value)"
        />
      </label>
    </div>
    <button
      v-if="editing.StartDate || editing.DueDate"
      type="button"
      class="self-start p-0 border-0 bg-transparent text-xs text-muted cursor-pointer hover:text-danger"
      @click="emit('change-span', editing, { start: '', end: '' })"
    >
      期間を未設定に戻す
    </button>
    <div class="flex flex-col gap-1.5">
      <span class="text-xs font-semibold text-muted">色</span>
      <div class="flex flex-wrap gap-1.5" role="radiogroup" aria-label="バーの色">
        <button
          v-for="color in EPIC_COLORS"
          :key="color.value"
          type="button"
          role="radio"
          :aria-checked="epicColor(editing) === color.value"
          :aria-label="color.label"
          :title="color.label"
          :class="[
            'w-7 h-7 p-0 rounded-md border-2 cursor-pointer',
            epicColor(editing) === color.value ? 'border-fg' : 'border-transparent'
          ]"
          :style="{ backgroundColor: color.value }"
          @click="emit('change-color', editing, color.value)"
        />
      </div>
    </div>
    <p v-if="editorError" class="m-0 text-xs text-danger">{{ editorError }}</p>
    <button
      type="button"
      class="self-end px-3 py-1 border-0 rounded-md bg-transparent text-sm font-semibold text-fg cursor-pointer hover:bg-raised"
      @click="closeEditor"
    >
      閉じる
    </button>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue';
import IssueTypeIcon from './IssueTypeIcon.vue';
import StatusDot from './StatusDot.vue';
import { issuePath } from '../utils/spaceRoutes';
import { readableTextOn } from '../utils/spaceColors';
import {
  timelineScale, timelineRange, timelineHeaders, epicSpan, epicProgress, draggedSpan, diffDays, addDays,
  isWeekend, spanLabel, todayText, epicColor, EPIC_COLORS, NEW_BAR_DAYS
} from '../utils/timeline';

const props = defineProps({
  // 表示するエピック（並び順のまま出す）
  epics: {
    type: Array,
    required: true
  },
  // エピックの子の課題を探す課題（サブタスクは除いて出す）
  todos: {
    type: Array,
    required: true
  },
  // 子の課題のバーに使うスプリント
  sprints: {
    type: Array,
    default: () => []
  },
  // week・month・quarter
  scaleValue: {
    type: String,
    required: true
  },
  maxHeight: {
    type: String,
    default: 'calc(100vh - 8rem)'
  }
});

// change-span: エピックの期間を変えた（{ start, end }。空文字は未設定）、change-color: 色を変えた、create: エピックを作る（名前）
const emit = defineEmits(['change-span', 'change-color', 'create']);

const LEFT_WIDTH = 300;
const HEADER_HEIGHT = 48;
const ROW_HEIGHT = 44;
// これより狭いバーは、名前をバーの外に出す
const LABEL_MIN_WIDTH = 72;

const today = ref(todayText());
const scale = computed(() => timelineScale(props.scaleValue));

// 展開しているエピック
const expanded = ref(new Set());
const toggle = (epicId) => {
  const next = new Set(expanded.value);
  if (next.has(epicId)) next.delete(epicId);
  else next.add(epicId);
  expanded.value = next;
};

// ドラッグ中のバー { epicId, mode, startX, span, days }
const drag = ref(null);

const spanOf = (epic) => {
  const span = epicSpan(epic);
  if (drag.value?.epicId === epic.ID && span) {
    return draggedSpan(drag.value.span, drag.value.mode, drag.value.days);
  }
  return span;
};

const sprintById = computed(() => new Map(props.sprints.map(sprint => [sprint.ID, sprint])));

const rows = computed(() => {
  const list = [];
  for (const epic of props.epics) {
    const children = props.todos.filter(todo => todo.EpicID === epic.ID && !todo.ParentID);
    list.push({ type: 'epic', key: `epic-${epic.ID}`, epic, children, span: spanOf(epic), progress: epicProgress(children) });
    if (!expanded.value.has(epic.ID)) continue;
    for (const todo of children) {
      const sprint = todo.SprintID ? sprintById.value.get(todo.SprintID) : null;
      const sprintSpan = sprint?.StartDate && sprint?.EndDate ? { start: sprint.StartDate, end: sprint.EndDate } : null;
      list.push({ type: 'todo', key: `todo-${todo.ID}`, todo, sprintSpan, sprintName: sprint?.Name ?? '', color: epicColor(epic) });
    }
  }
  return list;
});

// 表示の範囲はドラッグ前の期間で決める（ドラッグ中に範囲が変わってバーがずれないように）
const range = computed(() => {
  const spans = [];
  for (const epic of props.epics) {
    const span = epicSpan(epic);
    if (span) spans.push(span);
  }
  return timelineRange(spans, scale.value, today.value);
});
const headers = computed(() => timelineHeaders(range.value, scale.value.value));
const chartWidth = computed(() => (diffDays(range.value.start, range.value.end) + 1) * scale.value.dayWidth);
const weekendSegments = computed(() => headers.value.bottom.filter(segment => isWeekend(segment.start)));

const offsetOf = (date) => diffDays(range.value.start, date) * scale.value.dayWidth;
const inRange = (date) => date >= range.value.start && date <= range.value.end;
const todayInRange = computed(() => inRange(today.value));
const todayX = computed(() => offsetOf(today.value) + scale.value.dayWidth / 2);

const segmentStyle = (segment) => ({
  left: `${offsetOf(segment.start)}px`,
  width: `${segment.days * scale.value.dayWidth}px`
});
const barWidth = (span) => Math.max((diffDays(span.start, span.end) + 1) * scale.value.dayWidth, 6);
const barStyle = (span, color) => ({
  left: `${offsetOf(span.start)}px`,
  width: `${barWidth(span)}px`,
  backgroundColor: color
});

// バーのドラッグ。ポインターを掴んだまま動かした日数だけ期間を動かし、離したときに保存する
const startDrag = (event, epic, mode) => {
  if (event.button !== 0) return;
  const span = epicSpan(epic);
  if (!span) return;
  event.preventDefault();
  event.stopPropagation();
  event.currentTarget.setPointerCapture(event.pointerId);
  drag.value = { epicId: epic.ID, epic, mode, startX: event.clientX, span, days: 0 };
};
const moveDrag = (event) => {
  if (!drag.value) return;
  drag.value.days = Math.round((event.clientX - drag.value.startX) / scale.value.dayWidth);
};
const endDrag = () => {
  const current = drag.value;
  if (!current) return;
  drag.value = null;
  if (current.days === 0) return;
  const span = draggedSpan(current.span, current.mode, current.days);
  emit('change-span', current.epic, span);
};

// 期間のないエピックの行: クリックした日から、単位ごとの日数のバーを作る
const ghost = ref(null);
const dateAt = (event) => {
  const rect = event.currentTarget.getBoundingClientRect();
  const days = Math.floor((event.clientX - rect.left) / scale.value.dayWidth);
  return addDays(range.value.start, days);
};
const newSpanAt = (event) => {
  const start = dateAt(event);
  return { start, end: addDays(start, NEW_BAR_DAYS[scale.value.value] - 1) };
};
const handleGhostMove = (event, epic) => {
  ghost.value = { epicId: epic.ID, span: newSpanAt(event) };
};
const handleSetSpan = (event, epic) => {
  ghost.value = null;
  emit('change-span', epic, newSpanAt(event));
};

// 期間と色の編集
const editingId = ref(null);
const editing = computed(() => props.epics.find(epic => epic.ID === editingId.value) ?? null);
const editorRoot = ref(null);
const editorPosition = ref({ top: 0, left: 0 });
const editorError = ref('');
const openEditor = (epic, event) => {
  if (editingId.value === epic.ID) {
    closeEditor();
    return;
  }
  const rect = event.currentTarget.getBoundingClientRect();
  editorPosition.value = {
    top: Math.min(rect.bottom + 4, window.innerHeight - 320),
    left: Math.min(rect.left, window.innerWidth - 300)
  };
  editorError.value = '';
  editingId.value = epic.ID;
};
const closeEditor = () => {
  editingId.value = null;
};
// 日付の入力。もう片方より前後が逆になる日付は保存しない
const changeDate = (field, value) => {
  const epic = editing.value;
  const start = field === 'StartDate' ? value : epic.StartDate;
  const end = field === 'DueDate' ? value : epic.DueDate;
  if (start && end && end < start) {
    editorError.value = '期限日は開始日以降にしてください';
    return;
  }
  editorError.value = '';
  emit('change-span', epic, { start, end });
};
const closeOnOutside = (event) => {
  if (!editing.value || !editorRoot.value || editorRoot.value.contains(event.target)) return;
  // 開くボタンは自分で開閉する
  if (event.target.closest?.('[data-epic-editor-toggle]')) return;
  closeEditor();
};
const closeOnEscape = (event) => {
  if (event.key === 'Escape') closeEditor();
};

// エピックの作成
const newTitle = ref('');
const submitCreate = () => {
  const title = newTitle.value.trim();
  if (!title) return;
  emit('create', title);
  newTitle.value = '';
};

// 今日がスクロールの左から 3 分の 1 あたりに来るようにする
const scroller = ref(null);
const scrollToToday = () => {
  const el = scroller.value;
  if (!el) return;
  el.scrollLeft = Math.max(todayX.value - (el.clientWidth - LEFT_WIDTH) / 3, 0);
};
watch(() => props.scaleValue, () => nextTick(scrollToToday));

// 日付が変わったら今日の線を動かす
let todayTimer = null;
onMounted(() => {
  nextTick(scrollToToday);
  todayTimer = setInterval(() => {
    today.value = todayText();
  }, 60 * 1000);
  document.addEventListener('pointerdown', closeOnOutside);
  document.addEventListener('keydown', closeOnEscape);
});
onBeforeUnmount(() => {
  clearInterval(todayTimer);
  document.removeEventListener('pointerdown', closeOnOutside);
  document.removeEventListener('keydown', closeOnEscape);
});

defineExpose({ scrollToToday });
</script>
