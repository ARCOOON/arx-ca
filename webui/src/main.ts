import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useThemeStore } from './stores/theme'
import './assets/index.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)

// Apply the persisted theme before mount to avoid a flash of the wrong palette.
useThemeStore(pinia).init()

app.use(router)
app.mount('#app')
