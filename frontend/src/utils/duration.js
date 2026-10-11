// 時間（分）を「1w 2d 3h 30m」形式で表示・入力する。bff の domain.ParseDuration / FormatDuration と同じ規則で、
// 1 日は 8 時間、1 週は 5 日として数える
export const MINUTES_PER_HOUR = 60;
export const MINUTES_PER_DAY = 8 * MINUTES_PER_HOUR;
export const MINUTES_PER_WEEK = 5 * MINUTES_PER_DAY;

const UNITS = [
  { size: MINUTES_PER_WEEK, label: 'w' },
  { size: MINUTES_PER_DAY, label: 'd' },
  { size: MINUTES_PER_HOUR, label: 'h' },
  { size: 1, label: 'm' }
];

export const DURATION_FORMAT_HINT = '例: 1w 2d 3h 30m（1d=8h、1w=5d）';

// 分を「1w 2d 3h 30m」形式にする。0 の単位は省き、0 分は「0m」にする
export const formatDuration = (minutes) => {
  if (!minutes || minutes <= 0) return '0m';
  let rest = minutes;
  const parts = [];
  for (const unit of UNITS) {
    const n = Math.floor(rest / unit.size);
    if (n > 0) {
      parts.push(`${n}${unit.label}`);
      rest -= n * unit.size;
    }
  }
  return parts.join(' ');
};

// 入力欄の文字列を読む。空なら未設定（null）。読めなければ error を返す（送信前の確認用。最終的な判定は bff が行う）
export const parseDuration = (text) => {
  let rest = text.trim().toLowerCase();
  if (!rest) return { value: null, error: '' };
  const sizes = Object.fromEntries(UNITS.map(unit => [unit.label, unit.size]));
  let total = 0;
  while (rest) {
    const match = rest.match(/^(\d+(?:\.\d+)?)([wdhm])/);
    if (!match) {
      return { value: null, error: '「1w 2d 3h 30m」の形式で入力してください（w=週・d=日・h=時間・m=分、1d=8h、1w=5d）' };
    }
    total += Number(match[1]) * sizes[match[2]];
    rest = rest.slice(match[0].length).replace(/^[ \t]+/, '');
  }
  return { value: Math.round(total), error: '' };
};
