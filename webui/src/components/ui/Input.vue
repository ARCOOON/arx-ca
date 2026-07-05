<script setup lang="ts">
import { computed, useAttrs } from 'vue'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    modelValue?: string | number
    type?: string
    disabled?: boolean
    id?: string
    placeholder?: string
    autocomplete?: string
    maxlength?: number
    spellcheck?: boolean | 'true' | 'false'
  }>(),
  {
    type: 'text',
    disabled: false,
    spellcheck: undefined,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const attrs = useAttrs()

const classes = computed(() =>
  cn(
    'flex h-9 w-full rounded-[var(--radius-control)] border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm text-[var(--color-text)] outline-none transition-colors placeholder:text-[var(--color-text-muted)] focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-focus-ring)] disabled:cursor-not-allowed disabled:opacity-50',
    attrs.class as string | undefined,
  ),
)

function onInput(event: Event): void {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <input
    :id="id"
    :type="type"
    :value="modelValue"
    :disabled="disabled"
    :placeholder="placeholder"
    :autocomplete="autocomplete"
    :maxlength="maxlength"
    :spellcheck="spellcheck"
    :class="classes"
    @input="onInput"
  />
</template>
