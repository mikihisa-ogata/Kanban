import { formatStoryPoints } from './storyPoints';
import { pointTicks } from './burndown';

// コミットメントに対する完了の割合（%）。コミットメントが 0 なら null
export const completionRate = (sprint) =>
  (sprint.Commitment > 0 ? Math.round((sprint.Completed / sprint.Commitment) * 100) : null);

// グラフの図形。スプリントごとに、コミットメントと完了の 2 本の棒を並べる。
// width・height は SVG の大きさ、padding は軸の余白 { top, right, bottom, left }
export const velocityGeometry = (velocity, { width, height, padding }) => {
  const sprints = velocity.Sprints;
  const plotWidth = width - padding.left - padding.right;
  const plotHeight = height - padding.top - padding.bottom;
  const peak = Math.max(0, velocity.Average, ...sprints.flatMap(sprint => [sprint.Commitment, sprint.Completed]));
  const yTicks = pointTicks(peak);
  const yMax = yTicks[yTicks.length - 1];
  const y = (points) => padding.top + plotHeight - (points / yMax) * plotHeight;

  // 1 スプリントの幅に、棒 2 本と左右の余白。スプリントが少なくても棒が太くなりすぎないようにする
  const slot = plotWidth / Math.max(sprints.length, 1);
  const barWidth = Math.min(36, slot * 0.3);
  const groups = sprints.map((sprint, index) => {
    const center = padding.left + slot * (index + 0.5);
    const bar = (points, x) => ({ x, y: y(points), width: barWidth, height: y(0) - y(points), points, label: formatStoryPoints(points) });
    return {
      sprint,
      center,
      commitment: bar(sprint.Commitment, center - barWidth - 2),
      completed: bar(sprint.Completed, center + 2)
    };
  });

  return {
    plot: { left: padding.left, top: padding.top, right: padding.left + plotWidth, bottom: padding.top + plotHeight },
    groups,
    average: sprints.length ? y(velocity.Average) : null,
    yTicks: yTicks.map(value => ({ y: y(value), label: formatStoryPoints(value) }))
  };
};
