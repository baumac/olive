import { createApp } from 'vue'
import './assets/main.css'
import App from './App.vue'
import { signInPath, startSessionRefresh } from './composables/useAuth'
import { handleSessionEnded } from './lib/api'
import { clearAllDrafts } from './lib/drafts'
import { watchOtherTabs } from './lib/session'
import { router } from './router'

handleSessionEnded(() => {
  const route = router.currentRoute.value
  if (route.meta.requiresAuth) {
    void router.push(signInPath(route.fullPath, 'session-ended'))
  }
})

watchOtherTabs((before, after) => {
  if (before && after) {
    // The pages have data of the user before. A reload gets all of it again.
    window.location.reload()
    return
  }
  if (!after) {
    clearAllDrafts()
  }
  // force runs the route guards again for the same path. They move the
  // tab away from a page that the new session cannot open.
  const { path, query, hash } = router.currentRoute.value
  void router.replace({ path, query, hash, force: true })
})

startSessionRefresh()
createApp(App).use(router).mount('#app')
