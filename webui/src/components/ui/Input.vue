<script setup lang="ts">
import { cn } from '@/utils/cn'

const props = withDefaults(
  defineProps<{
    modelValue?: string | number
    placeholder?: string
    type?: string
    disabled?: boolean
    class?: string
  }>(),
  { type: 'text', disabled: false },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <input
    :value="modelValue"
    :type="type"
    :placeholder="placeholder"
    :disabled="disabled"
    :class="
      cn(
        'flex h-9 w-full rounded-md border border-input bg-surface px-3 py-2 text-sm',
        'text-foreground placeholder:text-foreground-subtle',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1',
        'disabled:cursor-not-allowed disabled:opacity-50',
        'transition-colors',
        props.class,
      )
    "
    v-bind="$attrs"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  />
</template>
