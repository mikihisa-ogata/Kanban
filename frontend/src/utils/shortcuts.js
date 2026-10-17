// キーボードショートカット（Jira と同じキー）。一覧は `?` のダイアログに出す
export const SHORTCUT_GROUPS = [
  {
    title: '全体',
    items: [
      { keys: ['/'], label: '検索' },
      { keys: ['⌘', 'K'], label: '検索（Windows は Ctrl+K）' },
      { keys: ['c'], label: '課題を作成' },
      { keys: ['g', 'b'], label: 'ボードを開く', sequence: true },
      { keys: ['g', 'k'], label: 'バックログを開く（スペースの中で）', sequence: true },
      { keys: ['?'], label: 'ショートカットの一覧' }
    ]
  },
  {
    title: '課題を開いているとき',
    items: [
      { keys: ['e'], label: '要約を編集' },
      { keys: ['a'], label: '担当者を変更' },
      { keys: ['i'], label: '自分に割り当て' },
      { keys: ['m'], label: 'コメントを追加' }
    ]
  }
];

// 入力欄（テキスト入力・テキストエリア・選択・contenteditable）にフォーカスがあるか
const isTypingTarget = (target) => {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  if (target.tagName === 'TEXTAREA' || target.tagName === 'SELECT') return true;
  if (target.tagName !== 'INPUT') return false;
  // チェックボックスやボタンの input ではショートカットを使える
  return !['checkbox', 'radio', 'button', 'submit', 'reset', 'file', 'range', 'color'].includes(target.type);
};

// 1 文字のショートカットを無視する場面か。入力中・修飾キー付き・日本語入力の変換中・
// ダイアログ（aria-modal）を開いているときは、ページのショートカットを動かさない
export const shouldIgnoreShortcut = (event) =>
  event.defaultPrevented ||
  event.metaKey || event.ctrlKey || event.altKey ||
  event.isComposing || event.keyCode === 229 ||
  isTypingTarget(event.target) ||
  document.querySelector('[aria-modal="true"]') !== null;

// 課題キー（例: KAN-12）の形か。検索ボックスで Enter を押したら、その課題を開く
export const looksLikeIssueKey = (text) => /^[A-Za-z][A-Za-z0-9]*-\d+$/.test(text.trim());
