<template>
  <form class="flex flex-col gap-4" @submit.prevent="submit">
    <h2 class="m-0 text-lg font-bold text-gray-900 text-center">サインアップ</h2>
    <div>
      <label for="displayName" class="block mb-2 text-sm font-semibold text-gray-700">表示名</label>
      <input
        id="displayName"
        v-model="displayName"
        type="text"
        autocomplete="name"
        maxlength="50"
        required
        class="w-full px-3 py-2.5 border-2 border-gray-300 rounded-lg text-sm transition-all duration-200 text-gray-900 bg-white focus:outline-none focus:border-blue-500 focus:shadow-sm focus:shadow-blue-100"
      />
    </div>
    <div>
      <label for="email" class="block mb-2 text-sm font-semibold text-gray-700">メールアドレス</label>
      <input
        id="email"
        v-model="email"
        type="email"
        autocomplete="email"
        required
        class="w-full px-3 py-2.5 border-2 border-gray-300 rounded-lg text-sm transition-all duration-200 text-gray-900 bg-white focus:outline-none focus:border-blue-500 focus:shadow-sm focus:shadow-blue-100"
      />
    </div>
    <div>
      <label for="password" class="block mb-2 text-sm font-semibold text-gray-700">パスワード（8 文字以上）</label>
      <input
        id="password"
        v-model="password"
        type="password"
        autocomplete="new-password"
        minlength="8"
        required
        class="w-full px-3 py-2.5 border-2 border-gray-300 rounded-lg text-sm transition-all duration-200 text-gray-900 bg-white focus:outline-none focus:border-blue-500 focus:shadow-sm focus:shadow-blue-100"
      />
    </div>
    <p v-if="error" class="m-0 text-sm text-red-600" role="alert">{{ error }}</p>
    <button
      type="submit"
      :disabled="submitting"
      class="w-full py-2.5 bg-blue-600 text-white border-0 rounded-md text-sm font-semibold cursor-pointer hover:bg-blue-700 transition-colors duration-150 disabled:opacity-50 disabled:cursor-not-allowed"
    >
      {{ submitting ? '登録中...' : '登録してログイン' }}
    </button>
    <p class="m-0 text-sm text-gray-600 text-center">
      アカウントがある場合は
      <NuxtLink :to="{ path: '/login', query: route.query }" class="text-blue-600 font-semibold hover:underline">ログイン</NuxtLink>
    </p>
  </form>
</template>

<script setup>
import { ref } from 'vue';
import { authApi } from '~/api/auth';
import { useCurrentUser } from '~/composables/useCurrentUser';
import { safeRedirect } from '~/utils/authRedirect';

definePageMeta({ layout: 'auth' });
useHead({ title: 'サインアップ - Kanban' });

const route = useRoute();
const { user, loaded } = useCurrentUser();
const displayName = ref('');
const email = ref('');
const password = ref('');
const error = ref('');
const submitting = ref(false);

const submit = async () => {
  submitting.value = true;
  error.value = '';
  try {
    user.value = await authApi.signup(displayName.value, email.value, password.value);
    loaded.value = true;
    await navigateTo(safeRedirect(route.query.redirect), { replace: true });
  } catch (err) {
    error.value = err.message;
    console.error('Failed to sign up:', err);
  } finally {
    submitting.value = false;
  }
};
</script>
