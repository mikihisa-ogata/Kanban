<template>
  <div id="app">
    <header class="app-header">
      <h1 class="app-title">📋 TODO カンバンボード</h1>
      <p class="app-subtitle">タスクをドラッグ&ドロップで管理</p>
    </header>
    <KanbanBoard />
  </div>
</template>

<script setup>
import KanbanBoard from './components/KanbanBoard.vue';
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
    'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
    sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  background: linear-gradient(135deg, #3b82f6 0%, #1e40af 100%);
  min-height: 100vh;
}

#app {
  min-height: 100vh;
}

.app-header {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  padding: 32px 24px;
  text-align: center;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.app-title {
  font-size: 36px;
  font-weight: 800;
  background: linear-gradient(135deg, #3b82f6 0%, #1e40af 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: 8px;
}

.app-subtitle {
  font-size: 16px;
  color: #6b7280;
  font-weight: 500;
}

/* スクロールバーのグローバルスタイリング */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.05);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.3);
}
</style>
