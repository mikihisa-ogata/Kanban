const API_BASE_URL = 'http://localhost:8080';

export const labelsApi = {
  // いずれかのタスクに付いているラベル（名前順）。入力欄の補完候補に使う
  async fetchAll() {
    const response = await fetch(`${API_BASE_URL}/labels`);
    if (!response.ok) {
      throw new Error('Failed to fetch labels');
    }
    return response.json();
  }
};
