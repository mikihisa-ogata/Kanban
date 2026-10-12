<template>
  <!-- バージョンの詳細（Jira のリリースの詳細）。含まれる課題と進捗、編集・リリース・アーカイブ・削除 -->
  <SpaceShell :spaces="spaces" :todos="todos" :current-space="space" @changed="fetchAll">
    <div class="px-8 py-8">
      <p v-if="loading" class="m-0 text-white">読み込み中...</p>
      <div v-else-if="!space || !version" class="max-w-3xl bg-white rounded-2xl px-8 py-6 shadow-xl text-gray-700">
        バージョンが見つかりません。
        <NuxtLink v-if="space" :to="releasesPath(space)" class="ml-2 text-blue-600 font-semibold">リリースの一覧へ戻る</NuxtLink>
      </div>

      <template v-else>
        <NuxtLink :to="releasesPath(space)" class="text-sm font-semibold text-white/90 no-underline hover:underline">← リリース</NuxtLink>

        <section class="mt-3 max-w-6xl bg-white rounded-2xl px-6 py-5 shadow-xl">
          <form v-if="editing" class="flex flex-col gap-3" @submit.prevent="handleSave">
            <VersionFields v-model="form" id-prefix="edit-version" />
            <p v-if="formError" class="m-0 text-sm text-red-600">{{ formError }}</p>
            <div class="flex justify-end gap-2">
              <button type="button" :class="secondaryButton" @click="editing = false">キャンセル</button>
              <button type="submit" :class="primaryButton" :disabled="!!formError || saving">保存</button>
            </div>
          </form>

          <template v-else>
            <div class="flex flex-wrap items-center gap-3">
              <h2 class="m-0 text-xl font-bold text-gray-900">{{ version.Name }}</h2>
              <span :class="['px-2 py-0.5 rounded text-xs font-semibold', VERSION_STATUS_BADGE_CLASS[version.Status]]">
                {{ VERSION_STATUS_LABEL[version.Status] }}
              </span>
              <div class="ml-auto flex flex-wrap items-center gap-2">
                <button type="button" :class="secondaryButton" @click="startEdit">編集</button>
                <button v-if="version.Status === VERSION_UNRELEASED" type="button" :class="primaryButton" @click="openRelease">リリース</button>
                <button v-if="version.Status === VERSION_RELEASED" type="button" :class="secondaryButton" @click="changeStatus(VERSION_UNRELEASED, '未リリースに戻す')">未リリースに戻す</button>
                <button v-if="version.Status !== VERSION_ARCHIVED" type="button" :class="secondaryButton" @click="changeStatus(VERSION_ARCHIVED, 'アーカイブ')">アーカイブ</button>
                <button v-else type="button" :class="secondaryButton" @click="changeStatus(unarchivedStatus, 'アーカイブの解除')">アーカイブを解除</button>
                <button type="button" :class="dangerButton" @click="deleting = true">削除</button>
              </div>
            </div>
            <dl class="mt-3 mb-0 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
              <dt class="text-gray-500">開始日</dt>
              <dd class="m-0 text-gray-900">{{ version.StartDate || '-' }}</dd>
              <dt class="text-gray-500">リリース日</dt>
              <dd class="m-0 text-gray-900">{{ version.ReleaseDate || '-' }}</dd>
            </dl>
            <p v-if="version.Description" class="mt-3 mb-0 text-sm text-gray-700 whitespace-pre-line">{{ version.Description }}</p>
          </template>

          <div class="mt-5">
            <VersionProgress :progress="progress" show-counts />
          </div>
        </section>

        <!-- 含まれる課題 -->
        <section class="mt-4 max-w-6xl bg-white rounded-2xl shadow-xl overflow-hidden">
          <h3 class="m-0 px-5 py-3 border-b border-gray-200 text-base font-bold text-gray-900">課題（{{ issues.length }} 件）</h3>
          <ol class="m-0 p-0 list-none">
            <li v-for="issue in issues" :key="issue.ID" class="flex items-center gap-3 px-5 py-2 border-b border-gray-100 text-sm">
              <IssueTypeIcon :type="issue.IssueType" />
              <NuxtLink :to="issuePath(issue.Key)" class="shrink-0 w-20 font-mono text-xs font-semibold text-gray-500 no-underline hover:text-blue-600 hover:underline">{{ issue.Key }}</NuxtLink>
              <span class="flex-1 min-w-0 truncate text-gray-900" :title="issue.Title">{{ issue.Title }}</span>
              <span :class="['shrink-0 max-w-32 truncate px-2 py-0.5 rounded text-xs font-semibold', STATUS_CATEGORY_BADGE_CLASS[issue.StatusCategory]]">
                {{ issue.StatusName }}
              </span>
              <span class="shrink-0 w-6 flex justify-center">
                <UserAvatar v-if="issue.Assignee" :user="issue.Assignee" size="sm" :title="`担当者: ${issue.Assignee.DisplayName}`" />
              </span>
            </li>
            <li v-if="!issues.length" class="px-5 py-6 text-center text-sm text-gray-500">
              このバージョンの課題はありません。課題の詳細の「修正バージョン」で付けられます。
            </li>
          </ol>
        </section>
      </template>
    </div>

    <!-- リリースの確認。未完了の課題があれば移し先を選ぶ -->
    <div v-if="releasing" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50" @click.self="releasing = false">
      <form
        role="dialog"
        aria-modal="true"
        aria-labelledby="release-version-title"
        class="bg-white rounded-2xl px-8 pt-7 pb-6 shadow-2xl mx-4 max-w-md w-full flex flex-col gap-4"
        @submit.prevent="handleRelease"
      >
        <h3 id="release-version-title" class="m-0 text-lg font-bold text-gray-900">「{{ version.Name }}」をリリース</h3>
        <div>
          <label for="release-date" class="block mb-1 text-sm font-semibold text-gray-700">リリース日</label>
          <input
            id="release-date"
            v-model="releaseDate"
            type="date"
            class="px-3 py-2 border-2 border-gray-300 rounded-lg text-sm text-gray-900 bg-white focus:outline-none focus:border-blue-500"
          />
        </div>
        <template v-if="openIssues.length">
          <p class="m-0 text-sm text-gray-700">このバージョンには完了していない課題が {{ openIssues.length }} 件あります。</p>
          <div>
            <label for="release-move-to" class="block mb-1 text-sm font-semibold text-gray-700">完了していない課題</label>
            <select
              id="release-move-to"
              v-model="moveTo"
              class="w-full px-3 py-2 border-2 border-gray-300 rounded-lg text-sm text-gray-800 bg-white cursor-pointer focus:outline-none focus:border-blue-500"
            >
              <option v-for="target in moveTargets" :key="target.ID" :value="target.ID">「{{ target.Name }}」へ移す</option>
              <option :value="0">移さずにリリースする（このバージョンのまま）</option>
            </select>
          </div>
        </template>
        <p v-else class="m-0 text-sm text-gray-700">すべての課題が完了しています。</p>
        <div class="flex justify-end gap-3">
          <button type="button" :class="secondaryButton" @click="releasing = false">キャンセル</button>
          <button type="submit" :class="primaryButton" :disabled="saving">リリース</button>
        </div>
      </form>
    </div>

    <ConfirmDialog
      v-if="deleting"
      title="バージョンを削除しますか？"
      :message="`「${version.Name}」を削除します。\nこのバージョンの課題は削除されず、修正バージョンから外れます。`"
      @confirm="handleDelete"
      @cancel="deleting = false"
    />

    <div v-if="error" class="fixed bottom-8 right-8 bg-red-100 text-red-800 px-6 py-4 rounded-xl shadow-xl max-w-96 text-sm font-semibold border-l-4 border-red-500 animate-slideIn">
      {{ error }}
    </div>
  </SpaceShell>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import SpaceShell from '~/components/SpaceShell.vue';
