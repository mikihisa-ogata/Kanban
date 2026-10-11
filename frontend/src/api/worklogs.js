const API_BASE_URL = 'http://localhost:8080';

// 失敗したときは bff のメッセージ（作業時間の形式が正しくない、など）をエラーにする
const throwResponseError = async (response, fallback) => {
  const body = await response.json().catch(() => ({}));
  throw new Error(body.error || fallback);
};

export const worklogsApi = {
  // 課題の作業ログ一覧取得（開始日時の古い順）
  async fetchByTodo(todoId) {
    const response = await fetch(`${API_BASE_URL}/todos/${todoId}/worklogs`);
    if (!response.ok) {
      await throwResponseError(response, 'Failed to fetch worklogs');
    }
    return response.json();
  },

  // 作業を記録（worklog は { timeSpent, startedAt, note, remainingEstimate? }。remainingEstimate を省くと残り時間を自動で減らす）
  async create(todoId, worklog) {
    const response = await fetch(`${API_BASE_URL}/todos/${todoId}/worklogs`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(worklog)
    });
    if (!response.ok) {
      await throwResponseError(response, 'Failed to create worklog');
    }
    return response.json();
  },

  // 作業ログ更新（形は create と同じ）
  async update(id, worklog) {
    const response = await fetch(`${API_BASE_URL}/worklogs/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(worklog)
    });
    if (!response.ok) {
      await throwResponseError(response, 'Failed to update worklog');
    }
    return response.json();
  },

  // 作業ログ削除（残り時間はその分だけ戻る）
  async delete(id) {
    const response = await fetch(`${API_BASE_URL}/worklogs/${id}`, {
      method: 'DELETE'
    });
    if (!response.ok) {
      await throwResponseError(response, 'Failed to delete worklog');
    }
    return response.json();
  }
};
