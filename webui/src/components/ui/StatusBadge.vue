<script setup lang="ts">
import { computed } from 'vue'

export type BadgeTone = 'valid' | 'revoked' | 'expired' | 'neutral' | 'enabled' | 'disabled'

const props = defineProps<{
  label: string
  tone?: BadgeTone
}>()

const toneClass = computed(() => {
  switch (props.tone ?? 'neutral') {
    case 'valid':
    case 'enabled':
      return 'border-emerald-800/80 bg-emerald-950/50 text-emerald-300'
    case 'revoked':
      return 'border-red-800/80 bg-red-950/50 text-red-300'
    case 'expired':
      return 'border-amber-800/80 bg-amber-950/50 text-amber-300'
    case 'disabled':
      return 'border-zinc-700 bg-zinc-900 text-zinc-500'
    default:
      return 'border-zinc-700 bg-zinc-900 text-zinc-400'
  }
})
</script>

<template>
  <span
    class="inline-flex items-center border px-2 py-0.5 text-[11px] font-medium uppercase tracking-wide"
    :class="toneClass"
  >
    {{ label }}
  </span>
</template>
