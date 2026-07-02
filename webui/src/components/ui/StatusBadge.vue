<script setup lang="ts">
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

export type BadgeTone = 'valid' | 'revoked' | 'expired' | 'neutral' | 'enabled' | 'disabled'

const props = defineProps<{
  label: string
  tone?: BadgeTone
}>()

const toneClass = computed(() => {
  switch (props.tone ?? 'neutral') {
    case 'valid':
    case 'enabled':
      return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
    case 'revoked':
      return 'border-destructive/30 bg-destructive/10 text-destructive'
    case 'expired':
      return 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400'
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
    :class="cn('text-[11px] font-medium uppercase tracking-wide', toneClass)"
  >
    {{ label }}
  </Badge>
</template>
