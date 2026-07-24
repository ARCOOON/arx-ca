<script setup lang="ts">
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'

const props = defineProps<{
  status: 'valid' | 'revoked' | 'expired' | 'enabled' | 'disabled' | 'neutral'
  label?: string
}>()

const text = computed(() => props.label ?? props.status)

const toneClass = computed(() => {
  switch (props.status) {
    case 'valid':
    case 'enabled':
      return 'border-transparent bg-success/15 text-success'
    case 'revoked':
      return 'border-transparent bg-destructive/10 text-destructive'
    case 'expired':
    case 'disabled':
      return 'border-transparent bg-warning/15 text-warning'
    default:
      return 'border-transparent bg-muted text-muted-foreground'
  }
})
</script>

<template>
  <Badge variant="outline" class="capitalize" :class="toneClass">{{ text }}</Badge>
</template>
