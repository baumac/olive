import { createApp } from 'vue'
import './assets/main.css'
import App from './App.vue'
import { signInPath, startSessionCheck } from './composables/useAuth'
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
  const { path, query, hash, meta } = router.currentRoute.value
  if (after ? meta.guestOnly : meta.requiresAuth) {
    // force runs the route guards again for the same path. They move the
    // tab away from a page that the new session cannot open.
    void router.replace({ path, query, hash, force: true })
  }
})

startSessionCheck()
createApp(App).use(router).mount('#app')
