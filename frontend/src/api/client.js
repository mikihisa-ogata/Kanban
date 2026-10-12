export const API_BASE_URL = 'http://localhost:8080';

// ログイン画面とサインアップ画面のパス。ここでは 401 でログイン画面に移さない
export const AUTH_PAGES = ['/login', '/signup'];

// bff へのリクエスト。ログインのセッションの Cookie を送るため、資格情報付きで送る。
// 401（未ログイン・セッション切れ）が返ったら、今のページに戻れるようにしてログイン画面に移す
export async function apiFetch(url, options = {}) {
  const response = await fetch(url, { credentials: 'include', ...options });
  if (response.status === 401 && import.meta.client && !AUTH_PAGES.includes(window.location.pathname)) {
    const redirect = window.location.pathname + window.location.search + window.location.hash;
    window.location.assign(`/login?redirect=${encodeURIComponent(redirect)}`);
  }
  return response;
}
