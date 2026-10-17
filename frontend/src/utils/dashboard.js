// ダッシュボードのガジェット（bff の domain.GadgetType）。usesFilter はフィルターを選べる種類
export const GADGET_TYPES = [
  { type: 'assignedToMe', label: '自分の担当課題', description: '自分が担当している未完了の課題', usesFilter: false },
  { type: 'activityStream', label: '最近のアクティビティ', description: 'すべての課題の最近の変更', usesFilter: false },
  { type: 'filterResults', label: '保存済みフィルターの結果', description: '選んだフィルターに合う課題', usesFilter: true },
  { type: 'statusPie', label: 'ステータス別の円グラフ', description: '課題のステータスの内訳', usesFilter: true },
  { type: 'createdVsResolved', label: '作成 vs 解決', description: '日ごとに作成・解決された課題の数', usesFilter: true }
];

export const gadgetType = (type) => GADGET_TYPES.find(t => t.type === type) ?? { type, label: type, description: '', usesFilter: false };

// ダッシュボードに置けるガジェットの数の上限（bff の domain.DashboardGadgetMax）
export const DASHBOARD_GADGET_MAX = 20;

// 自分の担当課題のガジェットの JQL（優先度の高い順、同じ優先度は更新の新しい順）
export const ASSIGNED_TO_ME_JQL = 'assignee = currentUser() AND statusCategory != Done ORDER BY priority ASC, updated DESC';

// 作成 vs 解決で集計する日数
export const TREND_DAYS = 30;

// 新しいガジェットの ID（並びの中で重ならない英数字）
export const newGadgetId = (gadgets) => {
  const used = new Set(gadgets.map(g => g.ID));
  let id;
  do {
    id = `g${Date.now().toString(36)}${Math.floor(Math.random() * 1296).toString(36)}`;
  } while (used.has(id));
  return id;
};

// index のガジェットを offset（-1 で前、1 で後ろ）へ動かした新しい並び。端を越えるなら同じ並び
export const moveGadget = (gadgets, index, offset) => {
  const target = index + offset;
  if (target < 0 || target >= gadgets.length) return gadgets;
  const next = [...gadgets];
  [next[index], next[target]] = [next[target], next[index]];
  return next;
};

// ステータスのカテゴリごとの円グラフの色。同じカテゴリの 2 つ目以降のステータスは、同じ色を薄くして見分ける
const CATEGORY_COLORS = { ToDo: 'var(--kb-muted)', InProgress: 'var(--kb-accent)', Done: 'var(--kb-success)' };
const SHADE_OPACITIES = [1, 0.65, 0.4];

// 円グラフ（ドーナツ）の扇形。counts は GET /dashboard/status-counts の結果（カテゴリの順）。
// 各扇形は { name, category, count, percent, color, opacity, path }。1 つで 100% のときは path の代わりに full が true
export const pieSlices = (counts, { cx, cy, outer, inner }) => {
  const total = counts.reduce((sum, c) => sum + c.Count, 0);
  if (total === 0) return [];
  const shadeIndex = {};
  let angle = -Math.PI / 2;
  return counts.map((c) => {
    const index = shadeIndex[c.Category] ?? 0;
    shadeIndex[c.Category] = index + 1;
    const sweep = (c.Count / total) * Math.PI * 2;
    const start = angle;
    const end = angle + sweep;
    angle = end;
    const slice = {
      name: c.Name || '（ステータスなし）',
      category: c.Category,
      count: c.Count,
      percent: Math.round((c.Count / total) * 1000) / 10,
      color: CATEGORY_COLORS[c.Category] ?? 'var(--kb-muted)',
      opacity: SHADE_OPACITIES[Math.min(index, SHADE_OPACITIES.length - 1)],
      full: c.Count === total,
      path: ''
    };
    if (!slice.full) {
      const large = sweep > Math.PI ? 1 : 0;
      const point = (r, a) => `${(cx + r * Math.cos(a)).toFixed(2)} ${(cy + r * Math.sin(a)).toFixed(2)}`;
      slice.path = `M ${point(outer, start)} A ${outer} ${outer} 0 ${large} 1 ${point(outer, end)} L ${point(inner, end)} A ${inner} ${inner} 0 ${large} 0 ${point(inner, start)} Z`;
    }
    return slice;
  });
};

// 縦軸の目盛りの間隔（1・2・5 の倍数で、目盛りが 4 本前後になるもの）
const niceStep = (max) => {
  const rough = Math.max(max, 1) / 4;
  const power = 10 ** Math.floor(Math.log10(rough));
  const step = [1, 2, 5, 10].map(m => m * power).find(s => s >= rough);
  return Math.max(1, step);
};

// 作成 vs 解決の折れ線の座標。trend は GET /dashboard/created-vs-resolved の結果
export const trendGeometry = (trend, { width, height, padding }) => {
  const plot = { left: padding.left, right: width - padding.right, top: padding.top, bottom: height - padding.bottom };
  const days = trend.Days;
  const max = Math.max(1, ...days.map(d => Math.max(d.Created, d.Resolved)));
  const step = niceStep(max);
  const top = Math.ceil(max / step) * step;
  const x = (i) => (days.length === 1 ? (plot.left + plot.right) / 2 : plot.left + ((plot.right - plot.left) * i) / (days.length - 1));
  const y = (v) => plot.bottom - ((plot.bottom - plot.top) * v) / top;
  const yTicks = [];
  for (let v = 0; v <= top; v += step) yTicks.push({ y: y(v), label: String(v) });
  // 横軸の日付は 6 つ程度に間引き、最後の日（今日）は必ず出す
  const every = Math.max(1, Math.ceil(days.length / 6));
  const xTicks = days
    .map((d, i) => ({ i, x: x(i), label: shortDate(d.Date) }))
    .filter(t => (days.length - 1 - t.i) % every === 0);
  const points = days.map((d, i) => ({ ...d, x: x(i), created: y(d.Created), resolved: y(d.Resolved) }));
  const line = (key) => points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x.toFixed(2)} ${p[key].toFixed(2)}`).join(' ');
  // カーソルを合わせる範囲は、隣の日との中間まで
  const half = days.length > 1 ? (plot.right - plot.left) / (days.length - 1) / 2 : (plot.right - plot.left) / 2;
  return { plot, yTicks, xTicks, points, half, createdPath: line('created'), resolvedPath: line('resolved') };
};

// YYYY-MM-DD を「10/17」にする
export const shortDate = (date) => {
  const [, m, d] = date.split('-');
  return `${Number(m)}/${Number(d)}`;
};
