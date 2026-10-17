import { API_BASE_URL, apiFetch } from '~/api/client';

// 失敗したときは bff のメッセージ（同じ名前がある、JQL の誤り、など）をエラーにする
const send = async (url, method, body, fallback) => {
  const response = await apiFetch(url, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body)
  });
  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    throw new Error(data.error || fallback);
  }
  return response.json();
};

export const filtersApi = {
  // ログイン中のユーザーの保存済みフィルター（名前順）。{ ID, Name, JQL, Favourite, ... }
  async fetchAll() {
    const response = await apiFetch(`${API_BASE_URL}/filters`);
    if (!response.ok) {
      throw new Error('Failed to fetch filters');
    }
    return response.json();
  },

  // 検索条件（JQL）に名前を付けて保存する（保存したフィルターを filter で返す）
  async create({ name, jql, favourite }) {
    return send(`${API_BASE_URL}/filters`, 'POST', { name, jql, favourite }, 'Failed to create filter');
  },

  // 名前と JQL を変える（上書き保存）
  async update(id, { name, jql }) {
    return send(`${API_BASE_URL}/filters/${id}`, 'PUT', { name, jql }, 'Failed to update filter');
  },

  // お気に入り（スター）を付ける・外す
  async setFavourite(id, favourite) {
    return send(`${API_BASE_URL}/filters/${id}/favourite`, 'PUT', { favourite }, 'Failed to set favourite');
  },

  async delete(id) {
    return send(`${API_BASE_URL}/filters/${id}`, 'DELETE', undefined, 'Failed to delete filter');
  }
};
