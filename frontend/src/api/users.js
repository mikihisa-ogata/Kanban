import { API_BASE_URL, apiFetch } from '~/api/client';

export const usersApi = {
  // すべてのユーザー（表示名の順）を取得する。担当者・報告者を選ぶ候補にする
  async fetchAll() {
    const response = await apiFetch(`${API_BASE_URL}/users`);
    if (!response.ok) {
      throw new Error('Failed to fetch users');
    }
    return response.json();
  }
};
