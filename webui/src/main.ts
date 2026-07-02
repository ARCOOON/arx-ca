import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import 'vue-sonner/style.css'
import './style.css'
import { initTheme } from './composables/useTheme'

initTheme()

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.mount('#app')
