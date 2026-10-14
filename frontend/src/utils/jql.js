// 課題一覧の詳細検索（JQL）を画面で扱うための処理。構文の解釈と検索は bff（GET /issues/jql）が行い、
// ここでは基本モードの条件から JQL を作る・ORDER BY を書き換える・入力中の補完を出すだけを行う

import { UNASSIGNED, DEFAULT_SORT, DEFAULT_ORDER } from '~/utils/issueSearch';

const KEYWORDS = ['AND', 'OR', 'NOT', 'IN', 'IS', 'EMPTY', 'NULL', 'ORDER', 'BY', 'ASC', 'DESC'];
const isKeyword = (word) => KEYWORDS.includes(word.toUpperCase());

// 引用符なしの語に使える文字（bff の isWordRune と同じ）
const isWordChar = (ch) => !/\s/.test(ch) && !'=!~<>(),"\''.includes(ch);

// 値を JQL に書く。空白や記号を含む値・キーワード・数字だけの名前（ID と区別するため）は "…" で囲む
export const quoteJql = (value) => {
  const text = String(value);
  if (text !== '' && [...text].every(isWordChar) && !isKeyword(text) && !/^\d+$/.test(text)) return text;
  return `"${text.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`;
};

// トークンに分ける（bff の tokenizeJQL と同じ規則）。入力途中でも止まらないよう、閉じていない引用符は末尾までを文字列にする。
// kind は word・string・op・lparen・rparen・comma。start・end は文字の位置
export const tokenizeJql = (query) => {
  const chars = [...query];
  const tokens = [];
  let i = 0;
  while (i < chars.length) {
    const ch = chars[i];
    if (/\s/.test(ch)) {
      i++;
    } else if (ch === '(' || ch === ')' || ch === ',') {
      tokens.push({ kind: ch === '(' ? 'lparen' : ch === ')' ? 'rparen' : 'comma', text: ch, start: i, end: i + 1 });
      i++;
    } else if ('=~!<>'.includes(ch)) {
      const two = (ch === '!' || ch === '<' || ch === '>') && chars[i + 1] === '=';
      tokens.push({ kind: 'op', text: two ? ch + '=' : ch, start: i, end: i + (two ? 2 : 1) });
      i += two ? 2 : 1;
    } else if (ch === '"' || ch === '\'') {
      const start = i;
      let text = '';
      let closed = false;
      i++;
      while (i < chars.length) {
        if (chars[i] === '\\' && i + 1 < chars.length) {
          text += chars[i + 1];
          i += 2;
          continue;
        }
        if (chars[i] === ch) {
          closed = true;
          i++;
          break;
        }
        text += chars[i];
        i++;
      }
      tokens.push({ kind: 'string', text, start, end: i, closed });
    } else {
      const start = i;
      while (i < chars.length && isWordChar(chars[i])) i++;
      tokens.push({ kind: 'word', text: chars.slice(start, i).join(''), start, end: i });
    }
  }
  return tokens;
};

const isWord = (token, word) => token && token.kind === 'word' && token.text.toUpperCase() === word;

// JQL の ORDER BY の項目と、表の列（並べ替えの列）の対応
const ORDER_FIELDS = {
  key: 'key', issuekey: 'key', project: 'key', type: 'type', issuetype: 'type', summary: 'summary', status: 'status',
  assignee: 'assignee', reporter: 'reporter', priority: 'priority', created: 'created', createddate: 'created',
  updated: 'updated', updateddate: 'updated', due: 'due', duedate: 'due'
};

// 括弧の外にある ORDER の位置（なければ -1）
const orderByStart = (tokens) => {
  let depth = 0;
  for (let i = 0; i < tokens.length; i++) {
    if (tokens[i].kind === 'lparen') depth++;
    if (tokens[i].kind === 'rparen') depth--;
    if (depth <= 0 && isWord(tokens[i], 'ORDER') && isWord(tokens[i + 1], 'BY')) return i;
  }
  return -1;
};

