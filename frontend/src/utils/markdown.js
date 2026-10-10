import { marked } from 'marked';
import DOMPurify from 'dompurify';

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
// <script> やイベント属性などを除くため、必ず DOMPurify でサニタイズしてから返す。
// DOMPurify はブラウザの DOM を使うので、ブラウザでだけ呼ぶ
export const renderMarkdown = (source) => {
  addLinkHook();
  const html = marked.parse(source ?? '', { gfm: true, breaks: true, async: false });
  return DOMPurify.sanitize(html, { USE_PROFILES: { html: true } });
};
