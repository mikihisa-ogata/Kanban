<template>
  <!-- Markdown を整形して表示する（説明欄。今後のコメントでも使う）。HTML はサニタイズ済みのものだけを入れる -->
  <div v-if="html !== null" class="markdown-body" v-html="html"></div>
  <!-- サーバーでの描画中はサニタイズできないので、元の文字列をそのまま出す -->
  <div v-else class="markdown-body whitespace-pre-wrap break-words">{{ source }}</div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { renderMarkdown } from '../utils/markdown';
import { useCurrentUser } from '../composables/useCurrentUser';

const props = defineProps({
  source: {
    type: String,
    default: ''
  }
});

const mounted = ref(false);
onMounted(() => {
  mounted.value = true;
});

// 自分へのメンションは目立つ色にする
const { user: currentUser } = useCurrentUser();

const html = computed(() => (mounted.value ? renderMarkdown(props.source, { currentUserId: currentUser.value?.ID ?? 0 }) : null));
</script>

<style scoped>
.markdown-body {
  font-size: 0.875rem;
  line-height: 1.7;
  color: var(--kb-fg);
  overflow-wrap: anywhere;
}
.markdown-body :deep(> :first-child) {
  margin-top: 0;
}
.markdown-body :deep(> :last-child) {
  margin-bottom: 0;
}
.markdown-body :deep(p),
.markdown-body :deep(ul),
.markdown-body :deep(ol),
.markdown-body :deep(blockquote),
.markdown-body :deep(pre),
.markdown-body :deep(table) {
  margin: 0 0 0.75rem;
}
.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4),
.markdown-body :deep(h5),
.markdown-body :deep(h6) {
  margin: 1.25rem 0 0.5rem;
  font-weight: 700;
  line-height: 1.4;
}
.markdown-body :deep(h1) {
  font-size: 1.375rem;
}
.markdown-body :deep(h2) {
  font-size: 1.2rem;
}
.markdown-body :deep(h3) {
  font-size: 1.05rem;
}
.markdown-body :deep(ul) {
  list-style: disc;
  padding-left: 1.5rem;
}
.markdown-body :deep(ol) {
  list-style: decimal;
  padding-left: 1.5rem;
}
.markdown-body :deep(li > ul),
.markdown-body :deep(li > ol) {
  margin-bottom: 0;
}
/* チェックボックス付きのリスト項目は、行頭の点を出さない */
.markdown-body :deep(li:has(> input[type='checkbox'])) {
  list-style: none;
  margin-left: -1.25rem;
}
.markdown-body :deep(input[type='checkbox']) {
  margin-right: 0.4rem;
  vertical-align: middle;
}
.markdown-body :deep(a) {
  color: var(--kb-accent);
  text-decoration: underline;
}
.markdown-body :deep(code) {
  padding: 0.1rem 0.35rem;
  border-radius: 0.25rem;
  background: var(--kb-raised);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.8125rem;
}
.markdown-body :deep(pre) {
  padding: 0.75rem 1rem;
  border-radius: 0.375rem;
  background: var(--kb-raised);
  overflow-x: auto;
}
.markdown-body :deep(pre code) {
  padding: 0;
  background: transparent;
}
.markdown-body :deep(blockquote) {
  padding-left: 0.75rem;
  border-left: 3px solid var(--kb-line);
  color: var(--kb-muted);
}
.markdown-body :deep(table) {
  display: block;
  max-width: 100%;
  overflow-x: auto;
  border-collapse: collapse;
}
.markdown-body :deep(th),
.markdown-body :deep(td) {
  padding: 0.35rem 0.75rem;
  border: 1px solid var(--kb-line);
}
.markdown-body :deep(th) {
  background: var(--kb-raised);
  font-weight: 600;
}
.markdown-body :deep(.mention) {
  padding: 0 0.3rem;
  border-radius: 0.75rem;
  background: var(--kb-line);
  color: var(--kb-fg);
  font-weight: 600;
  white-space: nowrap;
}
/* Jira と同じく、自分へのメンションは青い背景で目立たせる */
.markdown-body :deep(.mention-self) {
  background: var(--kb-accent);
  color: var(--kb-on-accent);
}
.markdown-body :deep(hr) {
  margin: 1rem 0;
  border: 0;
  border-top: 1px solid var(--kb-line);
}
</style>
