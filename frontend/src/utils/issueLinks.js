// 課題リンクの名前（課題から見た向き）。bff の domain.ParseRelation と同じ値で、表示もこの順にする
export const LINK_RELATION_OPTIONS = [
  { value: 'blocks', label: 'ブロックする' },
  { value: 'is blocked by', label: 'ブロックされている' },
  { value: 'relates to', label: '関連する' },
  { value: 'duplicates', label: '重複する' },
  { value: 'is duplicated by', label: '重複されている' },
  { value: 'clones', label: '複製する' },
  { value: 'is cloned by', label: '複製されている' }
];

export const DEFAULT_LINK_RELATION = 'blocks';

export const linkRelationLabel = (relation) =>
  LINK_RELATION_OPTIONS.find(option => option.value === relation)?.label ?? relation;

// まだ完了（クローズ）していない、この課題をブロックしている課題
export const openBlockers = (task) =>
  (task.Links || [])
    .filter(link => link.Relation === 'is blocked by' && link.Issue.Status !== 'Closed')
    .map(link => link.Issue);
