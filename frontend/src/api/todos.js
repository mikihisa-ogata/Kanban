import { API_BASE_URL, apiFetch } from '~/api/client';

export const todosApi = {
  // タスク一覧取得
  async fetchAll() {
    const response = await apiFetch(`${API_BASE_URL}/todos`);
    if (!response.ok) {
      throw new Error('Failed to fetch todos');
    }
    return response.json();
  },
  
  // 課題キー（例: KAN-12）でタスクを取得。存在しない場合は null を返す
  async fetchByKey(key) {
    const response = await apiFetch(`${API_BASE_URL}/issues/${encodeURIComponent(key)}`);
    if (response.status === 404) {
      return null;
    }
    if (!response.ok) {
      throw new Error('Failed to fetch todo');
    }
    return response.json();
  },

  // 課題一覧（課題ナビゲーター）の 1 ページ分 { Issues, Total, Page, PerPage }。
  // params は URLSearchParams（space・type・status・assignee・label は繰り返して複数指定。text・sort・order・page・perPage）
  async search(params) {
    const response = await apiFetch(`${API_BASE_URL}/issues/search?${params.toString()}`);
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new Error(data.error || 'Failed to search issues');
    }
    return response.json();
  },

  // タスク作成
  async create(todo) {
    const response = await apiFetch(`${API_BASE_URL}/todos`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(todo)
    });
    if (!response.ok) {
      throw new Error('Failed to create todo');
    }
    return response.json();
  },
  
  // タスク更新（更新後のタスクを todo に入れて返す。スペースを変えると課題キーが変わる）
  async update(id, todo) {
    const response = await apiFetch(`${API_BASE_URL}/todos/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(todo)
    });
    if (!response.ok) {
      throw new Error('Failed to update todo');
    }
    return response.json();
  },
  
  // タスクの並び替え（beforeId のタスクの直前へ移動。0 の場合は末尾へ移動）
  async move(id, beforeId) {
    const response = await apiFetch(`${API_BASE_URL}/todos/${id}/move`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ beforeId })
    });
    if (!response.ok) {
      throw new Error('Failed to move todo');
    }
    return response.json();
  },
  
  // 課題の変更履歴取得（記録順＝古い順）
  async fetchHistory(id) {
    const response = await apiFetch(`${API_BASE_URL}/todos/${id}/history`);
    if (!response.ok) {
      throw new Error('Failed to fetch history');
    }
    return response.json();
  },

  // 課題リンクを作成（relation は id の課題から見た名前。例: 'is blocked by'）。
  // 失敗したときは bff のメッセージ（同じリンクが既にある、など）をエラーにする
  async addLink(id, relation, targetId) {
    const response = await apiFetch(`${API_BASE_URL}/todos/${id}/links`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ relation, targetId })
    });
    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error || 'Failed to add link');
    }
    return response.json();
  },

  // 課題リンクを削除（両方の課題から消える）
  async deleteLink(linkId) {
    const response = await apiFetch(`${API_BASE_URL}/links/${linkId}`, {
      method: 'DELETE'
    });
    if (!response.ok) {
      throw new Error('Failed to delete link');
    }
    return response.json();
  },

  // タスク削除
  async delete(id) {
    const response = await apiFetch(`${API_BASE_URL}/todos/${id}`, {
      method: 'DELETE'
    });
    if (!response.ok) {
      throw new Error('Failed to delete todo');
    }
    return response.json();
  }
};
