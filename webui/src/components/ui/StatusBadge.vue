<script setup lang="ts">
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

export type BadgeTone = 'valid' | 'revoked' | 'expired' | 'neutral' | 'enabled' | 'disabled'

const props = defineProps<{
  label: string
  tone?: BadgeTone
}>()

const badgeClass = computed(() => {
  switch (props.tone ?? 'neutral') {
    case 'valid':
    case 'enabled':
      return 'border-primary/30 bg-primary/10 text-foreground'
    case 'revoked':
      return 'border-destructive/40 bg-destructive/10 text-destructive'
    case 'expired':
      return 'border-chart-4/40 bg-chart-4/10 text-foreground'
    case 'disabled':
      return 'opacity-60'
    default:
      return ''
  }
})
</script>

<template>
  <Badge
    variant="outline"
    :class="cn('text-[11px] font-medium uppercase tracking-wide', badgeClass)"
  >
    {{ label }}
  </Badge>
</template>
