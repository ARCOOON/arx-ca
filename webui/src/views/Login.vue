<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ShieldCheck from 'lucide-vue-next/dist/esm/icons/shield-check.js'
import axios from 'axios'
import { login } from '../api/auth'
import { useAuthStore } from '../store/auth'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert } from '@/components/ui/alert'

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
  <div class="flex min-h-screen items-center justify-center bg-background px-4 py-12">
    <div class="w-full max-w-md">
      <div class="mb-8 text-center">
        <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-md border border-border bg-muted">
          <ShieldCheck class="h-7 w-7 text-primary" aria-hidden="true" />
        </div>
        <h1 class="font-heading text-2xl font-semibold tracking-tight text-foreground">Arx Certificate Authority</h1>
        <p class="mt-2 text-sm text-muted-foreground">Sign in with your administrator credentials</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Sign in</CardTitle>
          <CardDescription>Enter your administrator email and password.</CardDescription>
        </CardHeader>
        <CardContent>
          <form class="space-y-4" @submit.prevent="handleSubmit">
            <Alert v-if="errorMessage" variant="destructive">
              {{ errorMessage }}
            </Alert>

            <div class="space-y-2">
              <Label for="email">Email</Label>
              <Input
                id="email"
                v-model="email"
                type="email"
                name="email"
                autocomplete="username"
                required
                placeholder="admin@example.com"
              />
            </div>

            <div class="space-y-2">
              <Label for="password">Password</Label>
              <Input
                id="password"
                v-model="password"
                type="password"
                name="password"
                autocomplete="current-password"
                required
                placeholder="Enter your password"
              />
            </div>

            <Button type="submit" class="w-full" :disabled="isSubmitting">
              {{ isSubmitting ? 'Signing in…' : 'Sign in' }}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
