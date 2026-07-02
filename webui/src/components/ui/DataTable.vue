<script setup lang="ts" generic="T extends Record<string, unknown>">
export interface DataTableColumn<T> {
  key: string
  label: string
  headerClass?: string
  cellClass?: string
  format?: (row: T) => string
}

defineProps<{
  columns: DataTableColumn<T>[]
  rows: T[]
  rowKey: (row: T) => string
  loading?: boolean
  emptyMessage?: string
}>()
</script>

<template>
  <div class="overflow-x-auto rounded-lg border border-border">
    <table class="min-w-full text-left text-sm">
      <thead class="border-b border-border bg-muted/50 text-[10px] uppercase tracking-wide text-muted-foreground">
        <tr>
          <th
            v-for="column in columns"
            :key="column.key"
            class="px-3 py-2 font-medium"
            :class="column.headerClass"
          >
            {{ column.label }}
          </th>
        </tr>
      </thead>
      <tbody class="bg-card [&>tr+tr]:border-t [&>tr+tr]:border-border">
        <tr v-if="loading">
          <td :colspan="columns.length" class="px-3 py-6 text-sm text-muted-foreground">Loading…</td>
        </tr>
        <tr v-else-if="rows.length === 0">
          <td :colspan="columns.length" class="px-3 py-6 text-sm text-muted-foreground">
            {{ emptyMessage ?? 'No records found.' }}
          </td>
        </tr>
        <template v-for="row in rows" v-else :key="rowKey(row)">
          <tr class="hover:bg-muted/50">
            <td
              v-for="column in columns"
              :key="column.key"
              class="px-3 py-2 text-foreground/80"
              :class="column.cellClass"
            >
              <slot :name="`cell-${column.key}`" :row="row">
                {{ column.format ? column.format(row) : String(row[column.key] ?? '') }}
              </slot>
            </td>
          </tr>
          <slot name="row-expanded" :row="row" :columns="columns" />
        </template>
      </tbody>
    </table>
  </div>
</template>
