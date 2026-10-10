const API_BASE_URL = 'http://localhost:8080';

export const todosApi = {
  // タスク一覧取得
  async fetchAll() {
    const response = await fetch(`${API_BASE_URL}/todos`);
    if (!response.ok) {
      throw new Error('Failed to fetch todos');
    }
    return response.json();
  },
  
  // 課題キー（例: KAN-12）でタスクを取得。存在しない場合は null を返す
  async fetchByKey(key) {
    const response = await fetch(`${API_BASE_URL}/issues/${encodeURIComponent(key)}`);
    if (response.status === 404) {
      return null;
    }
    if (!response.ok) {
      throw new Error('Failed to fetch todo');
    }
    return response.json();
  },

  // タスク作成
  async create(todo) {
    const response = await fetch(`${API_BASE_URL}/todos`, {
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
    const response = await fetch(`${API_BASE_URL}/todos/${id}`, {
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
    const response = await fetch(`${API_BASE_URL}/todos/${id}/move`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ beforeId })
    });
    if (!response.ok) {
      throw new Error('Failed to move todo');
    }
    return response.json();
  },
  
  // タスク削除
  async delete(id) {
    const response = await fetch(`${API_BASE_URL}/todos/${id}`, {
      method: 'DELETE'
    });
    if (!response.ok) {
      throw new Error('Failed to delete todo');
    }
    return response.json();
  }
};
