// スペースのキー（課題キーの接頭辞）の形式。bff の domain.ValidSpaceKey と同じ
const SPACE_KEY_PATTERN = /^[A-Z][A-Z0-9]{1,9}$/;
// スペース未割り当てのタスクの課題キーで使うため、スペースのキーには使えない
const RESERVED_KEY = 'TASK';

// 入力されたキーを保存する形（大文字・空白なし）にする
export const normalizeSpaceKey = (value) => value.toUpperCase().replace(/\s/g, '');

// キーの入力エラーを返す（問題なければ空文字）。excludeId のスペースのキーとは重複扱いにしない
export const spaceKeyError = (key, spaces, excludeId = null) => {
  if (!key) return 'キーを入力してください';
  if (!SPACE_KEY_PATTERN.test(key) || key === RESERVED_KEY) {
    return `英大文字で始まる英大文字・数字の 2〜10 文字にしてください（${RESERVED_KEY} は使えません）`;
  }
  if (spaces.some(space => space.ID !== excludeId && space.Key === key)) {
    return 'このキーは他のスペースで使われています';
  }
  return '';
};
