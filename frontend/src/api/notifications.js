import { API_BASE_URL, apiFetch } from '~/api/client';

export const notificationsApi = {
  // ログイン中のユーザーへの通知（新しい順に最大 50 件）と未読の数を { Unread, Notifications } で取得する
  async fetchAll() {
    const response = await apiFetch(`${API_BASE_URL}/notifications`);
    if (!response.ok) {
      throw new Error('Failed to fetch notifications');
    }
    return response.json();
  },

  // 通知を既読にする
  async markRead(id) {
    const response = await apiFetch(`${API_BASE_URL}/notifications/${id}/read`, { method: 'POST' });
    if (!response.ok) {
      throw new Error('Failed to mark notification as read');
    }
  },

  // すべての通知を既読にする
  async markAllRead() {
    const response = await apiFetch(`${API_BASE_URL}/notifications/read-all`, { method: 'POST' });
    if (!response.ok) {
      throw new Error('Failed to mark all notifications as read');
    }
  }
};
