import { API_BASE_URL, apiFetch } from '~/api/client';

// 失敗したときは bff のメッセージ（同じ名前がある、など）をエラーにする
const throwResponseError = async (response, fallback) => {
  const body = await response.json().catch(() => ({}));
  throw new Error(body.error || fallback);
};

const sendJSON = async (url, method, body, fallback) => {
  const response = await apiFetch(url, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  if (!response.ok) {
    await throwResponseError(response, fallback);
  }
  return response.json();
};

export const statusesApi = {
  // すべてのワークフローのステータスを取得（スペース・並び順の順。SpaceID 0 はスペース未割り当ての課題のワークフロー）
  async fetchAll() {
    const response = await apiFetch(`${API_BASE_URL}/statuses`);
    if (!response.ok) {
      throw new Error('Failed to fetch statuses');
    }
    return response.json();
  },

  // スペースのワークフローの末尾にステータスを追加（category は ToDo・InProgress・Done）
  async create(spaceId, { name, category }) {
    return sendJSON(`${API_BASE_URL}/spaces/${spaceId}/statuses`, 'POST', { name, category }, 'Failed to create status');
  },

  // ステータスの名前を変える
  async rename(id, name) {
    return sendJSON(`${API_BASE_URL}/statuses/${id}`, 'PUT', { name }, 'Failed to rename status');
  },

  // ステータスの列の最大件数（WIP 制限）を変える。null で制限なし
  async setWipLimit(id, wipLimit) {
    return sendJSON(`${API_BASE_URL}/statuses/${id}/wip-limit`, 'PUT', { wipLimit }, 'Failed to set WIP limit');
  },

  // スペースのステータスを statusIds の順に並べ直す（ボードの列の順）
  async reorder(spaceId, statusIds) {
    return sendJSON(`${API_BASE_URL}/spaces/${spaceId}/statuses/order`, 'PUT', { statusIds }, 'Failed to reorder statuses');
  },

  // ステータスを削除。そのステータスの課題は moveTo のステータスへ移す（課題がなければ省略できる）
  async delete(id, moveTo = 0) {
    const query = moveTo ? `?moveTo=${moveTo}` : '';
    const response = await apiFetch(`${API_BASE_URL}/statuses/${id}${query}`, { method: 'DELETE' });
    if (!response.ok) {
      await throwResponseError(response, 'Failed to delete status');
    }
    return response.json();
  }
};
