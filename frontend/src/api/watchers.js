import { API_BASE_URL, apiFetch } from '~/api/client';

export const watchersApi = {
  // 課題のウォッチャー（表示名の順）を取得する
  async fetchByTodo(todoId) {
    const response = await apiFetch(`${API_BASE_URL}/todos/${todoId}/watchers`);
    if (!response.ok) {
      throw new Error('Failed to fetch watchers');
    }
    return response.json();
  },

  // ログイン中のユーザーが課題をウォッチする（ウォッチャーを返す）
  async watch(todoId) {
    const response = await apiFetch(`${API_BASE_URL}/todos/${todoId}/watch`, { method: 'POST' });
    if (!response.ok) {
      throw new Error('Failed to watch issue');
    }
    return response.json();
  },

  // ログイン中のユーザーが課題のウォッチをやめる（ウォッチャーを返す）
  async unwatch(todoId) {
    const response = await apiFetch(`${API_BASE_URL}/todos/${todoId}/watch`, { method: 'DELETE' });
    if (!response.ok) {
      throw new Error('Failed to unwatch issue');
    }
    return response.json();
  }
};
