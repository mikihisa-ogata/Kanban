// スペース単位のページの URL。スペースは課題キーの接頭辞（例: KAN）で表す。
// space が null の場合は「すべてのスペース」のボード
export const boardPath = (space) => (space ? `/spaces/${encodeURIComponent(space.Key)}/board` : '/board');

export const settingsPath = (space) => `/spaces/${encodeURIComponent(space.Key)}/settings`;

// `/` を開いたときに戻る、最後に開いたボード（スペースのキー。「すべて」は空文字）
const LAST_BOARD_STORAGE_KEY = 'kanban:lastBoard';

// プライベートブラウズなどでブラウザ保存が使えない場合は覚えない
export const rememberLastBoard = (space) => {
  try {
    localStorage.setItem(LAST_BOARD_STORAGE_KEY, space ? space.Key : '');
  } catch {
    // 覚えられなくても動作に支障はない
  }
};

// 最後に開いたボードのスペースのキーを返す（未保存なら null、「すべて」なら空文字）
export const lastBoardKey = () => {
  try {
    return localStorage.getItem(LAST_BOARD_STORAGE_KEY);
  } catch {
    return null;
  }
};
