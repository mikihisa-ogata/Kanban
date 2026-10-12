import { API_BASE_URL, apiFetch } from '~/api/client';

// 失敗したときは bff のメッセージ（本文が空、など）をエラーにする
const throwResponseError = async (response, fallback) => {
  const body = await response.json().catch(() => ({}));
  throw new Error(body.error || fallback);
};

export const commentsApi = {
  // 課題のコメント一覧取得（投稿順＝古い順）
  async fetchByTodo(todoId) {
    const response = await apiFetch(`${API_BASE_URL}/todos/${todoId}/comments`);
    if (!response.ok) {
      await throwResponseError(response, 'Failed to fetch comments');
    }
    return response.json();
  },

  // コメント投稿（作成したコメントを comment に入れて返す）
  async create(todoId, body) {
    const response = await apiFetch(`${API_BASE_URL}/todos/${todoId}/comments`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ body })
    });
    if (!response.ok) {
      await throwResponseError(response, 'Failed to create comment');
    }
    return response.json();
  },

  // コメント編集（編集後のコメントを comment に入れて返す）
  async update(id, body) {
    const response = await apiFetch(`${API_BASE_URL}/comments/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ body })
    });
    if (!response.ok) {
      await throwResponseError(response, 'Failed to update comment');
    }
    return response.json();
  },

  // コメント削除
  async delete(id) {
    const response = await apiFetch(`${API_BASE_URL}/comments/${id}`, {
      method: 'DELETE'
    });
    if (!response.ok) {
      await throwResponseError(response, 'Failed to delete comment');
    }
    return response.json();
  }
};
