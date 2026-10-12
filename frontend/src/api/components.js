import { API_BASE_URL, apiFetch } from '~/api/client';

// 失敗したときは bff のメッセージ（同じ名前がある、など）をエラーにする
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

export const componentsApi = {
  // すべてのコンポーネント（スペース・名前順）。DefaultAssignee は既定の担当者の概要（なしは null）
  async fetchAll() {
    const response = await apiFetch(`${API_BASE_URL}/components`);
    if (!response.ok) {
      throw new Error('Failed to fetch components');
    }
    return response.json();
  },

  // スペースにコンポーネントを作る（defaultAssigneeId の 0 は既定の担当者なし）
  async create(spaceId, { name, description, defaultAssigneeId }) {
    return send(`${API_BASE_URL}/spaces/${spaceId}/components`, 'POST', { name, description, defaultAssigneeId }, 'Failed to create component');
  },

  // 名前・説明・既定の担当者を変える
  async update(id, { name, description, defaultAssigneeId }) {
    return send(`${API_BASE_URL}/components/${id}`, 'PUT', { name, description, defaultAssigneeId }, 'Failed to update component');
  },

  // 削除する（課題からも外れる）
  async delete(id) {
    return send(`${API_BASE_URL}/components/${id}`, 'DELETE', undefined, 'Failed to delete component');
  },

  // 課題のコンポーネントを componentIds に置き換える。担当者が空なら既定の担当者が入る（変更後の課題を todo で返す）
  async setTodoComponents(todoId, componentIds) {
    return send(`${API_BASE_URL}/todos/${todoId}/components`, 'PUT', { componentIds }, 'Failed to set components');
  }
};
