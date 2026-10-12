import { Marked } from 'marked';
import DOMPurify from 'dompurify';

const escapeHTML = (text) => text
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;');

// メンション（@[表示名](user:ID)。utils/mentions.js）を、ハイライトした「@表示名」にする marked の拡張。
// コードの中は Markdown として読まれないので、そのまま文字で出る
const mentionExtension = {
  name: 'mention',
  level: 'inline',
  start: (src) => src.indexOf('@['),
  tokenizer(src) {
    const match = /^@\[([^\]\n]+)\]\(user:(\d+)\)/.exec(src);
    if (!match) return undefined;
    return { type: 'mention', raw: match[0], name: match[1], userId: Number(match[2]) };
  },
  renderer: (token) => `<span class="mention" data-user-id="${token.userId}">@${escapeHTML(token.name)}</span>`
};

const markdown = new Marked({ gfm: true, breaks: true, async: false, extensions: [mentionExtension] });

// リンクは別のタブで開き、開いた先からこのページを操作させない
let hookAdded = false;
const addLinkHook = () => {
  if (hookAdded) return;
  DOMPurify.addHook('afterSanitizeAttributes', (node) => {
    if (node.tagName === 'A' && node.getAttribute('href')) {
      node.setAttribute('target', '_blank');
      node.setAttribute('rel', 'noopener noreferrer');
    }
  });
  hookAdded = true;
};

// Markdown を表示用の HTML にする（GitHub 風の表・チェックボックスに対応し、改行はそのまま改行にする）。
// メンションはハイライトし、currentUserId のユーザー（自分）へのメンションは目立つ色にする（mention-self）。
// <script> やイベント属性などを除くため、必ず DOMPurify でサニタイズしてから返す。
// DOMPurify はブラウザの DOM を使うので、ブラウザでだけ呼ぶ
export const renderMarkdown = (source, { currentUserId = 0 } = {}) => {
  addLinkHook();
  const html = markdown.parse(source ?? '');
  const sanitized = DOMPurify.sanitize(html, { USE_PROFILES: { html: true } });
  if (!currentUserId) return sanitized;
  return sanitized.replaceAll(`class="mention" data-user-id="${currentUserId}"`, `class="mention mention-self" data-user-id="${currentUserId}"`);
};
