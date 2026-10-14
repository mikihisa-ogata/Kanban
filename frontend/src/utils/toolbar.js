// ページ上部のツールバーのボタンの見た目。active は絞り込みなどが効いているとき
export const toolbarButtonClass = (active = false) => [
  'inline-flex items-center gap-1.5 px-2.5 py-1 border rounded-md text-xs font-medium cursor-pointer whitespace-nowrap transition-colors duration-150',
  active ? 'bg-accent/15 border-accent/40 text-accent' : 'bg-transparent border-line text-fg hover:bg-raised'
];

// ツールバーの小さな情報（スプリントの期間など）。danger は期限切れなど
export const toolbarChipClass = (danger = false) => [
  'inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full border text-xs whitespace-nowrap',
  danger ? 'border-danger/40 text-danger' : 'border-line text-muted'
];