// JQL の並べ替え（最初の列）を、表の列と向きで返す。ORDER BY がなければ作成日の新しい順（bff の既定）。
// 優先度は表では高い順が asc、JQL では DESC が高い順なので向きを逆にする
export const jqlSortOf = (jql) => {
  const tokens = tokenizeJql(jql);
  const start = orderByStart(tokens);
  if (start < 0) return { sort: DEFAULT_SORT, order: DEFAULT_ORDER };
  const fieldToken = tokens[start + 2];
  const sort = fieldToken && (fieldToken.kind === 'word' || fieldToken.kind === 'string') ? ORDER_FIELDS[fieldToken.text.toLowerCase()] : undefined;
  if (!sort) return { sort: '', order: '' };
  let desc = isWord(tokens[start + 3], 'DESC');
  if (sort === 'priority') desc = !desc;
  return { sort, order: desc ? 'desc' : 'asc' };
};

// JQL の ORDER BY を、表の列 sort・向き order の 1 列に置き換える
export const withJqlSort = (jql, sort, order) => {
  const tokens = tokenizeJql(jql);
  const start = orderByStart(tokens);
  const chars = [...jql];
  const where = (start < 0 ? jql : chars.slice(0, tokens[start].start).join('')).trim();
  const desc = sort === 'priority' ? order === 'asc' : order === 'desc';
  return `${where ? where + ' ' : ''}ORDER BY ${sort} ${desc ? 'DESC' : 'ASC'}`;
};

// 基本モードの条件を JQL にする。担当者は表示名（同じ表示名のユーザーがいれば ID）で書く
export const basicSearchToJql = (search, users) => {
  const clauses = [];
  const add = (field, values) => {
    if (!values.length) return;
    clauses.push(values.length === 1 ? `${field} = ${values[0]}` : `${field} IN (${values.join(', ')})`);
  };
  add('project', search.spaces.map(quoteJql));
  add('type', search.types.map(quoteJql));
  add('status', search.statuses.map(quoteJql));
  add('assignee', search.assignees.map((id) => {
    if (id === UNASSIGNED) return 'EMPTY';
    const user = users.find(u => u.ID === id);
    if (!user || users.filter(u => u.DisplayName === user.DisplayName).length > 1) return String(id);
    return quoteJql(user.DisplayName);
  }));
  add('labels', search.labels.map(quoteJql));
  if (search.text.trim()) clauses.push(`text ~ ${quoteJql(search.text.trim())}`);
  const where = clauses.join(' AND ');
  if (search.sort === DEFAULT_SORT && search.order === DEFAULT_ORDER) return where;
  return withJqlSort(where, search.sort, search.order);
};

// 補完

// 項目と、使える演算子。empty は IS EMPTY を使えるか
const JQL_FIELDS = [
  { name: 'project', detail: 'スペース', ops: ['=', '!=', 'IN', 'NOT IN'] },
  { name: 'type', detail: '課題タイプ', ops: ['=', '!=', 'IN', 'NOT IN'] },
  { name: 'status', detail: 'ステータス', ops: ['=', '!=', 'IN', 'NOT IN'] },
  { name: 'statusCategory', detail: 'ステータスのカテゴリ', ops: ['=', '!=', 'IN', 'NOT IN'] },
  { name: 'priority', detail: '優先度', ops: ['=', '!=', '>', '>=', '<', '<=', 'IN', 'NOT IN'] },
  { name: 'assignee', detail: '担当者', ops: ['=', '!=', 'IN', 'NOT IN', 'IS'] },
  { name: 'reporter', detail: '報告者', ops: ['=', '!=', 'IN', 'NOT IN', 'IS'] },
  { name: 'labels', detail: 'ラベル', ops: ['=', '!=', 'IN', 'NOT IN', 'IS'] },
  { name: 'epic', detail: 'エピック', ops: ['=', '!=', 'IN', 'NOT IN', 'IS'] },
  { name: 'sprint', detail: 'スプリント', ops: ['=', '!=', 'IN', 'NOT IN', 'IS'] },
  { name: 'fixVersion', detail: '修正バージョン', ops: ['=', '!=', 'IN', 'NOT IN', 'IS'] },
  { name: 'created', detail: '作成日時', ops: ['=', '!=', '>', '>=', '<', '<=', 'IN', 'NOT IN'] },
  { name: 'updated', detail: '更新日時', ops: ['=', '!=', '>', '>=', '<', '<=', 'IN', 'NOT IN'] },
  { name: 'due', detail: '期限', ops: ['=', '!=', '>', '>=', '<', '<=', 'IN', 'NOT IN', 'IS'] },
  { name: 'text', detail: '要約・説明・キー', ops: ['~'] }
];

