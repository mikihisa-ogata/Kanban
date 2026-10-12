import { authApi } from '~/api/auth';
import { AUTH_PAGES } from '~/api/client';
import { useCurrentUser } from '~/composables/useCurrentUser';
import { safeRedirect } from '~/utils/authRedirect';

// 未ログインならログイン画面に移す。ログイン済みでログイン画面・サインアップ画面を開いたら元のページに戻す。
// セッションの Cookie は localhost 宛てなので Nuxt のサーバーにも届く。サーバー側の描画でもその Cookie で確かめ、
// ブラウザで別のページに移し直さない（描画の食い違いを起こさない）ようにする
export default defineNuxtRouteMiddleware(async (to) => {
  const { user, loaded } = useCurrentUser();
  if (!loaded.value) {
    try {
      const cookie = import.meta.server ? useRequestHeaders(['cookie']).cookie || '' : '';
      user.value = await authApi.fetchMe(cookie);
      loaded.value = true;
    } catch (err) {
      // bff に接続できないときは、各ページの読み込みエラーの表示に任せる
      console.error('Failed to fetch current user:', err);
      return;
    }
  }

  const isAuthPage = AUTH_PAGES.includes(to.path);
  if (!user.value && !isAuthPage) {
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } });
  }
  if (user.value && isAuthPage) {
    return navigateTo(safeRedirect(to.query.redirect));
  }
});
