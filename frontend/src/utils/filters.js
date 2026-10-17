// 保存済みフィルター（Jira のフィルター）。課題一覧は ?filter=<ID か既定のフィルターの id>&jql=… で開く

// 既定のフィルター（Jira と同じ）。保存・削除はできず、「名前を付けて保存」で自分のフィルターにできる
export const DEFAULT_FILTERS = [
  { id: 'my-open', name: '自分の未完了課題', jql: 'assignee = currentUser() AND statusCategory != Done ORDER BY updated DESC' },
  { id: 'reported', name: '報告した課題', jql: 'reporter = currentUser() ORDER BY created DESC' },
  { id: 'recent', name: '最近表示した課題', jql: 'issuekey IN issueHistory() ORDER BY lastViewed DESC' },
  { id: 'all', name: 'すべての課題', jql: 'ORDER BY created DESC' }
];

// フィルターを開く課題一覧のリンク。filter は既定のフィルター { id, jql } か、保存済みのフィルター { ID, JQL }
export const filterLink = (filter) => {
  const id = filter.ID ?? filter.id;
  return { path: '/issues', query: { filter: String(id), jql: filter.JQL ?? filter.jql } };
};

// URL の ?filter= のフィルターを { id, name, jql, saved, favourite } で返す。なければ null
export const findFilter = (id, savedFilters) => {
  if (typeof id !== 'string' || !id) return null;
  const builtin = DEFAULT_FILTERS.find(f => f.id === id);
  if (builtin) return { ...builtin, saved: false, favourite: false };
  const saved = savedFilters.find(f => String(f.ID) === id);
  return saved ? { id, name: saved.Name, jql: saved.JQL, saved: true, favourite: saved.Favourite } : null;
};

// お気に入りのスターのアイコン（24×24 の path）
export const STAR_ICON = 'M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z';
