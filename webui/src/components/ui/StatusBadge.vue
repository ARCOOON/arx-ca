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
      return 'ui-badge-valid'
    case 'revoked':
      return 'ui-badge-revoked'
    case 'expired':
      return 'ui-badge-expired'
    case 'disabled':
      return 'ui-badge-neutral opacity-60'
    default:
      return 'ui-badge-neutral'
  }
})
</script>

<template>
  <span
    class="inline-flex items-center px-2 py-0.5 text-[11px] font-medium uppercase tracking-wide"
    :class="toneClass"
  >
    {{ label }}
  </span>
</template>
