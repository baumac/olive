<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import lockupDark from '@/assets/logos/sponga47-reviews-lockup-dark.svg'
import lockupLight from '@/assets/logos/sponga47-reviews-lockup-light.svg'
import markDark from '@/assets/logos/sponga47-reviews-mark-dark.svg'
import markLight from '@/assets/logos/sponga47-reviews-mark-light.svg'
import { UserMenu } from '@/components/ui/user-menu'
import { signInPath, signOut } from '@/composables/useAuth'
import { currentUser } from '@/lib/session'

const route = useRoute()
const router = useRouter()

// A sign in from these pages does not come back to them.
const authPaths = ['/sign-in', '/sign-up', '/confirm-email', '/forgot-password', '/reset-password']
const onAuthPage = computed(() => authPaths.includes(route.path))
const signIn = computed(() => {
  if (route.path === '/sign-in' || route.path === '/sign-up') {
    return route.fullPath
  }
  if (onAuthPage.value) {
    return '/sign-in'
  }
  return router.resolve(signInPath(route.fullPath)).fullPath
})

// A section link is current on each page of its section. See decision 00005.
const sections = [
  { path: '/restaurants', label: 'Restaurants' },
  { path: '/recipes', label: 'Recipes' },
]
function current(path: string): boolean {
  return route.path === path || route.path.startsWith(`${path}/`)
}
</script>

<template>
  <a
    href="#main"
    class="bg-background text-foreground sr-only z-50 rounded-md px-3 py-2 focus:not-sr-only focus:fixed focus:top-3 focus:left-3"
  >
    Skip to content
  </a>
  <header class="bg-background sticky top-0 z-40 border-b">
    <!-- Below sm, the gaps are small. The row then fits on a 320px screen. -->
    <div class="mx-auto flex h-16 max-w-6xl items-center gap-2 px-4 sm:gap-4 sm:px-6">
      <RouterLink to="/" class="shrink-0 rounded-md">
        <span class="sr-only">Sponga47 Reviews, home</span>
        <img :src="lockupLight" alt="" class="hidden h-9 md:block dark:md:hidden" />
        <img :src="lockupDark" alt="" class="hidden h-9 dark:md:block" />
        <img :src="markLight" alt="" class="h-9 md:hidden dark:hidden" />
        <img :src="markDark" alt="" class="hidden h-9 dark:block dark:md:hidden" />
      </RouterLink>
      <nav aria-label="Sections" class="flex min-w-0 flex-1 items-center gap-1">
        <RouterLink
          v-for="s in sections"
          :key="s.path"
          :to="s.path"
          :aria-current="current(s.path) ? 'page' : undefined"
          :class="[
            'rounded-md px-2 py-2 text-sm font-medium whitespace-nowrap sm:px-3',
            current(s.path) ? 'bg-secondary text-secondary-foreground' : 'text-muted-foreground hover:text-foreground',
          ]"
        >
          {{ s.label }}
        </RouterLink>
      </nav>
      <UserMenu
        :user="currentUser"
        :sign-in-path="signIn"
        my-reviews-path="/my-reviews"
        :my-lists-path="currentUser ? `/reviewers/${currentUser.id}` : undefined"
        account-path="/account"
        :sign-in-current="route.path === '/sign-in'"
        @sign-out="signOut(router)"
      />
    </div>
  </header>
</template>
