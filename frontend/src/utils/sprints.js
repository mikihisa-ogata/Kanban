// スプリントの状態（bff の domain.SprintState）
export const SPRINT_FUTURE = 'future';
export const SPRINT_ACTIVE = 'active';
export const SPRINT_CLOSED = 'closed';

// スペースの開始中のスプリント（なければ null）
export const activeSprintOf = (sprints, spaceId) =>
  sprints.find(sprint => sprint.SpaceID === spaceId && sprint.State === SPRINT_ACTIVE) ?? null;

// 日付（YYYY-MM-DD）をローカルの日付として読む
const parseDate = (text) => {
  const [y, m, d] = text.split('-').map(Number);
  return new Date(y, m - 1, d);
};

// 今日の日付（YYYY-MM-DD）と、そこから days 日後の日付
export const dateAfter = (days, from = new Date()) => {
  const date = new Date(from.getFullYear(), from.getMonth(), from.getDate() + days);
  const pad = (n) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
};

// スプリントの終了日までの残り日数（終了日当日は 0、過ぎたら負）。終了日がなければ null
export const remainingDays = (sprint, now = new Date()) => {
  if (!sprint.EndDate) return null;
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  return Math.round((parseDate(sprint.EndDate) - today) / (24 * 60 * 60 * 1000));
};

// 残り日数の表示（Jira の「残り 3 日」）
export const remainingDaysLabel = (sprint, now = new Date()) => {
  const days = remainingDays(sprint, now);
  if (days === null) return '';
  if (days > 0) return `残り ${days} 日`;
  if (days === 0) return '今日まで';
  return `終了日を ${-days} 日過ぎています`;
};

// 期間の表示（例: 10/12 〜 10/26）
export const sprintPeriodLabel = (sprint) => {
  if (!sprint.StartDate || !sprint.EndDate) return '';
  const short = (text) => {
    const date = parseDate(text);
    return `${date.getMonth() + 1}/${date.getDate()}`;
  };
  return `${short(sprint.StartDate)} 〜 ${short(sprint.EndDate)}`;
};
