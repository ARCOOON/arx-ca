<script setup lang="ts">
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

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
  <div class="flex items-center justify-between gap-4 rounded-md border border-border bg-background px-4 py-3">
    <Label class="text-sm text-foreground/80">{{ label }}</Label>
    <Switch
      :checked="enabled"
      :disabled="readonly"
      :aria-label="`${label}: ${enabled ? 'enabled' : 'disabled'}`"
      @update:checked="!readonly && emit('toggle')"
    />
  </div>
</template>
