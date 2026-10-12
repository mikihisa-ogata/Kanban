// バージョンの作成・編集フォームの値（名前・説明・開始日・リリース日）
export const emptyVersionFields = () => ({ name: '', description: '', startDate: '', releaseDate: '' });

// フォームの入力の誤り（なければ空文字）。bff の検証と同じ条件
export const versionFieldsError = ({ name, description, startDate, releaseDate }) => {
  if (!name.trim()) return '名前を入力してください';
  if (name.trim().length > 100) return '名前は 100 文字以内で入力してください';
  if (description.trim().length > 1000) return '説明は 1000 文字以内で入力してください';
  if (startDate && releaseDate && releaseDate < startDate) return 'リリース日は開始日以降にしてください';
  return '';
};
