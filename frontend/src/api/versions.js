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

export const versionsApi = {
  // すべてのバージョン（スペース・作成順）。Status は unreleased・released・archived
  async fetchAll() {
    const response = await apiFetch(`${API_BASE_URL}/versions`);
    if (!response.ok) {
      throw new Error('Failed to fetch versions');
    }
    return response.json();
  },

  // スペースに未リリースのバージョンを作る（日付は YYYY-MM-DD。空は未設定）
  async create(spaceId, { name, description, startDate, releaseDate }) {
    return send(`${API_BASE_URL}/spaces/${spaceId}/versions`, 'POST', { name, description, startDate, releaseDate }, 'Failed to create version');
  },

  // 名前・説明・日付・状態を変える
  async update(id, { name, description, startDate, releaseDate, status }) {
    return send(`${API_BASE_URL}/versions/${id}`, 'PUT', { name, description, startDate, releaseDate, status }, 'Failed to update version');
  },

  // リリース済みにする。完了していない課題は moveTo のバージョンへ移す（0 は外すだけ）
  async release(id, { releaseDate, moveTo }) {
    return send(`${API_BASE_URL}/versions/${id}/release`, 'POST', { releaseDate, moveTo }, 'Failed to release version');
  },

  // 削除する（課題からも外れる）
  async delete(id) {
    return send(`${API_BASE_URL}/versions/${id}`, 'DELETE', undefined, 'Failed to delete version');
  },

  // 課題の修正バージョンを versionIds に置き換える
  async setTodoVersions(todoId, versionIds) {
    return send(`${API_BASE_URL}/todos/${todoId}/fix-versions`, 'PUT', { versionIds }, 'Failed to set fix versions');
  }
};
