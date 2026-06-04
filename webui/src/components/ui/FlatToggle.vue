<script setup lang="ts">
defineProps<{
  enabled: boolean
  label: string
  readonly?: boolean
}>()

const emit = defineEmits<{
  toggle: []
}>()
</script>

<template>
  <div class="ui-inset flex items-center justify-between gap-4 px-4 py-3">
    <span class="text-sm ui-text-secondary">{{ label }}</span>
    <button
      type="button"
      class="ui-theme-toggle"
      :data-active="enabled"
      :disabled="readonly"
      :aria-pressed="enabled"
      :aria-label="`${label}: ${enabled ? 'enabled' : 'disabled'}`"
      :class="readonly ? 'cursor-default' : 'cursor-pointer'"
      @click="!readonly && emit('toggle')"
    >
      <span class="ui-theme-toggle-thumb" />
    </button>
  </div>
</template>
