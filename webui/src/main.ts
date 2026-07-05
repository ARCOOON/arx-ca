import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { Toaster } from 'vue-sonner'
import App from './App.vue'
import router from './router'
import { applyTheme } from './composables/useTheme'
import './style.css'

const storedTheme = localStorage.getItem('arx_theme_preference')
const legacyTheme = localStorage.getItem('arx_theme')
const pref =
  storedTheme === 'light' || storedTheme === 'dark'
    ? storedTheme
    : legacyTheme === 'light' || legacyTheme === 'dark'
      ? legacyTheme
      : window.matchMedia('(prefers-color-scheme: dark)').matches
        ? 'dark'
        : 'light'
applyTheme(pref)

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.component('Toaster', Toaster)
app.mount('#app')
