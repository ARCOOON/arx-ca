<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { LogOut, UserRound } from 'lucide-vue-next'
import { useAuthStore } from '../../store/auth'

const emit = defineEmits<{
  logout: []
}>()

const route = useRoute()
const authStore = useAuthStore()

const pageTitle = computed(() => {
  const title = route.meta.title
  return typeof title === 'string' ? title : 'Console'
})

const pageSubtitle = computed(() => {
  const subtitle = route.meta.subtitle
  return typeof subtitle === 'string' ? subtitle : ''
})

const roleLabel = computed(() => {
  if (authStore.roles.length > 0) {
    return authStore.roles.join(', ')
  }
  return 'Administrator'
})
</script>

<template>
  <header class="flex items-center justify-between border-b border-zinc-800 bg-zinc-900/40 px-5 py-3">
    <div class="min-w-0">
      <h1 class="truncate text-base font-semibold text-zinc-50">{{ pageTitle }}</h1>
      <p v-if="pageSubtitle" class="truncate text-xs text-zinc-500">{{ pageSubtitle }}</p>
    </div>

    <div class="flex items-center gap-3">
      <div class="hidden items-center gap-2 border border-zinc-800 bg-zinc-950 px-3 py-1.5 sm:flex">
        <UserRound class="h-3.5 w-3.5 text-zinc-500" aria-hidden="true" />
        <div class="text-right">
          <p class="text-[10px] uppercase tracking-wide text-zinc-500">Signed in</p>
          <p class="max-w-[12rem] truncate text-xs text-zinc-200">{{ roleLabel }}</p>
        </div>
      </div>

      <button
        type="button"
        class="inline-flex items-center gap-1.5 border border-zinc-700 bg-zinc-950 px-2.5 py-1.5 text-xs text-zinc-300 transition hover:border-zinc-600 hover:text-zinc-100"
        @click="emit('logout')"
      >
        <LogOut class="h-3.5 w-3.5" aria-hidden="true" />
        Logout
      </button>
    </div>
  </header>
</template>
