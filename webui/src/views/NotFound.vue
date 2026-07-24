<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { Button } from '@/components/ui/button'
</script>

<template>
  <div class="flex min-h-screen flex-col items-center justify-center gap-4 bg-background px-4 text-center">
    <p class="text-6xl font-bold text-primary">404</p>
    <h1 class="text-lg font-semibold">Page not found</h1>
    <p class="max-w-sm text-sm text-muted-foreground">
      The page you are looking for does not exist or has been moved.
    </p>
    <Button as-child>
      <RouterLink :to="{ name: 'dashboard' }">Back to dashboard</RouterLink>
    </Button>
  </div>
</template>
