<script setup lang="ts">
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'

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
  <div class="flex items-center justify-between gap-4 rounded-md border border-input bg-muted/30 px-4 py-3">
    <Label class="text-sm text-foreground/80">{{ label }}</Label>
    <Switch
      :checked="enabled"
      :disabled="readonly"
      :aria-label="`${label}: ${enabled ? 'enabled' : 'disabled'}`"
      @update:checked="!readonly && emit('toggle')"
    />
  </div>
</template>
