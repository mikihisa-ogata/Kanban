import { API_BASE_URL, apiFetch } from '~/api/client';

// 失敗したときは bff のメッセージ（JQL の誤りなど）をエラーにする
const getJson = async (url, fallback) => {
  const response = await apiFetch(url);
  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    throw new Error(data.error || fallback);
  }
  return response.json();
};

export const dashboardApi = {
  // ログイン中のユーザーのガジェットの並び [{ ID, Type, FilterID }]。保存していなければ既定の並び
  async fetch() {
    const data = await getJson(`${API_BASE_URL}/dashboard`, 'Failed to fetch dashboard');
    return data.gadgets;
  },

  // ガジェットの並び全体を保存し、保存した並びを返す
  async save(gadgets) {
    const response = await apiFetch(`${API_BASE_URL}/dashboard`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ gadgets: gadgets.map(g => ({ id: g.ID, type: g.Type, filterId: g.FilterID || 0 })) })
    });
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new Error(data.error || 'Failed to save dashboard');
    }
    return (await response.json()).gadgets;
  },

  // すべての課題の最近の変更履歴（新しい順）。各要素は変更履歴に課題の Key・Title を付けたもの
  async fetchActivity(limit) {
    return getJson(`${API_BASE_URL}/dashboard/activity?${new URLSearchParams({ limit: String(limit) })}`, 'Failed to fetch activity');
  },

  // JQL（空はすべての課題）に合う課題のステータスごとの件数 [{ Name, Category, Count }]
  async fetchStatusCounts(jql) {
    return getJson(`${API_BASE_URL}/dashboard/status-counts?${new URLSearchParams({ jql })}`, 'Failed to fetch status counts');
  },

  // JQL に合う課題の、今日までの days 日間の日ごとの作成数と解決数 { Days: [{ Date, Created, Resolved }], Created, Resolved }
  async fetchCreatedVsResolved(jql, days) {
    return getJson(`${API_BASE_URL}/dashboard/created-vs-resolved?${new URLSearchParams({ jql, days: String(days) })}`, 'Failed to fetch created vs resolved');
  }
};
