// バージョンの状態（bff の domain.VersionStatus）
export const VERSION_UNRELEASED = 'unreleased';
export const VERSION_RELEASED = 'released';
export const VERSION_ARCHIVED = 'archived';

export const VERSION_STATUS_LABEL = {
  unreleased: '未リリース',
  released: 'リリース済み',
  archived: 'アーカイブ'
};

// 状態のバッジの色（Jira と同じく、未リリースは灰、リリース済みは緑、アーカイブは薄い灰）
export const VERSION_STATUS_BADGE_CLASS = {
  unreleased: 'bg-raised text-fg',
  released: 'bg-success/10 text-success',
  archived: 'bg-raised text-muted'
};

// バージョンの課題の進捗（ステータスのカテゴリごとの件数）
export const versionProgress = (issues) => ({
  done: issues.filter(todo => todo.StatusCategory === 'Done').length,
  inProgress: issues.filter(todo => todo.StatusCategory === 'InProgress').length,
  toDo: issues.filter(todo => todo.StatusCategory !== 'Done' && todo.StatusCategory !== 'InProgress').length,
  total: issues.length
});

// バージョンが付いている課題
export const issuesOfVersion = (todos, versionId) => todos.filter(todo => (todo.FixVersionIDs || []).includes(versionId));
