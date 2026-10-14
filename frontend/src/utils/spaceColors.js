// スペース・ラベルの目印の色。どちらも色の設定はないので、ID や名前から決まった色を選ぶ（同じものはいつも同じ色）
const SPACE_COLORS = ['#5e6ad2', '#26b5ce', '#4cb782', '#f2994a', '#eb5757', '#bb87fc', '#b8860b', '#7b8494'];

export const spaceColor = (space) => SPACE_COLORS[Math.abs(space.ID) % SPACE_COLORS.length];

// ラベルの名前（大文字・小文字は区別しない）から色を選ぶ
export const labelColor = (name) => {
  let hash = 0;
  for (const ch of name.toLowerCase()) hash = (hash * 31 + ch.codePointAt(0)) | 0;
  return SPACE_COLORS[Math.abs(hash) % SPACE_COLORS.length];
};

// 色（#rrggbb）の上に置く文字の色。白と濃い色のうち、コントラスト比の高い方を返す
export const readableTextOn = (hex) => {
  const m = /^#?([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(hex || '');
  if (!m) return '#ffffff';
  const channel = (v) => {
    const c = parseInt(v, 16) / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  };
  const lum = 0.2126 * channel(m[1]) + 0.7152 * channel(m[2]) + 0.0722 * channel(m[3]);
  const dark = 0.0103; // #1a1d23 の相対輝度
  return (1.05 / (lum + 0.05)) >= ((lum + 0.05) / (dark + 0.05)) ? '#ffffff' : '#1a1d23';
};
