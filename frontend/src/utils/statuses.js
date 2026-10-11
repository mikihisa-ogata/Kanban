// ワークフローのステータスのカテゴリ（bff の domain.StatusCategory）。ボードの「すべてのスペース」はこの 3 列で表示する
export const STATUS_CATEGORIES = [
  { value: 'ToDo', label: 'To Do' },
  { value: 'InProgress', label: '進行中' },
  { value: 'Done', label: '完了' }
];

export const DEFAULT_STATUS_CATEGORY = 'ToDo';

export const statusCategoryLabel = (category) =>
  STATUS_CATEGORIES.find(c => c.value === category)?.label ?? category;

// カテゴリごとのボードの列見出しの色
export const STATUS_CATEGORY_HEADER_CLASS = {
  ToDo: 'bg-gradient-to-r from-gray-400 to-gray-600',
  InProgress: 'bg-gradient-to-r from-amber-400 to-amber-600',
  Done: 'bg-gradient-to-r from-emerald-500 to-emerald-700'
};

// カテゴリごとのステータスのバッジの色（詳細の選択欄や子課題・リンクの一覧）
export const STATUS_CATEGORY_BADGE_CLASS = {
  ToDo: 'bg-gray-100 text-gray-700',
  InProgress: 'bg-blue-50 text-blue-700',
  Done: 'bg-emerald-50 text-emerald-700'
};

// 完了（Done カテゴリ）の課題か
export const isDoneCategory = (category) => category === 'Done';

// スペース（0 は未割り当て）のステータスを並び順に返す
export const statusesOfSpace = (statuses, spaceId) =>
  statuses.filter(status => status.SpaceID === (spaceId ?? 0)).sort((a, b) => a.Position - b.Position || a.ID - b.ID);

// カテゴリの先頭のステータス。そのカテゴリがなければ先頭のステータス（なければ null）
export const firstStatusInCategory = (spaceStatuses, category) =>
  spaceStatuses.find(status => status.Category === category) ?? spaceStatuses[0] ?? null;

// 選択欄の選択肢（値はステータスの ID）
export const statusOptionsOf = (spaceStatuses) =>
  spaceStatuses.map(status => ({ value: status.ID, label: status.Name }));

// ボードの列に入る課題か。列はスペースのステータス（statusId あり）か、すべてのスペースでのカテゴリ（statusId が null）
export const isInColumn = (todo, column) =>
  column.statusId == null ? todo.StatusCategory === column.category : todo.StatusID === column.statusId;
