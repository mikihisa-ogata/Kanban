<template>
  <!-- バージョンの名前・開始日・リリース日・説明の入力欄（作成と編集で共有） -->
  <div class="flex flex-col gap-3">
    <div>
      <label :for="`${idPrefix}-name`" class="block mb-1 text-sm font-semibold text-fg">名前 <span class="text-danger">*</span></label>
      <input
        :id="`${idPrefix}-name`"
        :value="modelValue.name"
        type="text"
        maxlength="100"
        placeholder="例: v1.0"
        class="w-full px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface focus:outline-none focus:border-accent"
        @input="update('name', $event.target.value)"
      />
    </div>
    <div class="flex flex-wrap gap-3">
      <div>
        <label :for="`${idPrefix}-start`" class="block mb-1 text-sm font-semibold text-fg">開始日</label>
        <input
          :id="`${idPrefix}-start`"
          :value="modelValue.startDate"
          type="date"
          class="px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface focus:outline-none focus:border-accent"
          @input="update('startDate', $event.target.value)"
        />
      </div>
      <div>
        <label :for="`${idPrefix}-release`" class="block mb-1 text-sm font-semibold text-fg">リリース日</label>
        <input
          :id="`${idPrefix}-release`"
          :value="modelValue.releaseDate"
          type="date"
          class="px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface focus:outline-none focus:border-accent"
          @input="update('releaseDate', $event.target.value)"
        />
      </div>
    </div>
    <div>
      <label :for="`${idPrefix}-description`" class="block mb-1 text-sm font-semibold text-fg">説明</label>
      <textarea
        :id="`${idPrefix}-description`"
        :value="modelValue.description"
        rows="2"
        maxlength="1000"
        class="w-full px-3 py-2 border-2 border-line rounded-lg text-sm text-fg bg-surface resize-y focus:outline-none focus:border-accent"
        @input="update('description', $event.target.value)"
      ></textarea>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  // { name, description, startDate, releaseDate }
  modelValue: {
    type: Object,
    required: true
  },
  // label と入力欄を結ぶ ID の接頭辞
  idPrefix: {
    type: String,
    required: true
  }
});

const emit = defineEmits(['update:modelValue']);

const update = (field, value) => {
  emit('update:modelValue', { ...props.modelValue, [field]: value });
};
</script>
