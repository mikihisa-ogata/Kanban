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
  
  // タスク更新
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
