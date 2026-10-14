// スペース・ラベルの目印の色。どちらも色の設定はないので、ID や名前から決まった色を選ぶ（同じものはいつも同じ色）
const SPACE_COLORS = ['#5e6ad2', '#26b5ce', '#4cb782', '#f2994a', '#eb5757', '#bb87fc', '#f2c94c', '#95a2b3'];

export const spaceColor = (space) => SPACE_COLORS[Math.abs(space.ID) % SPACE_COLORS.length];

// ラベルの名前（大文字・小文字は区別しない）から色を選ぶ
export const labelColor = (name) => {
  let hash = 0;
  for (const ch of name.toLowerCase()) hash = (hash * 31 + ch.codePointAt(0)) | 0;
  return SPACE_COLORS[Math.abs(hash) % SPACE_COLORS.length];
};
