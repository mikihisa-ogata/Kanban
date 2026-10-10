// 課題タイプ（Jira の Task / Bug / Story）。bff の domain.IssueType と同じ値
export const ISSUE_TYPE_OPTIONS = [
  { value: 'Task', label: 'タスク' },
  { value: 'Bug', label: 'バグ' },
  { value: 'Story', label: 'ストーリー' }
];

export const DEFAULT_ISSUE_TYPE = 'Task';

export const issueTypeLabel = (type) => ISSUE_TYPE_OPTIONS.find(option => option.value === type)?.label ?? 'タスク';
