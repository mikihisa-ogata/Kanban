// サイドバーのスペースの色の四角。スペースに色の設定はないので、ID から決まった色を選ぶ（同じスペースはいつも同じ色）
const SPACE_COLORS = ['#5e6ad2', '#26b5ce', '#4cb782', '#f2994a', '#eb5757', '#bb87fc', '#f2c94c', '#95a2b3'];

export const spaceColor = (space) => SPACE_COLORS[Math.abs(space.ID) % SPACE_COLORS.length];
