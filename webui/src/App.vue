<script setup lang="ts">
import { RouterView } from 'vue-router'
import { Toaster } from 'vue-sonner'
import 'vue-sonner/style.css'
</script>

<template>
  <RouterView />
  <Toaster
    position="top-right"
    :toast-options="{
      style: {
        background: 'var(--color-surface)',
        color: 'var(--color-text)',
        border: '1px solid var(--color-border)',
        borderRadius: 'var(--radius-surface)',
        boxShadow: 'none',
      },
    }"
  />
</template>
