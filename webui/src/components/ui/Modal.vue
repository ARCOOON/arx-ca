<script setup lang="ts">
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'

defineProps<{
  open: boolean
  title: string
  wide?: boolean
}>()

const emit = defineEmits<{
  close: []
}>()
</script>

<template>
  <Dialog :open="open" @update:open="(value) => !value && emit('close')">
    <DialogContent
      :class="cn('flex max-h-[90vh] flex-col gap-0 p-0', wide ? 'sm:max-w-2xl' : 'sm:max-w-lg')"
    >
      <DialogHeader class="border-b border-border px-4 py-3">
        <DialogTitle class="text-sm font-semibold">{{ title }}</DialogTitle>
      </DialogHeader>

      <div class="min-h-0 flex-1 overflow-y-auto px-4 py-4">
        <slot />
      </div>

      <DialogFooter v-if="$slots.footer" class="border-t border-border px-4 py-3 sm:justify-end">
        <slot name="footer" />
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
