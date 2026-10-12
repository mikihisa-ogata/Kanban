// 課題一覧（課題ナビゲーター）の条件。条件は URL のクエリに持ち、リロード・共有しても同じ一覧を出す。
// 同じ種類の中では「いずれか」、種類の間では「すべて」を満たす課題を出す（ボードのクイックフィルターと同じ）

// 担当者の「未割り当て」を表す値
export const UNASSIGNED = 0;

// 1 ページの件数（bff の既定と同じ）
export const PER_PAGE = 50;

// 表の列。sort は bff の並べ替えの列（domain.IssueSortFields）
export const ISSUE_COLUMNS = [
  { id: 'type', label: 'タイプ', sort: 'type' },
  { id: 'key', label: 'キー', sort: 'key' },
  { id: 'summary', label: '要約', sort: 'summary' },
  { id: 'status', label: 'ステータス', sort: 'status' },
  { id: 'assignee', label: '担当者', sort: 'assignee' },
  { id: 'reporter', label: '報告者', sort: 'reporter' },
  { id: 'priority', label: '優先度', sort: 'priority' },
  { id: 'created', label: '作成日', sort: 'created' },
  { id: 'updated', label: '更新日', sort: 'updated' },
  { id: 'due', label: '期限', sort: 'due' }
];

export const DEFAULT_COLUMNS = ISSUE_COLUMNS.map(column => column.id);

// 並べ替えの既定（Jira と同じく作成日の新しい順）
export const DEFAULT_SORT = 'created';
export const DEFAULT_ORDER = 'desc';

const splitList = (value) => (typeof value === 'string' && value !== '' ? value.split(',') : []);
const positiveInt = (value) => {
  const n = Number(value);
  return Number.isInteger(n) && n > 0 ? n : 0;
};

// URL のクエリから条件を読む
export const parseIssueSearch = (query) => {
  const sort = ISSUE_COLUMNS.some(column => column.sort === query.sort) ? query.sort : DEFAULT_SORT;
  return {
    spaces: splitList(query.space),
    types: splitList(query.type),
    statuses: splitList(query.status),
    assignees: splitList(query.assignee).map(Number).filter(n => Number.isInteger(n) && n >= 0),
    labels: splitList(query.label),
    text: typeof query.text === 'string' ? query.text : '',
    sort,
    order: query.order === 'asc' || query.order === 'desc' ? query.order : (query.sort ? 'asc' : DEFAULT_ORDER),
    page: positiveInt(query.page) || 1
  };
};

// 条件を URL のクエリにする。既定の値は省く（ほかのクエリ、例えば selectedIssue は残す）
export const withIssueSearch = (query, search) => {
  const next = { ...query };
  for (const key of ['space', 'type', 'status', 'assignee', 'label', 'text', 'sort', 'order', 'page']) {
    delete next[key];
  }
  if (search.spaces.length) next.space = search.spaces.join(',');
  if (search.types.length) next.type = search.types.join(',');
  if (search.statuses.length) next.status = search.statuses.join(',');
  if (search.assignees.length) next.assignee = search.assignees.join(',');
  if (search.labels.length) next.label = search.labels.join(',');
  if (search.text.trim()) next.text = search.text;
  if (search.sort !== DEFAULT_SORT || search.order !== DEFAULT_ORDER) {
    next.sort = search.sort;
    next.order = search.order;
  }
  if (search.page > 1) next.page = String(search.page);
  return next;
};

// 条件を bff の GET /issues/search のパラメーターにする
export const issueSearchParams = (search) => {
  const params = new URLSearchParams();
  search.spaces.forEach(v => params.append('space', v));
  search.types.forEach(v => params.append('type', v));
  search.statuses.forEach(v => params.append('status', v));
  search.assignees.forEach(v => params.append('assignee', v === UNASSIGNED ? 'unassigned' : String(v)));
  search.labels.forEach(v => params.append('label', v));
  if (search.text.trim()) params.set('text', search.text.trim());
  params.set('sort', search.sort);
  params.set('order', search.order);
  params.set('page', String(search.page));
  params.set('perPage', String(PER_PAGE));
  return params;
};

export const isSearching = (search) =>
  search.spaces.length > 0 || search.types.length > 0 || search.statuses.length > 0 ||
  search.assignees.length > 0 || search.labels.length > 0 || !!search.text.trim();

// 表示する列はブラウザに覚える（使えない環境では既定の列）
const COLUMNS_STORAGE_KEY = 'issueNavigatorColumns';

export const loadColumns = () => {
  try {
    const saved = JSON.parse(localStorage.getItem(COLUMNS_STORAGE_KEY));
    if (Array.isArray(saved)) {
      const columns = DEFAULT_COLUMNS.filter(id => saved.includes(id));
      if (columns.length) return columns;
    }
  } catch {
    // 読めなければ既定の列
  }
  return [...DEFAULT_COLUMNS];
};

export const saveColumns = (columns) => {
  try {
    localStorage.setItem(COLUMNS_STORAGE_KEY, JSON.stringify(columns));
  } catch {
    // 保存できなくても表示はそのまま
  }
};
