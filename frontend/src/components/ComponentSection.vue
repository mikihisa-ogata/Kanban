<template>
  <!-- スペースの設定の「コンポーネント」。Jira と同じく、名前・説明・既定の担当者を持つ機能区分を追加・編集・削除する -->
  <section class="bg-surface rounded-2xl px-8 py-6 shadow-xl">
    <div class="flex items-start justify-between gap-4 mb-1">
      <h3 class="m-0 text-lg font-bold text-fg">コンポーネント</h3>
      <button
        v-if="!form"
        type="button"
        class="shrink-0 px-3 py-1.5 bg-accent text-on-accent border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-accent-hover"
        @click="startCreate"
      >
        コンポーネントを作成
      </button>
    </div>
    <p class="m-0 mb-4 text-sm text-muted">
      課題を機能ごとに分類します。担当者が未割り当ての課題に既定の担当者のあるコンポーネントを付けると、その人が担当者になります。
    </p>

    <form v-if="form" class="mb-4 p-4 border border-line rounded-lg flex flex-col gap-3" @submit.prevent="handleSave">
      <div>
        <label for="component-name" class="block mb-1 text-sm font-semibold text-fg">名前 <span class="text-danger">*</span></label>
        <input
          id="component-name"
          v-model="form.name"
          type="text"
          maxlength="100"
          placeholder="例: フロントエンド"
          class="w-full px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface focus:outline-none focus:border-accent"
        />
      </div>
      <div>
        <label for="component-description" class="block mb-1 text-sm font-semibold text-fg">説明</label>
        <textarea
          id="component-description"
          v-model="form.description"
          rows="2"
          class="w-full px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface focus:outline-none focus:border-accent"
        ></textarea>
      </div>
      <div>
        <label for="component-default-assignee" class="block mb-1 text-sm font-semibold text-fg">既定の担当者</label>
        <div class="max-w-72 pl-2">
          <UserPicker
            id="component-default-assignee"
            :user="userOf(form.defaultAssigneeId)"
            empty-label="なし"
            @select="form.defaultAssigneeId = $event || 0"
          />
        </div>
      </div>
      <p v-if="formError" class="m-0 text-sm text-danger">{{ formError }}</p>
      <div class="flex justify-end gap-2">
        <button type="button" class="px-3 py-1.5 bg-raised text-fg border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-line" @click="form = null">キャンセル</button>
        <button
          type="submit"
          class="px-3 py-1.5 bg-accent text-on-accent border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="!!formError || busy"
        >
          {{ form.id ? '保存' : '作成' }}
        </button>
      </div>
    </form>

    <p v-if="loading" class="m-0 text-sm text-muted">読み込み中...</p>
    <table v-else-if="spaceComponents.length" class="w-full border-collapse text-sm">
      <thead>
        <tr class="text-left text-xs text-muted">
          <th class="py-2 pr-3 font-semibold">コンポーネント</th>
          <th class="py-2 pr-3 font-semibold">説明</th>
          <th class="py-2 pr-3 font-semibold">既定の担当者</th>
          <th class="py-2 pr-3 font-semibold text-right">課題</th>
          <th class="py-2"><span class="sr-only">操作</span></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="component in spaceComponents" :key="component.ID" class="border-t border-line">
          <td class="py-2 pr-3 font-semibold text-fg">{{ component.Name }}</td>
          <td class="py-2 pr-3 text-muted whitespace-pre-line">{{ component.Description || '-' }}</td>
          <td class="py-2 pr-3">
            <span v-if="component.DefaultAssignee" class="flex items-center gap-2 text-fg">
              <UserAvatar :user="component.DefaultAssignee" size="sm" />
              <span class="truncate">{{ component.DefaultAssignee.DisplayName }}</span>
            </span>
            <span v-else class="text-muted">なし</span>
          </td>
          <td class="py-2 pr-3 text-right text-muted">{{ usage(component) }}</td>
          <td class="py-2 text-right whitespace-nowrap">
            <button type="button" class="px-2 py-1 bg-transparent text-fg border-0 rounded text-xs font-semibold cursor-pointer hover:bg-raised" @click="startEdit(component)">編集</button>
            <button type="button" class="px-2 py-1 bg-transparent text-danger border-0 rounded text-xs font-semibold cursor-pointer hover:bg-danger/10" @click="deleting = component">削除</button>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-else class="m-0 text-sm text-muted">コンポーネントはまだありません。</p>

    <p v-if="error" class="mt-3 mb-0 text-sm text-danger">{{ error }}</p>

    <ConfirmDialog
      v-if="deleting"
      title="コンポーネントを削除しますか？"
      :message="`「${deleting.Name}」を削除します。\nこのコンポーネントの課題は削除されず、コンポーネントから外れます。`"
      @confirm="handleDelete"
      @cancel="deleting = null"
    />
  </section>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import ConfirmDialog from './ConfirmDialog.vue';
