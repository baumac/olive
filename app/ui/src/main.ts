import { createApp } from 'vue'
import './assets/main.css'
import App from './App.vue'
import { signInPath, startSessionRefresh } from './composables/useAuth'
import { handleSessionEnded } from './lib/api'
import { router } from './router'

handleSessionEnded(() => {
  const route = router.currentRoute.value
  if (route.meta.requiresAuth) {
    void router.push(signInPath(route.fullPath, 'session-ended'))
  }
})

startSessionRefresh()
createApp(App).use(router).mount('#app')
