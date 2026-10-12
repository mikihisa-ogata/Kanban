// ログイン後に戻るページ。別サイトへ移されないよう、アプリ内のパス（/ で始まり // ではない）だけを使う
export const safeRedirect = (redirect) => {
  if (typeof redirect === 'string' && redirect.startsWith('/') && !redirect.startsWith('//')) {
    return redirect;
  }
  return '/';
};
