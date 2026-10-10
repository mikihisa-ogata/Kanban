<template>
  <div class="px-8 py-8">
    <div class="max-w-3xl mx-auto">
      <NuxtLink to="/" class="inline-flex items-center gap-1 text-sm font-semibold text-white opacity-80 hover:opacity-100 no-underline">
        ← ボードに戻る
      </NuxtLink>

      <p v-if="loading" class="mt-6 text-white">読み込み中...</p>
      <div v-else-if="!space" class="mt-6 bg-white rounded-2xl px-8 py-6 shadow-xl text-gray-700">
        {{ saveError || 'スペースが見つかりません。' }}
      </div>

      <template v-else>
        <h2 class="mt-4 mb-6 text-2xl font-bold text-white">
          スペースの設定
          <span class="ml-2 text-base font-semibold opacity-80">{{ space.Title }}</span>
        </h2>

        <!-- 詳細（名前とキー） -->
        <section class="bg-white rounded-2xl px-8 py-6 shadow-xl">
          <h3 class="m-0 mb-4 text-lg font-bold text-gray-900">詳細</h3>
          <form class="flex flex-col gap-4" @submit.prevent="handleSave">
            <div>
              <label for="space-title" class="block mb-2 text-sm font-semibold text-gray-700">
                名前 <span class="text-red-500">*</span>
              </label>
              <input
                id="space-title"
                v-model="form.title"
                type="text"
                @input="saved = false"
                class="w-full px-3 py-2.5 border-2 border-gray-300 rounded-lg text-sm text-gray-900 bg-white focus:outline-none focus:border-blue-500"
              />
              <p v-if="!form.title.trim()" class="mt-1 mb-0 text-xs text-red-600">名前を入力してください</p>
            </div>

            <div>
              <label for="space-key" class="block mb-2 text-sm font-semibold text-gray-700">
                キー <span class="text-red-500">*</span>
              </label>
              <div class="flex items-center gap-3">
                <input
                  id="space-key"
                  :value="form.key"
                  type="text"
                  maxlength="10"
                  class="w-40 px-3 py-2.5 border-2 border-gray-300 rounded-lg font-mono text-sm uppercase text-gray-900 bg-white focus:outline-none focus:border-blue-500"
                  @input="handleKeyInput"
                />
                <span v-if="!keyError" class="text-sm text-gray-500">
                  課題キーの例: <span class="font-mono">{{ form.key }}-1</span>
                </span>
              </div>
              <p v-if="keyError" class="mt-1 mb-0 text-xs text-red-600">{{ keyError }}</p>
              <p v-else-if="form.key !== space.Key" class="mt-1 mb-0 text-xs text-amber-700">
                キーを変えると、このスペースのタスクの課題キーがすべて変わります（{{ space.Key }}-1 → {{ form.key }}-1）。
              </p>
            </div>

            <p v-if="saveError" class="m-0 text-sm text-red-600">{{ saveError }}</p>
            <p v-if="saved" class="m-0 text-sm text-emerald-600 font-semibold">保存しました</p>

            <div class="flex justify-end">
              <button
                type="submit"
                :disabled="!canSave"
                class="py-2.5 px-8 bg-gradient-to-r from-blue-500 to-blue-600 text-white border-0 rounded-lg text-sm font-semibold cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {{ saving ? '保存中...' : '保存' }}
              </button>
            </div>
          </form>
        </section>

        <!-- スペースの削除 -->
        <section class="mt-6 bg-white rounded-2xl px-8 py-6 shadow-xl">
          <h3 class="m-0 mb-2 text-lg font-bold text-gray-900">スペースを削除</h3>
          <p class="m-0 mb-4 text-sm text-gray-600">
            エピックとタスクは削除されず、スペース未割り当てになります（課題キーは TASK-番号 に変わります）。
          </p>
          <p v-if="deleteError" class="m-0 mb-4 text-sm text-red-600">{{ deleteError }}</p>
          <button
            type="button"
            class="py-2 px-5 bg-white text-red-600 border-2 border-red-300 rounded-lg text-sm font-semibold cursor-pointer hover:bg-red-50"
            @click="confirmingDelete = true"
          >
            スペースを削除
          </button>
        </section>
      </template>
    </div>

    <ConfirmDialog
      v-if="confirmingDelete && space"
      title="スペースを削除しますか？"
      :message="`「${space.Title}」を削除します。\nエピックとタスクは削除されず、スペース未割り当てになります。`"
      @confirm="handleDelete"
      @cancel="confirmingDelete = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import ConfirmDialog from '~/components/ConfirmDialog.vue';
import { spacesApi } from '~/api/spaces';
import { normalizeSpaceKey, spaceKeyError } from '~/utils/spaceKey';

const route = useRoute();
const spaceId = Number(route.params.id);

// ボードで選択中のスペース（KanbanBoard と共有）。削除したら「すべて」に戻す
const selectedSpaceId = useState('selectedSpaceId', () => null);

const spaces = ref([]);
const loading = ref(true);
const form = ref({ title: '', key: '' });
const saving = ref(false);
const saved = ref(false);
const saveError = ref('');
const confirmingDelete = ref(false);
const deleteError = ref('');

const space = computed(() => spaces.value.find(s => s.ID === spaceId) || null);

const keyError = computed(() => spaceKeyError(form.value.key, spaces.value, spaceId));

const canSave = computed(() =>
  !saving.value &&
  form.value.title.trim() !== '' &&
  !keyError.value &&
  (form.value.title.trim() !== space.value.Title || form.value.key !== space.value.Key)
);

const fetchSpaces = async () => {
  spaces.value = await spacesApi.fetchAll();
  if (space.value) {
    form.value = { title: space.value.Title, key: space.value.Key };
  }
};

const handleKeyInput = (event) => {
  form.value.key = normalizeSpaceKey(event.target.value);
  event.target.value = form.value.key;
  saved.value = false;
};

const handleSave = async () => {
  if (!canSave.value) return;
  saving.value = true;
  saved.value = false;
  saveError.value = '';
  try {
    await spacesApi.update(spaceId, { title: form.value.title.trim(), key: form.value.key });
    await fetchSpaces();
    saved.value = true;
  } catch (err) {
    saveError.value = '保存に失敗しました: ' + err.message;
    console.error('Failed to update space:', err);
  } finally {
    saving.value = false;
  }
};

const handleDelete = async () => {
  confirmingDelete.value = false;
  deleteError.value = '';
  try {
    await spacesApi.delete(spaceId);
    if (selectedSpaceId.value === spaceId) {
      selectedSpaceId.value = null;
    }
    await navigateTo('/');
  } catch (err) {
    deleteError.value = '削除に失敗しました: ' + err.message;
    console.error('Failed to delete space:', err);
  }
};

onMounted(async () => {
  try {
    await fetchSpaces();
  } catch (err) {
    saveError.value = 'スペースの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch spaces:', err);
  } finally {
    loading.value = false;
  }
});
</script>
