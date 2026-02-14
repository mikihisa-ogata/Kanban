<template>
  <div class="bg-white rounded-2xl p-8 shadow-lg mx-auto max-w-2xl">
    <h3 class="m-0 mb-6 text-2xl font-bold text-gray-900">新しいタスクを作成</h3>
    <form @submit.prevent="handleSubmit">
      <div class="mb-6">
        <label for="title" class="block mb-2 text-sm font-semibold text-gray-700">
          タイトル <span class="text-red-500">*</span>
        </label>
        <input
          id="title"
          v-model="formData.title"
          type="text"
          placeholder="タスクのタイトルを入力"
          class="w-full px-3 py-2.5 border-2 border-gray-300 rounded-lg text-sm transition-all duration-200 text-gray-900 bg-white focus:outline-none focus:border-blue-500 focus:shadow-sm focus:shadow-blue-100"
          required
        />
      </div>
      
      <div class="mb-6">
        <label for="deadline" class="block mb-2 text-sm font-semibold text-gray-700">
          期限 <span class="text-red-500">*</span>
        </label>
        <input
          id="deadline"
          v-model="formData.deadline"
          type="date"
          class="w-full px-3 py-2.5 border-2 border-gray-300 rounded-lg text-sm transition-all duration-200 text-gray-900 bg-white focus:outline-none focus:border-blue-500 focus:shadow-sm focus:shadow-blue-100"
          required
        />
      </div>
      
      <div class="mb-7">
        <label for="status" class="block mb-2 text-sm font-semibold text-gray-700">
          ステータス
        </label>
        <select 
          id="status" 
          v-model="formData.status"
          class="w-full px-3 py-2.5 border-2 border-gray-300 rounded-lg text-sm transition-all duration-200 text-gray-900 bg-white cursor-pointer focus:outline-none focus:border-blue-500 focus:shadow-sm focus:shadow-blue-100"
        >
          <option value="Open">オープン</option>
          <option value="InProgress">進行中</option>
          <option value="Waiting">待ち状況</option>
          <option value="Closed">クローズ</option>
        </select>
      </div>
      
      <div class="mt-5">
        <button 
          type="submit" 
          :disabled="isSubmitting"
          class="w-full py-3 px-6 bg-gradient-to-r from-blue-500 to-blue-600 text-white border-0 rounded-lg text-base font-semibold cursor-pointer transition-all duration-200 hover:enabled:-translate-y-0.5 hover:enabled:shadow-lg hover:enabled:shadow-blue-400 active:enabled:translate-y-0 active:enabled:shadow-md disabled:opacity-60 disabled:cursor-not-allowed"
        >
          {{ isSubmitting ? '作成中...' : 'タスクを作成' }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { ref, defineEmits } from 'vue';

const emit = defineEmits(['submit']);

const formData = ref({
  title: '',
  deadline: '',
  status: 'Open'
});

const isSubmitting = ref(false);

const handleSubmit = async () => {
  if (isSubmitting.value) return;
  
  isSubmitting.value = true;
  try {
    await emit('submit', { ...formData.value });
    // フォームをリセット
    formData.value = {
      title: '',
      deadline: '',
      status: 'Open'
    };
  } finally {
    isSubmitting.value = false;
  }
};
</script>
