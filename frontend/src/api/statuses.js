const API_BASE_URL = 'http://localhost:8080';

export const statusesApi = {
  // すべてのワークフローのステータスを取得（スペース・並び順の順。SpaceID 0 はスペース未割り当ての課題のワークフロー）
  async fetchAll() {
    const response = await fetch(`${API_BASE_URL}/statuses`);
    if (!response.ok) {
      throw new Error('Failed to fetch statuses');
    }
    return response.json();
  }
};
