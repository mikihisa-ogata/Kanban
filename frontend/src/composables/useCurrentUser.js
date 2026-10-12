// ログイン中のユーザー（未ログインなら null）と、bff に問い合わせ済みかどうか。ページ間で共有する
export const useCurrentUser = () => ({
  user: useState('currentUser', () => null),
  loaded: useState('currentUserLoaded', () => false)
});