// 項目の別名（bff の jqlFieldNames と同じ）
const FIELD_ALIASES = {
  issuetype: 'type', label: 'labels', 'epic link': 'epic', createddate: 'created', updateddate: 'updated', duedate: 'due'
};
const findField = (name) => {
  const lower = name.toLowerCase();
  const canonical = FIELD_ALIASES[lower] || lower;
  return JQL_FIELDS.find(f => f.name.toLowerCase() === canonical);
};

const ORDER_COLUMNS = ['key', 'type', 'summary', 'status', 'assignee', 'reporter', 'priority', 'created', 'updated', 'due'];

const unique = (list) => [...new Set(list)];

// 項目の値の候補。masters は { spaces, statuses, users, labels, epics, sprints, versions }
const valueSuggestions = (field, masters, op) => {
  const names = (list, detail) => list.map(([value, d]) => ({ label: value, insert: value, detail: d ?? detail }));
  const allowEmpty = op === '=' || op === '!=' || op === 'IN' || op === 'NOT IN';
  const empty = field.ops.includes('IS') && allowEmpty ? [{ label: 'EMPTY', insert: 'EMPTY', detail: '値なし' }] : [];
  const fn = (name, detail) => ({ label: `${name}()`, insert: `${name}()`, detail });
  switch (field.name) {
    case 'project':
      return [
        ...masters.spaces.filter(s => s.Key).map(s => ({ label: s.Key, insert: quoteJql(s.Key), detail: s.Title })),
        { label: 'TASK', insert: 'TASK', detail: 'スペース未割り当て' }
      ];
    case 'type':
      return names([['Task', 'タスク'], ['Bug', 'バグ'], ['Story', 'ストーリー'], ['Subtask', 'サブタスク']]);
    case 'status':
      return unique(masters.statuses.map(s => s.Name)).map(name => ({ label: name, insert: quoteJql(name) }));
    case 'statusCategory':
      return [
        { label: 'ToDo', insert: 'ToDo', detail: '未着手' },
        { label: 'In Progress', insert: '"In Progress"', detail: '進行中' },
        { label: 'Done', insert: 'Done', detail: '完了' }
      ];
    case 'priority':
      return names([['Highest', '最高'], ['High', '高'], ['Medium', '中'], ['Low', '低'], ['Lowest', '最低']]);
    case 'assignee':
    case 'reporter':
      return [
        fn('currentUser', '自分'),
        ...empty,
        ...masters.users.map(u => ({ label: u.DisplayName, insert: quoteJql(u.DisplayName), detail: u.Email }))
      ];
    case 'labels':
      return [...empty, ...masters.labels.map(name => ({ label: name, insert: quoteJql(name) }))];
    case 'epic':
      return [...empty, ...masters.epics.map(e => ({ label: e.Title, insert: quoteJql(e.Title), detail: `エピック ${e.ID}` }))];
    case 'sprint':
      return [
        fn('openSprints', '開始中のスプリント'), fn('futureSprints', '未開始のスプリント'), fn('closedSprints', '完了したスプリント'),
        ...empty,
        ...masters.sprints.map(s => ({ label: s.Name, insert: quoteJql(s.Name) }))
      ];
    case 'fixVersion':
      return [
        fn('unreleasedVersions', '未リリースのバージョン'), fn('releasedVersions', 'リリース済みのバージョン'),
        ...empty,
        ...masters.versions.map(v => ({ label: v.Name, insert: quoteJql(v.Name) }))
      ];
    case 'created':
    case 'updated':
    case 'due':
      return [
        fn('now', '現在'), fn('startOfDay', '今日の 0 時'),
        ...empty,
        { label: '-1d', insert: '-1d', detail: '1 日前' },
        { label: '-7d', insert: '-7d', detail: '7 日前' },
        { label: '-30d', insert: '-30d', detail: '30 日前' },
        ...(field.name === 'due' ? [{ label: '7d', insert: '7d', detail: '7 日後' }] : [])
      ];
  }
  return [];
};

