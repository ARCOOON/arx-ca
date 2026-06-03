<script setup lang="ts">
import { onMounted, ref } from 'vue'
import axios from 'axios'
import { listCertificates } from '../api/certificates'
import type { CertificateSummary } from '../types/api'
import DashboardLayout from '../layouts/DashboardLayout.vue'

const certificates = ref<CertificateSummary[]>([])
const total = ref(0)
const rawResponse = ref<string>('')
const isLoading = ref(true)
const errorMessage = ref('')

onMounted(async () => {
  isLoading.value = true
  errorMessage.value = ''

  try {
    const response = await listCertificates()
    certificates.value = response.certificates
    total.value = response.total
    rawResponse.value = JSON.stringify(response, null, 2)
  } catch (error) {
    if (axios.isAxiosError(error)) {
      const apiError = error.response?.data as { error?: string } | undefined
      errorMessage.value = apiError?.error ?? error.message
    } else if (error instanceof Error) {
      errorMessage.value = error.message
    } else {
      errorMessage.value = 'Failed to load certificates'
    }
  } finally {
    isLoading.value = false
  }
})

function formatDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}
</script>

<template>
  <DashboardLayout>
    <div class="space-y-6">
      <section class="rounded-xl border border-zinc-800 bg-zinc-900/40 p-5">
        <div class="flex items-start justify-between gap-4">
          <div>
            <h2 class="text-base font-semibold text-zinc-50">Managed Certificates</h2>
            <p class="mt-1 text-sm text-zinc-400">
              Authenticated request to <code class="rounded bg-zinc-800 px-1.5 py-0.5 text-xs text-emerald-300">GET /api/v1/certificates</code>
            </p>
          </div>
          <span
            v-if="!isLoading && !errorMessage"
            class="rounded-full border border-emerald-900/60 bg-emerald-950/40 px-3 py-1 text-xs font-medium text-emerald-300"
          >
            {{ total }} total
          </span>
        </div>

        <div v-if="isLoading" class="mt-6 text-sm text-zinc-400">Loading certificate inventory…</div>

        <div
          v-else-if="errorMessage"
          class="mt-6 rounded-lg border border-red-900/60 bg-red-950/40 px-4 py-3 text-sm text-red-300"
          role="alert"
        >
          {{ errorMessage }}
        </div>

        <div v-else-if="certificates.length === 0" class="mt-6 text-sm text-zinc-400">
          No certificates have been issued yet.
        </div>

        <div v-else class="mt-6 overflow-x-auto rounded-lg border border-zinc-800">
          <table class="min-w-full divide-y divide-zinc-800 text-left text-sm">
            <thead class="bg-zinc-900/80 text-xs uppercase tracking-wide text-zinc-500">
              <tr>
                <th class="px-4 py-3 font-medium">Serial</th>
                <th class="px-4 py-3 font-medium">Subject</th>
                <th class="px-4 py-3 font-medium">Valid Until</th>
                <th class="px-4 py-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-zinc-800 bg-zinc-950/40">
              <tr v-for="certificate in certificates" :key="certificate.serial" class="hover:bg-zinc-900/50">
                <td class="px-4 py-3 font-mono text-xs text-zinc-300">{{ certificate.serial }}</td>
                <td class="px-4 py-3 text-zinc-200">{{ certificate.subject }}</td>
                <td class="px-4 py-3 text-zinc-400">{{ formatDate(certificate.not_after) }}</td>
                <td class="px-4 py-3">
                  <span
                    class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium"
                    :class="
                      certificate.revoked
                        ? 'bg-red-950/50 text-red-300 ring-1 ring-red-900/60'
                        : 'bg-emerald-950/50 text-emerald-300 ring-1 ring-emerald-900/60'
                    "
                  >
                    {{ certificate.revoked ? 'Revoked' : 'Active' }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="rounded-xl border border-zinc-800 bg-zinc-900/40 p-5">
        <h2 class="text-base font-semibold text-zinc-50">Raw API Response</h2>
        <p class="mt-1 text-sm text-zinc-400">Unmodified JSON payload returned by the protected endpoint.</p>
        <pre
          class="mt-4 overflow-x-auto rounded-lg border border-zinc-800 bg-zinc-950 p-4 text-xs leading-relaxed text-zinc-300"
        >{{ rawResponse || (isLoading ? 'Waiting for response…' : 'No data') }}</pre>
      </section>
    </div>
  </DashboardLayout>
</template>
