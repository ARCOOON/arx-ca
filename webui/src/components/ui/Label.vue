<script setup lang="ts">
import { cn } from '@/utils/cn'

const props = defineProps<{
  class?: string
  for?: string
}>()
</script>

<template>
  <label
    :for="props.for"
    :class="cn('block text-sm font-medium text-foreground leading-none', props.class)"
  >
    <slot />
  </label>
</template>
