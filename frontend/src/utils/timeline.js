// タイムライン（Jira のロードマップ）の日付の計算。日付は YYYY-MM-DD の文字列で扱い、ローカルの日付として読む

const DAY_MS = 24 * 60 * 60 * 1000;

// 日付（YYYY-MM-DD）をローカルの 0 時の Date にする
export const parseDate = (text) => {
  const [y, m, d] = text.split('-').map(Number);
  return new Date(y, m - 1, d);
};

export const formatDate = (date) => {
  const pad = (n) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
};

export const addDays = (text, days) => {
  const date = parseDate(text);
  return formatDate(new Date(date.getFullYear(), date.getMonth(), date.getDate() + days));
};

// from から to までの日数（同じ日は 0）。夏時間の切り替えをまたいでも整数になるよう丸める
export const diffDays = (from, to) => Math.round((parseDate(to) - parseDate(from)) / DAY_MS);

export const todayText = (now = new Date()) => formatDate(now);

// 表示の単位。dayWidth は 1 日の幅（px）。before・after は今日の前後に最低限出す日数
export const TIMELINE_SCALES = [
  { value: 'week', label: '週', dayWidth: 32, before: 14, after: 70 },
  { value: 'month', label: '月', dayWidth: 8, before: 31, after: 200 },
  { value: 'quarter', label: '四半期', dayWidth: 3, before: 92, after: 550 }
];

export const DEFAULT_TIMELINE_SCALE = 'month';

export const timelineScale = (value) => TIMELINE_SCALES.find(scale => scale.value === value) ?? TIMELINE_SCALES[1];

// 期間を設定していないエピックの行をクリックしたときに作るバーの日数（クリックした日を含む）
export const NEW_BAR_DAYS = { week: 7, month: 14, quarter: 30 };

// エピックのバーの期間 { start, end }（どちらも含む）。片方だけなら 1 日、どちらもなければ null
export const epicSpan = (epic) => {
  const start = epic.StartDate || epic.DueDate;
  const end = epic.DueDate || epic.StartDate;
  if (!start) return null;
  return { start, end };
};

// 表示する範囲 { start, end }。今日の前後（単位ごとの日数）と、すべての期間が入るようにし、
// 週の表示は月曜、月・四半期の表示は月初から始め、月末で終える
export const timelineRange = (spans, scale, today) => {
  let start = addDays(today, -scale.before);
  let end = addDays(today, scale.after);
  for (const span of spans) {
    if (span.start < start) start = span.start;
    if (span.end > end) end = span.end;
  }
  // 端のバーを掴めるよう、前後に少し余白を取る
  start = addDays(start, -7);
  end = addDays(end, 14);

  const first = parseDate(start);
  const last = parseDate(end);
  if (scale.value === 'week') {
    const weekday = (first.getDay() + 6) % 7; // 月曜を 0 にする
    start = formatDate(new Date(first.getFullYear(), first.getMonth(), first.getDate() - weekday));
  } else if (scale.value === 'quarter') {
    start = formatDate(new Date(first.getFullYear(), Math.floor(first.getMonth() / 3) * 3, 1));
  } else {
    start = formatDate(new Date(first.getFullYear(), first.getMonth(), 1));
  }
  end = formatDate(new Date(last.getFullYear(), last.getMonth() + 1, 0));
  return { start, end };
};

// range の中で、日付が単位の区切り（unit）ごとに変わる区間を返す。label(date) は区間の先頭の日の見出し
const segments = (range, nextStart, label) => {
  const result = [];
  let cursor = parseDate(range.start);
  const last = parseDate(range.end);
  while (cursor <= last) {
    let next = nextStart(cursor);
    if (next > last) next = new Date(last.getFullYear(), last.getMonth(), last.getDate() + 1);
    const start = formatDate(cursor);
    result.push({ start, days: diffDays(start, formatDate(next)), label: label(cursor) });
    cursor = next;
  }
  return result;
};

const nextMonth = (date) => new Date(date.getFullYear(), date.getMonth() + 1, 1);
const nextQuarter = (date) => new Date(date.getFullYear(), Math.floor(date.getMonth() / 3) * 3 + 3, 1);
const nextYear = (date) => new Date(date.getFullYear() + 1, 0, 1);
const nextDay = (date) => new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);

// 見出しの 2 段 { top, bottom }。各区間は { start, days, label }。週は 月/日、月は 年/月、四半期は 年/四半期
export const timelineHeaders = (range, scaleValue) => {
  if (scaleValue === 'week') {
    return {
      top: segments(range, nextMonth, d => `${d.getFullYear()}年${d.getMonth() + 1}月`),
      bottom: segments(range, nextDay, d => String(d.getDate()))
    };
  }
  if (scaleValue === 'quarter') {
    return {
      top: segments(range, nextYear, d => `${d.getFullYear()}年`),
      bottom: segments(range, nextQuarter, d => `Q${Math.floor(d.getMonth() / 3) + 1}`)
    };
  }
  return {
    top: segments(range, nextYear, d => `${d.getFullYear()}年`),
    bottom: segments(range, nextMonth, d => `${d.getMonth() + 1}月`)
  };
};

// 土日か（週の表示で背景を変える）
export const isWeekend = (text) => {
  const day = parseDate(text).getDay();
  return day === 0 || day === 6;
};

// ドラッグの結果の期間。mode は move（全体を動かす）・start（左端）・end（右端）。
// 端を反対側の端より先へ動かしたときは、1 日の期間で止める
export const draggedSpan = (span, mode, days) => {
  if (mode === 'move') return { start: addDays(span.start, days), end: addDays(span.end, days) };
  if (mode === 'start') {
    const start = addDays(span.start, days);
    return { start: start > span.end ? span.end : start, end: span.end };
  }
  const end = addDays(span.end, days);
  return { start: span.start, end: end < span.start ? span.start : end };
};

// エピックの進捗。子の課題（サブタスクを除く）のうち、完了カテゴリの件数と割合
export const epicProgress = (children) => {
  const total = children.length;
  const done = children.filter(todo => todo.StatusCategory === 'Done').length;
  return { done, total, percent: total ? Math.round((done / total) * 100) : 0 };
};

// 期間の表示（例: 2026/10/1 〜 2026/10/31）
export const spanLabel = (span) => {
  const short = (text) => {
    const date = parseDate(text);
    return `${date.getFullYear()}/${date.getMonth() + 1}/${date.getDate()}`;
  };
  return span.start === span.end ? short(span.start) : `${short(span.start)} 〜 ${short(span.end)}`;
};

// エピックのバーの色（Jira のロードマップの色に近いもの）。先頭が既定の色
export const EPIC_COLORS = [
  { value: '#6554C0', label: '紫' },
  { value: '#0052CC', label: '青' },
  { value: '#00B8D9', label: '水色' },
  { value: '#00875A', label: '緑' },
  { value: '#36B37E', label: '明るい緑' },
  { value: '#FF991F', label: 'オレンジ' },
  { value: '#FF5630', label: '赤' },
  { value: '#C1C7D0', label: '灰' }
];

export const epicColor = (epic) => epic.Color || EPIC_COLORS[0].value;
