import { API_BASE_URL, apiFetch } from '~/api/client';

export const labelsApi = {
  // いずれかのタスクに付いているラベル（名前順）。入力欄の補完候補に使う
  async fetchAll() {
    const response = await apiFetch(`${API_BASE_URL}/labels`);
    if (!response.ok) {
      throw new Error('Failed to fetch labels');
    }
    return response.json();
  }
};
