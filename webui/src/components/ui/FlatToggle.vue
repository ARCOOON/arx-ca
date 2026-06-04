<script setup lang="ts">
defineProps<{
  enabled: boolean
  label: string
  readonly?: boolean
}>()
</script>

<template>
  <div class="flex items-center justify-between gap-4 border border-zinc-800 bg-zinc-950 px-4 py-3">
    <span class="text-sm text-zinc-300">{{ label }}</span>
    <button
      type="button"
      class="relative inline-flex h-5 w-9 shrink-0 border transition-colors"
      :class="[
        enabled ? 'border-emerald-700 bg-emerald-900/60' : 'border-zinc-700 bg-zinc-900',
        readonly ? 'cursor-default' : 'cursor-pointer',
      ]"
      :disabled="readonly"
      :aria-pressed="enabled"
      :aria-label="`${label}: ${enabled ? 'enabled' : 'disabled'}`"
    >
      <span
        class="absolute top-0.5 h-3.5 w-3.5 border transition-all"
        :class="[
          enabled
            ? 'left-[calc(100%-1rem)] border-emerald-500 bg-emerald-400'
            : 'left-0.5 border-zinc-600 bg-zinc-500',
        ]"
      />
    </button>
  </div>
</template>
