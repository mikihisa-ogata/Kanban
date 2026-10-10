// 優先度（Jira の Highest〜Lowest）。bff の domain.Priority と同じ値で、高い順に並べる
export const PRIORITY_OPTIONS = [
  { value: 'Highest', label: '最高' },
  { value: 'High', label: '高' },
  { value: 'Medium', label: '中' },
  { value: 'Low', label: '低' },
  { value: 'Lowest', label: '最低' }
];

export const DEFAULT_PRIORITY = 'Medium';

export const priorityLabel = (priority) => PRIORITY_OPTIONS.find(option => option.value === priority)?.label ?? '中';
