<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/utils/cn'

type Variant = 'default' | 'secondary' | 'success' | 'destructive' | 'warning' | 'info' | 'outline'

const props = withDefaults(
  defineProps<{
    variant?: Variant
    class?: string
  }>(),
  { variant: 'default' },
)

const variants: Record<Variant, string> = {
  default:   'bg-primary/10 text-primary border-primary/20',
  secondary: 'bg-muted text-muted-foreground border-border',
  success:   'bg-success/10 text-success border-success/20',
  destructive: 'bg-destructive/10 text-destructive border-destructive/20',
  warning:   'bg-warning/10 text-warning-foreground border-warning/20',
  info:      'bg-info/10 text-info border-info/20',
  outline:   'bg-transparent text-foreground border-border',
}

const classes = computed(() =>
  cn(
    'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs font-medium leading-none',
    variants[props.variant],
    props.class,
  ),
)
</script>

<template>
  <span :class="classes"><slot /></span>
</template>
