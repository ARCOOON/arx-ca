<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import Button from '@/components/ui/Button.vue'
import Spinner from '@/components/ui/Spinner.vue'
import { fetchAuditLogs } from '@/api/audit'
import type { AuditLogEntry } from '@/types/api'
import { formatDate } from '@/utils/format'
import { extractErrorMessage } from '@/utils/errors'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const loading = ref(true)
const logs = ref<AuditLogEntry[]>([])
const total = ref(0)
const page = ref(0)
const pageSize = 50
const search = ref('')

async function load(): Promise<void> {
  loading.value = true
  try {
    const res = await fetchAuditLogs({ limit: pageSize, offset: page.value * pageSize })
    logs.value = res.logs
    total.value = res.total
  } catch (err) {
    toast.error(extractErrorMessage(err))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())

function statusVariant(code: number): 'success' | 'warning' | 'destructive' | 'secondary' {
  if (code >= 200 && code < 300) return 'success'
  if (code >= 400 && code < 500) return 'warning'
  if (code >= 500) return 'destructive'
  return 'secondary'
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center gap-2">
      <Input v-model="search" placeholder="Search actor, endpoint, action…" class="w-64" />
      <Button variant="ghost" size="sm" @click="load">Refresh</Button>
    </div>

    <Card class="overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-border bg-muted/50">
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide">Time</th>
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide">Actor</th>
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide">Action</th>
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide hidden md:table-cell">Endpoint</th>
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wide">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="5" class="text-center py-10"><Spinner class="mx-auto" /></td>
            </tr>
            <tr v-else-if="logs.length === 0">
              <td colspan="5" class="text-center py-10 text-sm text-foreground-muted">No audit log entries</td>
            </tr>
            <tr
              v-for="entry in logs.filter(e => !search || e.action.includes(search) || e.actor_id.includes(search) || e.endpoint.includes(search))"
              :key="entry.id"
              class="border-b border-border last:border-0 hover:bg-muted/30 transition-colors"
            >
              <td class="px-4 py-2.5 text-xs text-foreground-muted whitespace-nowrap">{{ formatDate(entry.timestamp) }}</td>
              <td class="px-4 py-2.5 text-xs text-foreground max-w-[120px] truncate">{{ entry.actor_id }}</td>
              <td class="px-4 py-2.5 text-xs text-foreground font-medium">{{ entry.action }}</td>
              <td class="px-4 py-2.5 font-mono text-xs text-foreground-muted hidden md:table-cell max-w-[200px] truncate">
                <span class="mr-1 text-primary font-semibold">{{ entry.http_method }}</span>{{ entry.endpoint }}
              </td>
              <td class="px-4 py-2.5">
                <Badge :variant="statusVariant(entry.status_code)">{{ entry.status_code }}</Badge>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="flex items-center justify-between px-4 py-2.5 border-t border-border bg-muted/30 text-xs text-foreground-muted">
        <span>{{ total }} total</span>
        <div class="flex gap-1">
          <Button variant="ghost" size="sm" :disabled="page === 0" @click="() => { page--; load() }">Previous</Button>
          <Button variant="ghost" size="sm" :disabled="(page + 1) * pageSize >= total" @click="() => { page++; load() }">Next</Button>
        </div>
      </div>
    </Card>
  </div>
</template>
