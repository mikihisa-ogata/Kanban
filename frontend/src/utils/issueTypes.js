// サブタスク。親課題を指定して作り、他の課題タイプとは相互に変えられないので、選択肢には含めない
export const SUBTASK_ISSUE_TYPE = 'Subtask';

// 選んで作成・変更できる課題タイプ（Jira の Task / Bug / Story）。bff の domain.IssueType と同じ値
export const ISSUE_TYPE_OPTIONS = [
  { value: 'Task', label: 'タスク' },
  { value: 'Bug', label: 'バグ' },
  { value: 'Story', label: 'ストーリー' }
];

export const DEFAULT_ISSUE_TYPE = 'Task';

export const issueTypeLabel = (type) =>
  type === SUBTASK_ISSUE_TYPE ? 'サブタスク' : (ISSUE_TYPE_OPTIONS.find(option => option.value === type)?.label ?? 'タスク');
