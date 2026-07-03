<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ShieldCheck, Loader2 } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import ThemeSwitcher from '@/components/ThemeSwitcher.vue'
import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { extractApiError } from '@/lib/errors'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const isSubmitting = ref(false)
const errorMessage = ref('')

async function onSubmit(): Promise<void> {
  errorMessage.value = ''
  if (!email.value.trim() || !password.value) {
    errorMessage.value = 'Email and password are required.'
    return
  }

  isSubmitting.value = true
  try {
    const response = await login({ email: email.value.trim(), password: password.value })
    authStore.setSession(response.token, response.roles ?? [])
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : null
    await router.push(redirect ?? { name: 'dashboard' })
  } catch (error) {
    errorMessage.value = extractApiError(error, 'Invalid email or password')
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div class="relative flex min-h-screen items-center justify-center bg-background px-4">
    <div class="absolute right-4 top-4">
      <ThemeSwitcher />
    </div>

    <Card class="w-full max-w-sm">
      <CardHeader class="items-center text-center">
        <div
          class="mb-2 flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground"
        >
          <ShieldCheck class="size-6" />
        </div>
        <CardTitle class="text-xl">Arx CA Console</CardTitle>
        <CardDescription>Sign in to manage your certificate authority.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="onSubmit">
          <div class="space-y-2">
            <Label for="email">Email</Label>
            <Input
              id="email"
              v-model="email"
              type="email"
              autocomplete="username"
              placeholder="admin@example.com"
              :disabled="isSubmitting"
            />
          </div>
          <div class="space-y-2">
            <Label for="password">Password</Label>
            <Input
              id="password"
              v-model="password"
              type="password"
              autocomplete="current-password"
              placeholder="••••••••"
              :disabled="isSubmitting"
            />
          </div>

          <p
            v-if="errorMessage"
            class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
            role="alert"
          >
            {{ errorMessage }}
          </p>

          <Button type="submit" class="w-full" :disabled="isSubmitting">
            <Loader2 v-if="isSubmitting" class="size-4 animate-spin" />
            {{ isSubmitting ? 'Signing in…' : 'Sign in' }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
