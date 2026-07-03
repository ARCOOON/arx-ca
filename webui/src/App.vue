<script setup lang="ts">
import { onMounted } from 'vue'
import { initTheme } from '@/composables/useTheme'

onMounted(() => {
  initTheme()
})
</script>

<template>
  <RouterView />
</template>
