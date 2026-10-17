import { API_BASE_URL, apiFetch } from '~/api/client';

// 失敗したときは bff のメッセージ（開始中のスプリントがある、など）をエラーにする
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

export const sprintsApi = {
  // すべてのスプリント（スペース・作成順）。State は future・active・closed
  async fetchAll() {
    const response = await apiFetch(`${API_BASE_URL}/sprints`);
    if (!response.ok) {
      throw new Error('Failed to fetch sprints');
    }
    return response.json();
  },

  // スペースに未開始のスプリントを作る（名前を省略すると「<キー> スプリント <番号>」）
  async create(spaceId, name = '') {
    return send(`${API_BASE_URL}/spaces/${spaceId}/sprints`, 'POST', { name }, 'Failed to create sprint');
  },

  // 名前・ゴール・期間（YYYY-MM-DD）を変える
  async update(id, { name, goal, startDate, endDate }) {
    return send(`${API_BASE_URL}/sprints/${id}`, 'PUT', { name, goal, startDate, endDate }, 'Failed to update sprint');
  },

  // 名前・ゴール・期間を決めて開始する（スペースごとに同時に 1 つまで）
  async start(id, { name, goal, startDate, endDate }) {
    return send(`${API_BASE_URL}/sprints/${id}/start`, 'POST', { name, goal, startDate, endDate }, 'Failed to start sprint');
  },

  // 完了する。完了していない課題は moveTo のスプリント（0 はバックログ）へ移す
  async complete(id, moveTo) {
    return send(`${API_BASE_URL}/sprints/${id}/complete`, 'POST', { moveTo }, 'Failed to complete sprint');
  },

  // 未開始のスプリントを削除する（課題はバックログに戻る）
  async delete(id) {
    return send(`${API_BASE_URL}/sprints/${id}`, 'DELETE', undefined, 'Failed to delete sprint');
  },

  // バーンダウン（開始中・完了したスプリントだけ）。日付はブラウザのタイムゾーンで区切る
  async burndown(id) {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone || '';
    const response = await apiFetch(`${API_BASE_URL}/sprints/${id}/burndown?tz=${encodeURIComponent(tz)}`);
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new Error(data.error || 'Failed to fetch burndown');
    }
    return response.json();
  },

  // 課題（とそのサブタスク）をスプリント（0 はバックログ）に入れる
  async setTodoSprint(todoId, sprintId) {
    return send(`${API_BASE_URL}/todos/${todoId}/sprint`, 'PUT', { sprintId }, 'Failed to set sprint');
  }
};
