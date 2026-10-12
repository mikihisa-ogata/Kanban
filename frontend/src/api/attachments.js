import { API_BASE_URL, apiFetch } from '~/api/client';

// 1 ファイルの上限（bff の domain.AttachmentMaxSize と同じ 10MB）
export const ATTACHMENT_MAX_SIZE = 10 * 1024 * 1024;

export const attachmentsApi = {
  // 課題の添付ファイル一覧取得（添付順）
  async fetchByTodo(todoId) {
    const response = await apiFetch(`${API_BASE_URL}/todos/${todoId}/attachments`);
    if (!response.ok) {
      throw new Error('Failed to fetch attachments');
    }
    return response.json();
  },

  // ファイルを添付（添付したファイルの情報を attachment に入れて返す）。
  // 失敗したときは bff のメッセージ（上限を超えている、など）をエラーにする
  async upload(todoId, file) {
    const body = new FormData();
    body.append('file', file);
    const response = await apiFetch(`${API_BASE_URL}/todos/${todoId}/attachments`, {
      method: 'POST',
      body
    });
    if (!response.ok) {
      const result = await response.json().catch(() => ({}));
      throw new Error(result.error || 'Failed to upload attachment');
    }
    return response.json();
  },

  // ファイル本体の URL。inline は画像をブラウザで表示する（サムネイル・拡大表示用）
  fileUrl(id, { inline = false } = {}) {
    return `${API_BASE_URL}/attachments/${id}/file${inline ? '?inline=1' : ''}`;
  },

  // 添付ファイル削除
  async delete(id) {
    const response = await apiFetch(`${API_BASE_URL}/attachments/${id}`, {
      method: 'DELETE'
    });
    if (!response.ok) {
      throw new Error('Failed to delete attachment');
    }
    return response.json();
  }
};
