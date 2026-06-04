<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ShieldCheck from 'lucide-vue-next/dist/esm/icons/shield-check.js'
import axios from 'axios'
import { login } from '../api/auth'
import { useAuthStore } from '../store/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const errorMessage = ref('')
const isSubmitting = ref(false)

async function handleSubmit(): Promise<void> {
  errorMessage.value = ''
  isSubmitting.value = true

  try {
    const session = await login({
      email: email.value.trim(),
      password: password.value,
    })

    authStore.setSession(session.token, session.roles ?? [])

    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.push(redirect)
  } catch (error) {
    if (axios.isAxiosError(error)) {
      const apiError = error.response?.data as { error?: string } | undefined
      if (error.response?.status === 401) {
        errorMessage.value = apiError?.error ?? 'Invalid email or password'
      } else {
        errorMessage.value = apiError?.error ?? 'Unable to sign in. Please try again.'
      }
    } else if (error instanceof Error) {
      errorMessage.value = error.message
    } else {
      errorMessage.value = 'Unable to sign in. Please try again.'
    }
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div class="ui-shell flex min-h-screen items-center justify-center px-4 py-12">
    <div class="w-full max-w-md">
      <div class="mb-8 text-center">
        <div class="ui-brand-icon mx-auto mb-4 flex h-14 w-14 items-center justify-center bg-[var(--bg-elevated)]">
          <ShieldCheck class="h-7 w-7" style="color: var(--accent-color)" aria-hidden="true" />
        </div>
        <h1 class="text-2xl font-semibold tracking-tight ui-text-primary">Arx Certificate Authority</h1>
        <p class="mt-2 text-sm ui-text-muted">Sign in with your administrator credentials</p>
      </div>

      <form class="ui-elevated ui-dialog p-6" @submit.prevent="handleSubmit">
        <div v-if="errorMessage" class="mb-4 ui-alert-error text-sm" role="alert">
          {{ errorMessage }}
        </div>

        <div class="space-y-4">
          <div>
            <label for="email" class="mb-1.5 block text-sm font-medium ui-text-secondary">Email</label>
            <input
              id="email"
              v-model="email"
              type="email"
              name="email"
              autocomplete="username"
              required
              class="ui-input py-2.5 text-sm"
              placeholder="admin@example.com"
            />
          </div>

          <div>
            <label for="password" class="mb-1.5 block text-sm font-medium ui-text-secondary">Password</label>
            <input
              id="password"
              v-model="password"
              type="password"
              name="password"
              autocomplete="current-password"
              required
              class="ui-input py-2.5 text-sm"
              placeholder="Enter your password"
            />
          </div>
        </div>

        <button
          type="submit"
          :disabled="isSubmitting"
          class="ui-btn-primary mt-6 w-full py-2.5"
          style="background-color: var(--accent-color); color: var(--text-inverse); border-color: var(--accent-muted)"
        >
          {{ isSubmitting ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>
    </div>
  </div>
</template>
