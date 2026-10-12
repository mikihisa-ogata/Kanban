import { API_BASE_URL } from '~/api/client';

// 失敗したときは bff のメッセージ（パスワードが正しくない、など）をエラーにする
const throwResponseError = async (response, fallback) => {
  const body = await response.json().catch(() => ({}));
  throw new Error(body.error || fallback);
};

const sendJSON = (path, body) => fetch(`${API_BASE_URL}${path}`, {
  method: 'POST',
  credentials: 'include',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body)
});

export const authApi = {
  // ユーザーを登録してログインする（登録したユーザーを返す）
  async signup(displayName, email, password) {
    const response = await sendJSON('/auth/signup', { displayName, email, password });
    if (!response.ok) {
      await throwResponseError(response, 'Failed to sign up');
    }
    return (await response.json()).user;
  },

  // ログインする（ログインしたユーザーを返す）
  async login(email, password) {
    const response = await sendJSON('/auth/login', { email, password });
    if (!response.ok) {
      await throwResponseError(response, 'Failed to log in');
    }
    return (await response.json()).user;
  },

  // ログアウトする
  async logout() {
    const response = await sendJSON('/auth/logout', {});
    if (!response.ok) {
      await throwResponseError(response, 'Failed to log out');
    }
  },

  // ログイン中のユーザーを返す。ログインしていなければ null を返す。
  // サーバー側の描画では、ブラウザから届いた Cookie を cookie に渡す
  async fetchMe(cookie = '') {
    const response = await fetch(`${API_BASE_URL}/auth/me`, {
      credentials: 'include',
      headers: cookie ? { cookie } : {}
    });
    if (response.status === 401) {
      return null;
    }
    if (!response.ok) {
      await throwResponseError(response, 'Failed to fetch current user');
    }
    return response.json();
  }
};