import ConfirmDialog from '~/components/ConfirmDialog.vue';
import IssueTypeIcon from '~/components/IssueTypeIcon.vue';
import UserAvatar from '~/components/UserAvatar.vue';
import VersionFields from '~/components/VersionFields.vue';
import VersionProgress from '~/components/VersionProgress.vue';
import { todosApi } from '~/api/todos';
import { spacesApi } from '~/api/spaces';
import { versionsApi } from '~/api/versions';
import { releasesPath, issuePath } from '~/utils/spaceRoutes';
import { STATUS_CATEGORY_BADGE_CLASS, isDoneCategory } from '~/utils/statuses';
import { dateAfter } from '~/utils/sprints';
import {
  VERSION_UNRELEASED, VERSION_RELEASED, VERSION_ARCHIVED, VERSION_STATUS_LABEL, VERSION_STATUS_BADGE_CLASS, versionProgress, issuesOfVersion
} from '~/utils/versions';
import { versionFieldsError } from '~/utils/versionFields';

const route = useRoute();
const spaceKey = computed(() => String(route.params.key));
const versionId = computed(() => Number(route.params.id));

const spaces = ref([]);
const todos = ref([]);
const versions = ref([]);
const loading = ref(true);
const error = ref('');
const saving = ref(false);

const space = computed(() => spaces.value.find(s => s.Key === spaceKey.value) || null);
// URL のスペースのバージョンだけを出す
const version = computed(() =>
  space.value ? versions.value.find(v => v.ID === versionId.value && v.SpaceID === space.value.ID) || null : null
);
useHead(() => ({ title: version.value ? `${version.value.Name} - リリース - Kanban` : 'Kanban' }));

