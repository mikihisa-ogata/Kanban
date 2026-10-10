// ストーリーポイントの上限（bff と同じ）
const STORY_POINTS_MAX = 999999.99;

// 表示用の文字列にする（小数第 2 位までで、余分な 0 は付けない。例: 3、0.5）
export const formatStoryPoints = (points) => String(Math.round(points * 100) / 100);

// タスクのストーリーポイントの合計（未設定は数えない）。小数の誤差が出ないよう第 2 位で丸める
export const sumStoryPoints = (todos) =>
  Math.round(todos.reduce((sum, todo) => sum + (todo.StoryPoints ?? 0) * 100, 0)) / 100;

// 入力欄の文字列を読む。空なら未設定（null）。不正なら error を返す
export const parseStoryPoints = (text) => {
  const value = String(text ?? '').trim();
  if (value === '') return { value: null, error: '' };
  if (!/^\d+(\.\d{1,2})?$/.test(value) || Number(value) > STORY_POINTS_MAX) {
    return { value: null, error: '0 以上の数値（小数は第 2 位まで）を入力してください' };
  }
  return { value: Number(value), error: '' };
};
