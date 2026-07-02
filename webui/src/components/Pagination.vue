<script setup lang="ts">
import { computed } from 'vue'
import ChevronLeft from 'lucide-vue-next/dist/esm/icons/chevron-left.js'
import ChevronRight from 'lucide-vue-next/dist/esm/icons/chevron-right.js'

const props = withDefaults(
  defineProps<{
    total: number
    limit: number
    offset: number
    disabled?: boolean
    maxVisible?: number
  }>(),
  {
    disabled: false,
    maxVisible: 7,
  },
)

const emit = defineEmits<{
  'update:offset': [offset: number]
}>()

const currentPage = computed(() => Math.floor(props.offset / props.limit) + 1)

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.limit)))

const hasPrevious = computed(() => props.offset > 0)

const hasNext = computed(() => props.offset + props.limit < props.total)

type PageItem = number | 'ellipsis'

const pageItems = computed((): PageItem[] => {
  const total = totalPages.value
  const current = currentPage.value
  const max = props.maxVisible

  if (total <= max) {
    return Array.from({ length: total }, (_, index) => index + 1)
  }

  const items: PageItem[] = [1]
  const innerSlots = max - 2
  const half = Math.floor(innerSlots / 2)

  let start = Math.max(2, current - half)
  let end = Math.min(total - 1, start + innerSlots - 1)

  if (end - start + 1 < innerSlots) {
    start = Math.max(2, end - innerSlots + 1)
  }

  if (start > 2) {
    items.push('ellipsis')
  }

  for (let page = start; page <= end; page += 1) {
    items.push(page)
  }

  if (end < total - 1) {
    items.push('ellipsis')
  }

  items.push(total)
  return items
})

function goToPage(page: number): void {
  if (props.disabled || page < 1 || page > totalPages.value || page === currentPage.value) {
    return
  }
  emit('update:offset', (page - 1) * props.limit)
}

function goPrevious(): void {
  if (!hasPrevious.value) {
    return
  }
  goToPage(currentPage.value - 1)
}

function goNext(): void {
  if (!hasNext.value) {
    return
  }
  goToPage(currentPage.value + 1)
}
</script>

<template>
  <nav
    class="flex flex-wrap items-center justify-center gap-1"
    aria-label="Pagination"
  >
    <button
      type="button"
      class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 inline-flex h-8 w-8 items-center justify-center p-0"
      :disabled="disabled || !hasPrevious"
      aria-label="Previous page"
      @click="goPrevious"
    >
      <ChevronLeft class="h-4 w-4" />
    </button>

    <template v-for="(item, index) in pageItems" :key="`${item}-${index}`">
      <span
        v-if="item === 'ellipsis'"
        class="inline-flex h-8 min-w-8 items-center justify-center px-1 text-xs text-muted-foreground"
       
      >
        …
      </span>
      <button
        v-else
        type="button"
        class="inline-flex h-8 min-w-8 items-center justify-center px-2 text-xs"
        :class="
          item === currentPage
            ? 'rounded-md border border-input bg-background px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground border-primary bg-primary/15 text-foreground font-medium'
            : 'inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50'
        "
        :aria-current="item === currentPage ? 'page' : undefined"
        :disabled="disabled"
        @click="goToPage(item)"
      >
        {{ item }}
      </button>
    </template>

    <button
      type="button"
      class="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium text-foreground shadow-none transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50 inline-flex h-8 w-8 items-center justify-center p-0"
      :disabled="disabled || !hasNext"
      aria-label="Next page"
      @click="goNext"
    >
      <ChevronRight class="h-4 w-4" />
    </button>
  </nav>
</template>
