import { API_BASE_URL, apiFetch } from '~/api/client';

export const epicsApi = {
  // エピック一覧取得
  async fetchAll() {
    const response = await apiFetch(`${API_BASE_URL}/epics`);
    if (!response.ok) {
      throw new Error('Failed to fetch epics');
    }
    return response.json();
  },

  // エピック作成
  async create(epic) {
    const response = await apiFetch(`${API_BASE_URL}/epics`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(epic)
    });
    if (!response.ok) {
      throw new Error('Failed to create epic');
    }
    return response.json();
  },

  // エピック更新（スペースを変えると子タスクも移動する）
  async update(id, epic) {
    const response = await apiFetch(`${API_BASE_URL}/epics/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(epic)
    });
    if (!response.ok) {
      throw new Error('Failed to update epic');
    }
    return response.json();
  },

  // エピック削除
  async delete(id) {
    const response = await apiFetch(`${API_BASE_URL}/epics/${id}`, {
      method: 'DELETE'
    });
    if (!response.ok) {
      throw new Error('Failed to delete epic');
    }
    return response.json();
  }
};
