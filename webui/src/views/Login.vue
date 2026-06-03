<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ShieldCheck } from 'lucide-vue-next'
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
  <div class="flex min-h-screen items-center justify-center bg-zinc-950 px-4 py-12">
    <div class="w-full max-w-md">
      <div class="mb-8 text-center">
        <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-xl border border-zinc-800 bg-zinc-900">
          <ShieldCheck class="h-7 w-7 text-emerald-400" aria-hidden="true" />
        </div>
        <h1 class="text-2xl font-semibold tracking-tight text-zinc-50">Arx Certificate Authority</h1>
        <p class="mt-2 text-sm text-zinc-400">Sign in with your administrator credentials</p>
      </div>

      <form
        class="rounded-xl border border-zinc-800 bg-zinc-900/80 p-6"
        @submit.prevent="handleSubmit"
      >
        <div
          v-if="errorMessage"
          class="mb-4 rounded-lg border border-red-900/60 bg-red-950/50 px-3 py-2 text-sm text-red-300"
          role="alert"
        >
          {{ errorMessage }}
        </div>

        <div class="space-y-4">
          <div>
            <label for="email" class="mb-1.5 block text-sm font-medium text-zinc-300">Email</label>
            <input
              id="email"
              v-model="email"
              type="email"
              name="email"
              autocomplete="username"
              required
              class="w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 outline-none transition focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/20"
              placeholder="admin@example.com"
            />
          </div>

          <div>
            <label for="password" class="mb-1.5 block text-sm font-medium text-zinc-300">Password</label>
            <input
              id="password"
              v-model="password"
              type="password"
              name="password"
              autocomplete="current-password"
              required
              class="w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 outline-none transition focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/20"
              placeholder="Enter your password"
            />
          </div>
        </div>

        <button
          type="submit"
          :disabled="isSubmitting"
          class="mt-6 w-full rounded-lg bg-emerald-600 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {{ isSubmitting ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>
    </div>
  </div>
</template>
