<script setup lang="ts">
import { RouterView } from 'vue-router'
import { Toaster } from 'vue-sonner'
</script>

<template>
  <RouterView />
  <Toaster rich-colors close-button :duration="3000" :offset="80" position="bottom-right" />
</template>
