<script setup lang="ts">
import { TokenInput } from '@/components/ui/token-input'

withDefaults(
  defineProps<{
    modelValue: string[]
    id?: string
    placeholder?: string
    disabled?: boolean
    suggestions?: string[]
  }>(),
  {
    id: undefined,
    placeholder: 'Type a DNS name or IP and press Enter',
    disabled: false,
    suggestions: () => [],
  },
)

defineEmits<{
  'update:modelValue': [value: string[]]
}>()
</script>

<template>
  <TokenInput
    :id="id"
    :model-value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    :suggestions="suggestions"
    @update:model-value="$emit('update:modelValue', $event)"
  />
</template>
