<script setup lang="ts">
import type { Component } from 'vue'
import { Card, CardContent } from '@/components/ui/card'

defineProps<{
  label: string
  value: string | number
  icon?: Component
  hint?: string
  accent?: 'default' | 'warning' | 'destructive'
}>()
</script>

<template>
  <Card>
    <CardContent class="flex items-center gap-4 py-1">
      <div
        v-if="icon"
        class="flex size-10 shrink-0 items-center justify-center rounded-md"
        :class="{
          'bg-accent text-foreground': !accent || accent === 'default',
          'bg-warning/15 text-warning': accent === 'warning',
          'bg-destructive/10 text-destructive': accent === 'destructive',
        }"
      >
        <component :is="icon" class="size-5" />
      </div>
      <div class="min-w-0">
        <p class="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {{ label }}
        </p>
        <p class="mt-0.5 text-2xl font-semibold tabular-nums">{{ value }}</p>
        <p v-if="hint" class="text-xs text-muted-foreground">{{ hint }}</p>
      </div>
    </CardContent>
  </Card>
</template>
