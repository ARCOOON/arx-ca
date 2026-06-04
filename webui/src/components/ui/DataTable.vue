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
  <div class="overflow-x-auto border border-zinc-800">
    <table class="min-w-full text-left text-xs">
      <thead class="border-b border-zinc-800 bg-zinc-900 text-[10px] uppercase tracking-wider text-zinc-500">
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
      <tbody class="divide-y divide-zinc-800/80 bg-zinc-950">
        <tr v-if="loading">
          <td :colspan="columns.length" class="px-3 py-6 text-sm text-zinc-500">Loading…</td>
        </tr>
        <tr v-else-if="rows.length === 0">
          <td :colspan="columns.length" class="px-3 py-6 text-sm text-zinc-500">
            {{ emptyMessage ?? 'No records found.' }}
          </td>
        </tr>
        <tr
          v-for="row in rows"
          v-else
          :key="rowKey(row)"
          class="hover:bg-zinc-900/60"
        >
          <td
            v-for="column in columns"
            :key="column.key"
            class="px-3 py-2 text-zinc-300"
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
