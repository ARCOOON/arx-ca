<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/utils/cn'

type Variant = 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive' | 'link'
type Size = 'sm' | 'md' | 'lg' | 'icon'

const props = withDefaults(
  defineProps<{
    variant?: Variant
    size?: Size
    disabled?: boolean
    class?: string
  }>(),
  { variant: 'default', size: 'md', disabled: false },
)

const base =
  'inline-flex items-center justify-center gap-2 rounded-md font-medium text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-50 cursor-pointer select-none'

const variants: Record<Variant, string> = {
  default:
    'bg-primary text-primary-foreground hover:bg-primary/90 active:bg-primary/80',
  secondary:
    'bg-surface-raised text-foreground border border-border hover:bg-accent active:bg-muted',
  outline:
    'border border-border bg-transparent text-foreground hover:bg-accent active:bg-muted',
  ghost:
    'bg-transparent text-foreground hover:bg-accent active:bg-muted',
  destructive:
    'bg-destructive text-destructive-foreground hover:bg-destructive/90 active:bg-destructive/80',
  link: 'text-primary underline-offset-4 hover:underline bg-transparent',
}

const sizes: Record<Size, string> = {
  sm: 'h-8 px-3 text-xs rounded-md',
  md: 'h-9 px-4',
  lg: 'h-10 px-6 text-base',
  icon: 'h-9 w-9 p-0',
}

const classes = computed(() =>
  cn(base, variants[props.variant], sizes[props.size], props.class),
)
</script>

<template>
  <button :class="classes" :disabled="disabled" v-bind="$attrs">
    <slot />
  </button>
</template>
