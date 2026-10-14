// 画面のテーマ（システムに合わせる / ライト / ダーク）。選んだ値はブラウザ（localStorage）に保存する。
// ライト・ダークを選んだら <html data-theme="light|dark"> を付け、システムに合わせるときは外して CSS の prefers-color-scheme に任せる

export const THEME_STORAGE_KEY = 'kanbanTheme';

export const THEME_OPTIONS = [
  { value: 'system', label: 'システムに合わせる' },
  { value: 'light', label: 'ライト' },
  { value: 'dark', label: 'ダーク' }
];

const isTheme = (value) => THEME_OPTIONS.some(option => option.value === value);

// 保存しているテーマ。保存していない・読めないときは「システムに合わせる」
export const loadTheme = () => {
  try {
    const saved = localStorage.getItem(THEME_STORAGE_KEY);
    return isTheme(saved) ? saved : 'system';
  } catch {
    return 'system';
  }
};

export const saveTheme = (theme) => {
  try {
    if (theme === 'system') {
      localStorage.removeItem(THEME_STORAGE_KEY);
    } else {
      localStorage.setItem(THEME_STORAGE_KEY, theme);
    }
  } catch {
    // 保存できなくても、この画面では切り替える
  }
};

export const applyTheme = (theme) => {
  if (theme === 'light' || theme === 'dark') {
    document.documentElement.dataset.theme = theme;
  } else {
    delete document.documentElement.dataset.theme;
  }
};

// 読み込み直後に別のテーマが一瞬表示されないよう、head で CSS より先に実行するスクリプト（nuxt.config.ts から使う）。
// 上の loadTheme・applyTheme と同じことをする
export const THEME_INIT_SCRIPT = `(function(){try{var t=localStorage.getItem('${THEME_STORAGE_KEY}');if(t==='light'||t==='dark'){document.documentElement.dataset.theme=t;}}catch(e){}})();`;