const LOGIC_SUGGESTIONS = [
  { label: 'AND', insert: 'AND' },
  { label: 'OR', insert: 'OR' },
  { label: 'ORDER BY', insert: 'ORDER BY' }
];

// 位置 caret までの入力から、次に書けるものの候補を返す。
// 戻り値 { from, to, items }。候補を選んだら [from, to) を item.insert に置き換える。items の detail は説明
export const jqlSuggestions = (query, caret, masters) => {
  const chars = [...query];
  const before = chars.slice(0, caret).join('');
  const tokens = tokenizeJql(before);
  // 入力途中の語（カーソルの直前の語・閉じていない文字列）は候補を絞る文字列にし、置き換える範囲に含める
  let partial = null;
  const last = tokens[tokens.length - 1];
  if (last && last.end === caret && ((last.kind === 'word') || (last.kind === 'string' && !last.closed))) {
    partial = tokens.pop();
  }
  const prefix = partial ? partial.text.toLowerCase() : '';
  const from = partial ? partial.start : caret;

  // 関数の呼び出し（名前と括弧）は 1 つの値にまとめる
  const merged = [];
  for (let i = 0; i < tokens.length; i++) {
    const t = tokens[i];
    if (t.kind === 'word' && tokens[i + 1]?.kind === 'lparen' && tokens[i + 1].start === t.end && tokens[i + 2]?.kind === 'rparen') {
      merged.push({ ...t, text: `${t.text}()`, end: tokens[i + 2].end });
      i += 2;
      continue;
    }
    merged.push(t);
  }

  // トークンをたどって、次に来るものを決める
  let state = 'field';
  let field = null;
  let op = '';
  let depth = 0; // 開いている括弧（条件のまとまり）の数
  for (const t of merged) {
    const word = t.kind === 'word' ? t.text.toUpperCase() : '';
    switch (state) {
      case 'field':
        if (t.kind === 'lparen') { depth++; break; }
        if (word === 'NOT') break;
        if (word === 'ORDER') { state = 'by'; break; }
        if (t.kind === 'word' || t.kind === 'string') { field = findField(t.text) || null; state = 'operator'; break; }
        state = 'unknown';
        break;
      case 'operator':
        if (t.kind === 'op') { op = t.text; state = 'value'; break; }
        if (word === 'IN') { op = 'IN'; state = 'listOpen'; break; }
        if (word === 'NOT') { state = 'notIn'; break; }
        if (word === 'IS') { state = 'isTail'; break; }
        state = 'unknown';
        break;
      case 'notIn':
        state = word === 'IN' ? 'listOpen' : 'unknown';
        op = 'NOT IN';
        break;
      case 'isTail':
      case 'isNotTail':
        if (word === 'NOT' && state === 'isTail') { state = 'isNotTail'; break; }
        state = word === 'EMPTY' || word === 'NULL' ? 'logic' : 'unknown';
        break;
      case 'value':
        state = t.kind === 'word' || t.kind === 'string' ? 'logic' : 'unknown';
        break;
      case 'listOpen':
        state = t.kind === 'lparen' ? 'listValue' : 'unknown';
        break;
      case 'listValue':
        if (t.kind === 'word' || t.kind === 'string') { state = 'listNext'; break; }
        state = 'unknown';
        break;
      case 'listNext':
        state = t.kind === 'rparen' ? 'logic' : t.kind === 'comma' ? 'listValue' : 'unknown';
        break;
      case 'logic':
        if (t.kind === 'rparen') { depth--; break; }
        if (word === 'AND' || word === 'OR') { state = 'field'; field = null; break; }
        if (word === 'ORDER') { state = 'by'; break; }
        state = 'unknown';
        break;
      case 'by':
        state = word === 'BY' ? 'orderField' : 'unknown';
        break;
      case 'orderField':
        state = t.kind === 'word' || t.kind === 'string' ? 'orderNext' : 'unknown';
        break;
      case 'orderNext':
        if (word === 'ASC' || word === 'DESC') break;
        state = t.kind === 'comma' ? 'orderField' : 'unknown';
        break;
    }
  }

  let items = [];
  switch (state) {
    case 'field':
      items = [
        ...JQL_FIELDS.map(f => ({ label: f.name, insert: f.name, detail: f.detail })),
        ...(merged.length === 0 ? [{ label: 'ORDER BY', insert: 'ORDER BY' }] : []),
        { label: 'NOT', insert: 'NOT' }
      ];
      break;
    case 'operator':
      items = (field ? field.ops : ['=', '!=', 'IN', 'NOT IN']).map(o => ({ label: o === 'IS' ? 'IS EMPTY' : o, insert: o === 'IS' ? 'IS EMPTY' : o }));
      if (field && field.ops.includes('IS')) items.push({ label: 'IS NOT EMPTY', insert: 'IS NOT EMPTY' });
      break;
    case 'notIn':
      items = [{ label: 'IN', insert: 'IN (' }];
      break;
    case 'isTail':
      items = [{ label: 'EMPTY', insert: 'EMPTY' }, { label: 'NOT EMPTY', insert: 'NOT EMPTY' }];
      break;
    case 'isNotTail':
      items = [{ label: 'EMPTY', insert: 'EMPTY' }];
      break;
    case 'value':
    case 'listValue':
      items = field ? valueSuggestions(field, masters, op) : [];
      break;
    case 'listOpen':
      items = [{ label: '(', insert: '(' }];
      break;
    case 'listNext':
      items = [{ label: ',', insert: ',' }, { label: ')', insert: ')' }];
      break;
    case 'logic':
      // 括弧の中では ORDER BY の代わりに閉じ括弧
      items = depth > 0 ? [...LOGIC_SUGGESTIONS.slice(0, 2), { label: ')', insert: ')' }] : LOGIC_SUGGESTIONS;
      break;
    case 'by':
      items = [{ label: 'BY', insert: 'BY' }];
      break;
    case 'orderField':
      items = ORDER_COLUMNS.map(name => ({ label: name, insert: name }));
      break;
    case 'orderNext':
      items = [{ label: 'ASC', insert: 'ASC' }, { label: 'DESC', insert: 'DESC' }, { label: ',', insert: ',' }];
      break;
  }
  if (prefix) {
    // 先頭が合うものを先に、途中に含むものを後に出す
    const starts = items.filter(item => item.label.toLowerCase().startsWith(prefix));
    const contains = items.filter(item => !starts.includes(item) && item.label.toLowerCase().includes(prefix));
    items = [...starts, ...contains];
    // 入力済みの語そのものしか候補がなければ出さない
    if (items.length === 1 && items[0].label.toLowerCase() === prefix) items = [];
  }
  return { from, to: caret, items: items.slice(0, 20) };
};

// 候補を選んだ後の文字列とカーソルの位置。語の後には空白を足す（括弧を開いた後は足さない）
export const applyJqlSuggestion = (query, { from, to }, item) => {
  const chars = [...query];
  const rest = chars.slice(to).join('');
  const space = item.insert.endsWith('(') || rest.startsWith(' ') ? '' : ' ';
  const head = chars.slice(0, from).join('') + item.insert + space;
  return { value: head + rest, caret: [...head].length };
};
