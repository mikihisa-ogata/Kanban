<template>
  <div class="task-form">
    <h3 class="form-title">新しいタスクを作成</h3>
    <form @submit.prevent="handleSubmit">
      <div class="form-group">
        <label for="title">タイトル <span class="required">*</span></label>
        <input
          id="title"
          v-model="formData.title"
          type="text"
          placeholder="タスクのタイトルを入力"
          required
        />
      </div>
      
      <div class="form-group">
        <label for="deadline">期限 <span class="required">*</span></label>
        <input
          id="deadline"
          v-model="formData.deadline"
          type="date"
          required
        />
      </div>
      
      <div class="form-group">
        <label for="status">ステータス</label>
        <select id="status" v-model="formData.status">
          <option value="Open">オープン</option>
          <option value="InProgress">進行中</option>
          <option value="Waiting">待ち状況</option>
          <option value="Closed">クローズ</option>
        </select>
      </div>
      
      <div class="form-actions">
        <button type="submit" class="btn-primary" :disabled="isSubmitting">
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

<style scoped>
.task-form {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  margin-bottom: 24px;
}

.form-title {
  margin: 0 0 20px 0;
  font-size: 20px;
  font-weight: 700;
  color: #1f2937;
}

.form-group {
  margin-bottom: 16px;
}

label {
  display: block;
  margin-bottom: 6px;
  font-size: 14px;
  font-weight: 600;
  color: #374151;
}

.required {
  color: #ef4444;
}

input[type="text"],
input[type="date"],
select {
  width: 100%;
  padding: 10px 12px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  font-size: 14px;
  transition: all 0.2s ease;
  font-family: inherit;
  color: #1f2937;
  background-color: white;
}

input[type="text"]:focus,
input[type="date"]:focus,
select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

select {
  cursor: pointer;
}

.form-actions {
  margin-top: 20px;
}

.btn-primary {
  width: 100%;
  padding: 12px 24px;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
}

.btn-primary:active:not(:disabled) {
  transform: translateY(0);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