import UserAvatar from './UserAvatar.vue';
import UserPicker from './UserPicker.vue';
import { componentsApi } from '../api/components';
import { usersApi } from '../api/users';

const props = defineProps({
  space: {
    type: Object,
    required: true
  },
  // すべての課題（コンポーネントごとの件数を出す）
  todos: {
    type: Array,
    default: () => []
  }
});

const components = ref([]);
const users = ref([]);
const loading = ref(true);
const busy = ref(false);
const error = ref('');

const spaceComponents = computed(() => components.value.filter(c => c.SpaceID === props.space.ID));
const usage = (component) => props.todos.filter(todo => (todo.ComponentIDs || []).includes(component.ID)).length;
const userOf = (id) => users.value.find(user => user.ID === id) || null;

const fetchComponents = async () => {
  try {
    [components.value, users.value] = await Promise.all([componentsApi.fetchAll(), usersApi.fetchAll()]);
  } catch (err) {
    error.value = 'コンポーネントの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch components:', err);
  }
};

// 失敗したらメッセージを出す。どちらの場合もコンポーネントを取り直す
const run = async (label, action) => {
  if (busy.value) return false;
  busy.value = true;
  error.value = '';
  try {
    await action();
    return true;
  } catch (err) {
    error.value = `${label}に失敗しました: ${err.message}`;
    console.error(`Failed: ${label}`, err);
    return false;
  } finally {
    busy.value = false;
    await fetchComponents();
  }
};

// 作成・編集のフォーム（id が null なら作成）
const form = ref(null);

const formError = computed(() => {
  if (!form.value) return '';
  const name = form.value.name.trim();
  if (!name) return '名前を入力してください';
  if (name.length > 100) return '名前は 100 文字以内で入力してください';
  if (form.value.description.trim().length > 1000) return '説明は 1000 文字以内で入力してください';
  if (spaceComponents.value.some(c => c.ID !== form.value.id && c.Name.toLowerCase() === name.toLowerCase())) {
    return '同じ名前のコンポーネントが既にあります';
  }
  return '';
});

const startCreate = () => {
  form.value = { id: null, name: '', description: '', defaultAssigneeId: 0 };
};

const startEdit = (component) => {
  form.value = { id: component.ID, name: component.Name, description: component.Description, defaultAssigneeId: component.DefaultAssigneeID };
};

const handleSave = async () => {
  if (formError.value) return;
  const { id, ...fields } = form.value;
  const ok = id
    ? await run('コンポーネントの更新', () => componentsApi.update(id, fields))
    : await run('コンポーネントの作成', () => componentsApi.create(props.space.ID, fields));
  if (ok) form.value = null;
};

// 削除
const deleting = ref(null);

const handleDelete = async () => {
  const target = deleting.value;
  deleting.value = null;
  await run('コンポーネントの削除', () => componentsApi.delete(target.ID));
};

onMounted(async () => {
  await fetchComponents();
  loading.value = false;
});
</script>
