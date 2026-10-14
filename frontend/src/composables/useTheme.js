import { THEME_OPTIONS, loadTheme, saveTheme, applyTheme } from '~/utils/theme';

// 選んでいるテーマ（system・light・dark）と、切り替える関数。ページ間で共有する。
// サーバーの描画では保存した値を読めないので、ブラウザで読み直す（見た目は head のスクリプトで先に決まっている）
export const useTheme = () => {
  const theme = useState('theme', () => 'system');
  if (import.meta.client && !useState('themeLoaded', () => false).value) {
    theme.value = loadTheme();
    useState('themeLoaded').value = true;
  }
  const setTheme = (next) => {
    theme.value = next;
    saveTheme(next);
    applyTheme(next);
  };
  return { theme, setTheme, options: THEME_OPTIONS };
};
