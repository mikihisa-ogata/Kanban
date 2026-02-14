<template>
  <div id="app" class="min-h-screen bg-gradient-to-br from-blue-500 to-blue-800">
    <header class="bg-white bg-opacity-95 backdrop-blur-md px-8 py-10 text-center shadow-lg border-b-4 border-blue-400">
      <h1 class="text-5xl font-black mb-3 bg-gradient-to-r from-blue-600 to-blue-900 bg-clip-text text-transparent">
        📋 TODO カンバンボード
      </h1>
      <p class="text-lg text-gray-600 font-medium">
        タスクをドラッグ&ドロップで管理
      </p>
    </header>
    <KanbanBoard />
  </div>
</template>

<script setup>
import KanbanBoard from './components/KanbanBoard.vue';
</script>
