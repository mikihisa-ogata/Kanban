import { API_BASE_URL, apiFetch } from '~/api/client';

export const spacesApi = {
  // スペース一覧取得
  async fetchAll() {
    const response = await apiFetch(`${API_BASE_URL}/spaces`);
    if (!response.ok) {
      throw new Error('Failed to fetch spaces');
    }
    return response.json();
  },

  // スペース作成（key を省略すると名前から作る）。
  // キーの形式違反や重複のときは、サーバーのメッセージを持つ Error を投げる
  async create(space) {
    const response = await apiFetch(`${API_BASE_URL}/spaces`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(space)
    });
    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error || 'Failed to create space');
    }
    return response.json();
  },

  // スペース名から、既存と重複しないキーの候補を取得
  async suggestKey(title) {
    const response = await apiFetch(`${API_BASE_URL}/spaces/key-suggestion?title=${encodeURIComponent(title)}`);
    if (!response.ok) {
      throw new Error('Failed to suggest space key');
    }
    const body = await response.json();
    return body.key;
  },

  // スペースの名前とキーを変更。キーの形式違反や重複のときは、サーバーのメッセージを持つ Error を投げる
  async update(id, space) {
    const response = await apiFetch(`${API_BASE_URL}/spaces/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(space)
    });
    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error || 'Failed to update space');
    }
    return response.json();
  },

  // スペース削除
  async delete(id) {
    const response = await apiFetch(`${API_BASE_URL}/spaces/${id}`, {
      method: 'DELETE'
    });
    if (!response.ok) {
      throw new Error('Failed to delete space');
    }
    return response.json();
  }
};
