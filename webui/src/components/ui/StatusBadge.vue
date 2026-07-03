<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/utils/cn'

type Status = 'active' | 'revoked' | 'expired' | 'enabled' | 'disabled' | 'online' | 'offline' | string

const props = defineProps<{
  status: Status
  class?: string
}>()

const config = computed(() => {
  switch (props.status.toLowerCase()) {
    case 'active':
    case 'enabled':
    case 'online':
    case 'ok':
      return { dot: 'bg-success', text: 'text-success', label: props.status }
    case 'revoked':
    case 'disabled':
    case 'offline':
    case 'error':
      return { dot: 'bg-destructive', text: 'text-destructive', label: props.status }
    case 'expired':
    case 'warning':
      return { dot: 'bg-warning', text: 'text-warning-foreground', label: props.status }
    default:
      return { dot: 'bg-foreground-muted', text: 'text-foreground-muted', label: props.status }
  }
})
</script>

<template>
  <span :class="cn('inline-flex items-center gap-1.5 text-xs font-medium capitalize', config.text, props.class)">
    <span :class="cn('h-1.5 w-1.5 rounded-full', config.dot)" />
    {{ config.label }}
  </span>
</template>
