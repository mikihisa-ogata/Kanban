import { formatStoryPoints } from './storyPoints';
import { SPRINT_ACTIVE, SPRINT_CLOSED } from './sprints';

// バーンダウンの出来事の種類（bff の domain.BurndownEventKind）の表示名
export const BURNDOWN_EVENT_LABELS = {
  added: '追加',
  removed: '除外',
  completed: '完了',
  reopened: '再開',
  estimate: '見積もり変更'
};

// レポートに出せるスプリント（開始中と完了したもの）。開始中を先に、完了したものは新しい順
export const reportableSprints = (sprints) => [
  ...sprints.filter(sprint => sprint.State === SPRINT_ACTIVE),
  ...sprints
    .filter(sprint => sprint.State === SPRINT_CLOSED)
    .sort((a, b) => String(b.CompletedAt ?? '').localeCompare(String(a.CompletedAt ?? '')) || b.ID - a.ID)
];

// 残りポイントの増減の表示（例: +3、−5）。0 は空文字
export const formatChange = (value) => {
  if (!value) return '';
  return `${value > 0 ? '+' : '−'}${formatStoryPoints(Math.abs(value))}`;
};

// 日付（YYYY-MM-DD）をローカルの 0 時として読む
const parseDay = (text) => {
  const [y, m, d] = text.split('-').map(Number);
  return new Date(y, m - 1, d);
};

// 縦軸の目盛り。0 から max 以上のきりのよい値まで、5 つ前後に分ける
export const pointTicks = (max) => {
  if (max <= 0) return [0, 1];
  const rough = max / 5;
  const magnitude = 10 ** Math.floor(Math.log10(rough));
  const step = [1, 2, 5, 10].map(n => n * magnitude).find(n => n >= rough);
  const ticks = [];
  for (let value = 0; value < max + step; value += step) ticks.push(Math.round(value * 100) / 100);
  return ticks;
};

// グラフの図形。width・height は SVG の大きさ、padding は軸の余白 { top, right, bottom, left }
export const burndownGeometry = (burndown, { width, height, padding }) => {
  const days = burndown.Days;
  const from = parseDay(days[0].Date).getTime();
  const to = parseDay(days[days.length - 1].Date).getTime() + 24 * 60 * 60 * 1000;
  const plotWidth = width - padding.left - padding.right;
  const plotHeight = height - padding.top - padding.bottom;
  const peak = Math.max(burndown.StartPoints, ...burndown.Events.map(event => event.Remaining), 0);
  const yTicks = pointTicks(peak);
  const yMax = yTicks[yTicks.length - 1];

  const x = (time) => padding.left + ((new Date(time).getTime() - from) / (to - from)) * plotWidth;
  const y = (points) => padding.top + plotHeight - (points / yMax) * plotHeight;

  // 実績線は階段状（出来事のときに縦に動く）
  const startX = x(burndown.Start);
  let path = `M${startX},${y(burndown.StartPoints)}`;
  const markers = [];
  for (const event of burndown.Events) {
    const ex = x(event.Time);
    path += ` H${ex} V${y(event.Remaining)}`;
    markers.push({ x: ex, y: y(event.Remaining), event });
  }
  const untilX = x(burndown.Until);
  path += ` H${untilX}`;

  // 日付の目盛り。日が多いときは間引く
  const every = Math.max(1, Math.ceil(days.length / 14));
  const xTicks = days
    .map((day, index) => ({ day, index }))
    .filter(({ index }) => index % every === 0)
    .map(({ day }) => {
      const date = parseDay(day.Date);
      return { x: x(date), label: `${date.getMonth() + 1}/${date.getDate()}`, weekend: date.getDay() === 0 || date.getDay() === 6 };
    });
  // 土日の帯
  const weekends = days
    .map(day => parseDay(day.Date))
    .filter(date => date.getDay() === 0 || date.getDay() === 6)
    .map(date => ({ x: x(date), width: plotWidth / days.length }));

  return {
    plot: { left: padding.left, top: padding.top, right: padding.left + plotWidth, bottom: padding.top + plotHeight },
    ideal: { x1: startX, y1: y(burndown.StartPoints), x2: x(burndown.End), y2: y(0) },
    path,
    markers,
    untilX,
    xTicks,
    yTicks: yTicks.map(value => ({ y: y(value), label: formatStoryPoints(value) })),
    weekends
  };
};

// 日ごとの表の行。課題の出来事をその日に寄せる
export const burndownRows = (burndown) =>
  burndown.Days.map(day => ({
    ...day,
    events: burndown.Events.filter(event => event.Date === day.Date)
  }));
