<template>
  <KanbanBoard :space-key="spaceKey" />
</template>

<script setup>
import { computed } from 'vue';
import KanbanBoard from '~/components/KanbanBoard.vue';

const route = useRoute();
const spaceKey = computed(() => String(route.params.key));
</script>
