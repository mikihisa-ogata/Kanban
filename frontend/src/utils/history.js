import { issueTypeLabel } from './issueTypes';
import { priorityLabel } from './priorities';
import { formatStoryPoints } from './storyPoints';
import { formatDuration } from './duration';

// 変更履歴の項目名（bff の domain.HistoryField* と同じ値）の表示名
const FIELD_LABELS = {
  title: '要約',
  description: '説明',
  status: 'ステータス',
  done: '完了',
  issueType: '課題タイプ',
  priority: '優先度',
  labels: 'ラベル',
  storyPoints: 'ストーリーポイント',
  originalEstimate: '初期見積もり',
  remainingEstimate: '残り時間',
  deadline: '期限',
  epic: 'エピック',
  space: 'スペース',
  key: '課題キー'
};

const STATUS_LABELS = {
  Open: 'オープン',
  InProgress: '進行中',
  Waiting: '待ち状況',
  Closed: 'クローズ'
};

export const historyFieldLabel = (field) => FIELD_LABELS[field] ?? field;

// 変更前・変更後の値を表示用の文字列にする。null は「なし」
export const formatHistoryValue = (field, value) => {
  if (value == null) return 'なし';
  switch (field) {
    case 'status':
      return STATUS_LABELS[value] ?? value;
    case 'done':
      return value === 'true' ? '完了' : '未完了';
    case 'issueType':
      return issueTypeLabel(value);
    case 'priority':
      return priorityLabel(value);
    case 'storyPoints':
      return formatStoryPoints(Number(value));
    case 'originalEstimate':
    case 'remainingEstimate':
      return formatDuration(Number(value));
    default:
      return value;
  }
};
