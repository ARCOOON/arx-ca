<script setup lang="ts">
import { computed, ref } from 'vue'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    modelValue: string[]
    id?: string
    placeholder?: string
    disabled?: boolean
  }>(),
  {
    id: undefined,
    placeholder: 'Type a DNS name or IP and press Enter',
    disabled: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const draft = ref('')

const tags = computed({
  get: () => props.modelValue,
  set: (value: string[]) => emit('update:modelValue', value),
})

function normalizeToken(raw: string): string {
  return raw.trim().replace(/,+$/, '').trim()
}

function commitDraft(): void {
  const value = normalizeToken(draft.value)
  draft.value = ''
  if (!value) {
    return
  }
  if (tags.value.includes(value)) {
    return
  }
  tags.value = [...tags.value, value]
}

function removeTag(index: number): void {
  tags.value = tags.value.filter((_, i) => i !== index)
}

function onInputKeydown(event: KeyboardEvent): void {
  if (event.key === 'Enter' || event.key === ' ' || event.key === ',') {
    event.preventDefault()
    commitDraft()
    return
  }
  if (event.key === 'Backspace' && draft.value === '' && tags.value.length > 0) {
    tags.value = tags.value.slice(0, -1)
  }
}

function onPaste(event: ClipboardEvent): void {
  const text = event.clipboardData?.getData('text') ?? ''
  if (!text.includes(',') && !text.includes('\n')) {
    return
  }
  event.preventDefault()
  const parts = text.split(/[\n,]+/)
  const next = [...tags.value]
  for (const part of parts) {
    const value = normalizeToken(part)
    if (value && !next.includes(value)) {
      next.push(value)
    }
  }
  tags.value = next
}
</script>

<template>
  <div
    :class="cn(
      'mt-1.5 flex min-h-10 flex-wrap items-center gap-1.5 rounded-md border border-input bg-background px-2 py-1.5',
      disabled && 'opacity-60',
    )"
  >
    <Badge
      v-for="(tag, index) in tags"
      :key="`${tag}-${index}`"
      variant="secondary"
      class="gap-1 font-mono text-[11px]"
    >
      {{ tag }}
      <button
        type="button"
        class="leading-none text-muted-foreground hover:text-foreground"
        :disabled="disabled"
        :aria-label="`Remove ${tag}`"
        @click="removeTag(index)"
      >
        ×
      </button>
    </Badge>
    <input
      :id="id"
      v-model="draft"
      type="text"
      class="min-w-32 flex-1 border-0 bg-transparent p-0 text-xs outline-none"
      :placeholder="tags.length === 0 ? placeholder : ''"
      :disabled="disabled"
      spellcheck="false"
      autocomplete="off"
      @keydown="onInputKeydown"
      @blur="commitDraft"
      @paste="onPaste"
    />
  </div>
</template>