const issues = computed(() => (version.value ? issuesOfVersion(todos.value, version.value.ID) : []));
const openIssues = computed(() => issues.value.filter(issue => !isDoneCategory(issue.StatusCategory)));
const progress = computed(() => versionProgress(issues.value));

const primaryButton = 'px-3 py-1.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed';
const secondaryButton = 'px-3 py-1.5 bg-gray-100 text-gray-700 border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-gray-200';
const dangerButton = 'px-3 py-1.5 bg-transparent text-red-600 border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-red-50';

const fetchAll = async () => {
  try {
    error.value = '';
    [spaces.value, todos.value, versions.value] = await Promise.all([spacesApi.fetchAll(), todosApi.fetchAll(), versionsApi.fetchAll()]);
  } catch (err) {
    error.value = 'バージョンの取得に失敗しました: ' + err.message;
    console.error('Failed to fetch version:', err);
  }
};

// 失敗したらメッセージを出す。どちらの場合もバージョンと課題を取り直す
const run = async (label, action) => {
  saving.value = true;
  try {
    error.value = '';
    await action();
    return true;
  } catch (err) {
    error.value = `${label}に失敗しました: ${err.message}`;
    console.error(`Failed: ${label}`, err);
    return false;
  } finally {
    saving.value = false;
    await fetchAll();
  }
};

const fieldsOf = (v) => ({ name: v.Name, description: v.Description, startDate: v.StartDate, releaseDate: v.ReleaseDate });

// 編集
const editing = ref(false);
const form = ref({ name: '', description: '', startDate: '', releaseDate: '' });
const formError = computed(() => versionFieldsError(form.value));

const startEdit = () => {
  form.value = fieldsOf(version.value);
  editing.value = true;
};

const handleSave = async () => {
  if (formError.value) return;
  if (await run('バージョンの更新', () => versionsApi.update(version.value.ID, { ...form.value, status: version.value.Status }))) {
    editing.value = false;
  }
};

// 状態の変更（未リリースに戻す・アーカイブ・アーカイブの解除）
// アーカイブを解除したら、リリース日を過ぎていればリリース済み、そうでなければ未リリースに戻す
const unarchivedStatus = computed(() =>
  version.value?.ReleaseDate && version.value.ReleaseDate <= dateAfter(0) ? VERSION_RELEASED : VERSION_UNRELEASED
);

const changeStatus = (status, label) =>
  run(label, () => versionsApi.update(version.value.ID, { ...fieldsOf(version.value), status }));

// リリース。移し先の既定は、同じスペースの別の未リリースのバージョンの先頭（なければ移さない）
const releasing = ref(false);
const releaseDate = ref('');
const moveTo = ref(0);
const moveTargets = computed(() =>
  versions.value.filter(v => v.SpaceID === version.value.SpaceID && v.ID !== version.value.ID && v.Status === VERSION_UNRELEASED)
);

const openRelease = () => {
  releaseDate.value = version.value.ReleaseDate || dateAfter(0);
  moveTo.value = moveTargets.value[0]?.ID ?? 0;
  releasing.value = true;
};

const handleRelease = async () => {
  const target = openIssues.value.length ? moveTo.value : 0;
  if (await run('リリース', () => versionsApi.release(version.value.ID, { releaseDate: releaseDate.value, moveTo: target }))) {
    releasing.value = false;
  }
};

// 削除したら一覧へ戻る
const deleting = ref(false);

const handleDelete = async () => {
  deleting.value = false;
  const target = space.value;
  if (await run('バージョンの削除', () => versionsApi.delete(version.value.ID))) {
    await navigateTo(releasesPath(target));
  }
};

onMounted(async () => {
  await fetchAll();
  loading.value = false;
});
</script>
