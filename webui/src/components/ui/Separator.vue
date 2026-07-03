<script setup lang="ts">
import { cn } from '@/utils/cn'

const props = withDefaults(
  defineProps<{
    orientation?: 'horizontal' | 'vertical'
    class?: string
  }>(),
  { orientation: 'horizontal' },
)
</script>

<template>
  <div
    :class="
      cn(
        'shrink-0 bg-border',
        props.orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px',
        props.class,
      )
    "
  />
</template>
