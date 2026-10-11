// API が返す日時（UTC の RFC 3339 形式）を、ブラウザのタイムゾーンの「2026/10/10 21:30」形式にする
export const formatDateTime = (value) => {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleString('ja-JP', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  });
};

// API が返す日時を now からの相対表示（「たった今」「3 分前」「2 時間前」「5 日前」）にする。
// 1 週間以上前は formatDateTime の絶対日時にする
export const formatRelativeTime = (value, now = Date.now()) => {
  if (!value) return '';
  const time = new Date(value).getTime();
  if (Number.isNaN(time)) return '';
  const minutes = Math.floor((now - time) / 60000);
  if (minutes < 1) return 'たった今';
  if (minutes < 60) return `${minutes} 分前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} 時間前`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days} 日前`;
  return formatDateTime(value);
};
