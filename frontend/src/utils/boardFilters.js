// ボードのクイックフィルター。条件は URL のクエリに持ち、リロードしても残す。
// 同じ種類の中では「いずれか」、種類の間では「すべて」を満たす課題を出す（Jira のクイックフィルターと同じ）

// 「最近更新したもの」とみなす期間（Jira の Recently Updated と同じ 1 日）
export const RECENT_MS = 24 * 60 * 60 * 1000;

// 担当者・エピックの「なし」を表す値
export const NONE = 0;

// URL のクエリの名前
const KEYS = {
  text: 'q',
  assignees: 'assignee',
  mine: 'mine',
  recent: 'recent',
  labels: 'label',
  types: 'type',
  epics: 'epic'
};

const splitList = (value) => (typeof value === 'string' && value !== '' ? value.split(',') : []);
const splitIds = (value) => splitList(value).map(Number).filter(n => Number.isInteger(n) && n >= 0);

// URL のクエリから条件を読む
export const parseFilters = (query) => ({
  text: typeof query[KEYS.text] === 'string' ? query[KEYS.text] : '',
  assignees: splitIds(query[KEYS.assignees]),
  mine: query[KEYS.mine] === '1',
  recent: query[KEYS.recent] === '1',
  labels: splitList(query[KEYS.labels]),
  types: splitList(query[KEYS.types]),
  epics: splitIds(query[KEYS.epics])
});

// 条件を URL のクエリにする。条件のない種類は消す（ほかのクエリ、例えば selectedIssue は残す）
export const withFilters = (query, filters) => {
  const next = { ...query };
  for (const key of Object.values(KEYS)) {
    delete next[key];
  }
  if (filters.text.trim()) next[KEYS.text] = filters.text;
  if (filters.assignees.length) next[KEYS.assignees] = filters.assignees.join(',');
  if (filters.mine) next[KEYS.mine] = '1';
  if (filters.recent) next[KEYS.recent] = '1';
  if (filters.labels.length) next[KEYS.labels] = filters.labels.join(',');
  if (filters.types.length) next[KEYS.types] = filters.types.join(',');
  if (filters.epics.length) next[KEYS.epics] = filters.epics.join(',');
  return next;
};

export const isFiltering = (filters) =>
  !!filters.text.trim() || filters.assignees.length > 0 || filters.mine || filters.recent ||
  filters.labels.length > 0 || filters.types.length > 0 || filters.epics.length > 0;

// 課題が条件を満たすか。currentUserId は「自分の課題のみ」、now は「最近更新したもの」に使う
export const matchesFilters = (todo, filters, { currentUserId = 0, now = Date.now() } = {}) => {
  const text = filters.text.trim().toLowerCase();
  if (text && !todo.Title.toLowerCase().includes(text) && !(todo.Key || '').toLowerCase().includes(text)) {
    return false;
  }
  if (filters.assignees.length && !filters.assignees.includes(todo.AssigneeID || NONE)) {
    return false;
  }
  if (filters.mine && (!currentUserId || todo.AssigneeID !== currentUserId)) {
    return false;
  }
  if (filters.recent && !(now - new Date(todo.UpdatedAt).getTime() <= RECENT_MS)) {
    return false;
  }
  if (filters.labels.length && !(todo.Labels || []).some(label => filters.labels.includes(label))) {
    return false;
  }
  if (filters.types.length && !filters.types.includes(todo.IssueType)) {
    return false;
  }
  if (filters.epics.length && !filters.epics.includes(todo.EpicID || NONE)) {
    return false;
  }
  return true;
};
