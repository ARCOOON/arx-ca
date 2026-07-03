<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/store/auth'
import { login } from '@/api/auth'
import { useToast } from '@/composables/useToast'
import { initTheme } from '@/composables/useTheme'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import ToastHost from '@/components/ui/ToastHost.vue'
import { extractErrorMessage } from '@/utils/errors'

initTheme()

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const toast = useToast()

const email = ref('')
const password = ref('')
const loading = ref(false)

async function handleLogin(): Promise<void> {
  if (!email.value || !password.value) {
    toast.error('Email and password are required')
    return
  }

  loading.value = true
  try {
    const res = await login({ email: email.value, password: password.value })
    authStore.setSession(res.token, res.roles ?? [])
    const redirect = (route.query.redirect as string | undefined) ?? '/dashboard'
    await router.push(redirect)
  } catch (err) {
    toast.error(extractErrorMessage(err, 'Invalid credentials'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background p-4">
    <div class="w-full max-w-sm">
      <!-- Logo mark -->
      <div class="mb-8 flex flex-col items-center gap-3">
        <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary shadow-lg shadow-primary/20">
          <svg class="h-6 w-6 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
        <div class="text-center">
          <h1 class="text-xl font-bold text-foreground tracking-tight">ARX CA</h1>
          <p class="mt-1 text-sm text-foreground-muted">Certificate Authority Operator Console</p>
        </div>
      </div>

      <!-- Login card -->
      <div class="rounded-xl border border-border bg-card shadow-sm px-6 py-6">
        <form class="space-y-4" @submit.prevent="handleLogin">
          <div class="space-y-1.5">
            <Label for="email">Email</Label>
            <Input
              id="email"
              v-model="email"
              type="email"
              placeholder="operator@example.com"
              autocomplete="username"
              :disabled="loading"
              class="w-full"
            />
          </div>

          <div class="space-y-1.5">
            <Label for="password">Password</Label>
            <Input
              id="password"
              v-model="password"
              type="password"
              placeholder="••••••••"
              autocomplete="current-password"
              :disabled="loading"
              class="w-full"
            />
          </div>

          <Button type="submit" class="w-full" :disabled="loading">
            <span v-if="loading" class="flex items-center gap-2">
              <svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              Authenticating…
            </span>
            <span v-else>Sign in</span>
          </Button>
        </form>
      </div>

      <p class="mt-6 text-center text-xs text-foreground-subtle">
        ARX CA — Enterprise Certificate Authority
      </p>
    </div>

    <ToastHost />
  </div>
</template>
