<template>
  <p class="px-8 py-8 text-fg">{{ message }}</p>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { spacesApi } from '~/api/spaces';
import { boardPath, lastBoardKey } from '~/utils/spaceRoutes';

const message = ref('読み込み中...');

// 最後に開いたボードへ移る。覚えていない（または消えた）スペースなら先頭のスペース、スペースがなければ全スペースのボード
onMounted(async () => {
  try {
    const spaces = await spacesApi.fetchAll();
    const last = lastBoardKey();
    let target;
    if (last === '') {
      target = null;
    } else {
      target = spaces.find(space => space.Key === last) || spaces.find(space => space.Key) || null;
    }
    await navigateTo(boardPath(target), { replace: true });
  } catch (err) {
    message.value = 'スペースの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch spaces:', err);
  }
});
</script>
