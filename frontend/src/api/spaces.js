const API_BASE_URL = 'http://localhost:8080';

export const spacesApi = {
  // スペース一覧取得
  async fetchAll() {
    const response = await fetch(`${API_BASE_URL}/spaces`);
    if (!response.ok) {
      throw new Error('Failed to fetch spaces');
    }
    return response.json();
  },

  // スペース作成
  async create(space) {
    const response = await fetch(`${API_BASE_URL}/spaces`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(space)
    });
    if (!response.ok) {
      throw new Error('Failed to create space');
    }
    return response.json();
  },

  // スペース削除
  async delete(id) {
    const response = await fetch(`${API_BASE_URL}/spaces/${id}`, {
      method: 'DELETE'
    });
    if (!response.ok) {
      throw new Error('Failed to delete space');
    }
    return response.json();
  }
};
