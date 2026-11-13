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

const onAuthPage = computed(() => route.path === '/sign-in' || route.path === '/sign-up')
const signIn = computed(() => {
  if (onAuthPage.value) {
    return route.fullPath
  }
  return router.resolve(signInPath(route.fullPath)).fullPath
})
</script>

<template>
  <a
    href="#main"
    class="bg-background text-foreground sr-only z-50 rounded-md px-3 py-2 focus:not-sr-only focus:fixed focus:top-3 focus:left-3"
  >
    Skip to content
  </a>
  <header class="bg-background sticky top-0 z-40 border-b">
    <div class="mx-auto flex h-16 max-w-6xl items-center gap-4 px-4 sm:px-6">
      <RouterLink to="/" class="shrink-0 rounded-md">
        <span class="sr-only">Sponga47 Reviews, home</span>
        <img :src="lockupLight" alt="" class="hidden h-9 md:block dark:md:hidden" />
        <img :src="lockupDark" alt="" class="hidden h-9 dark:md:block" />
        <img :src="markLight" alt="" class="h-9 md:hidden dark:hidden" />
        <img :src="markDark" alt="" class="hidden h-9 dark:block dark:md:hidden" />
      </RouterLink>
      <div class="flex-1" />
      <UserMenu :user="currentUser" :sign-in-path="signIn" my-reviews-path="/my-reviews" :sign-in-current="route.path === '/sign-in'" @sign-out="signOut(router)" />
    </div>
  </header>
</template>
