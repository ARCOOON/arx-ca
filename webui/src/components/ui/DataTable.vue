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
  <div class="overflow-x-auto border" style="border-color: var(--border-color)">
    <table class="min-w-full text-left text-xs">
      <thead class="ui-table-head">
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
      <tbody class="ui-table-body ui-table-divide">
        <tr v-if="loading">
          <td :colspan="columns.length" class="px-3 py-6 text-sm ui-text-muted">Loading…</td>
        </tr>
        <tr v-else-if="rows.length === 0">
          <td :colspan="columns.length" class="px-3 py-6 text-sm ui-text-muted">
            {{ emptyMessage ?? 'No records found.' }}
          </td>
        </tr>
        <tr v-for="row in rows" v-else :key="rowKey(row)" class="ui-table-row">
          <td
            v-for="column in columns"
            :key="column.key"
            class="px-3 py-2 ui-text-secondary"
            :class="column.cellClass"
          >
            <slot :name="`cell-${column.key}`" :row="row">
              {{ column.format ? column.format(row) : String(row[column.key] ?? '') }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
