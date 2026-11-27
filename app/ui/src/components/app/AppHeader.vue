<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import lockupDark from '@/assets/logos/olive-lockup-dark.svg'
import lockupLight from '@/assets/logos/olive-lockup-light.svg'
import markDark from '@/assets/logos/olive-mark-dark.svg'
import markLight from '@/assets/logos/olive-mark-light.svg'
import { PlusIcon } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { UserMenu } from '@/components/ui/user-menu'
import { signInPath, signOut } from '@/composables/useAuth'
import { isCurrentSection, sections } from '@/lib/sections'
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

const showAddReview = computed(() => !!currentUser.value && !route.path.startsWith('/reviews/new'))
</script>

<template>
  <a
    href="#main"
    class="bg-background text-foreground sr-only z-50 rounded-md px-3 py-2 focus:not-sr-only focus:fixed focus:top-3 focus:left-3"
  >
    Skip to content
  </a>
  <header class="bg-background sticky top-0 z-40 border-b">
    <div class="mx-auto flex h-15 max-w-6xl items-center gap-2 pr-2 pl-4 sm:px-6 md:h-18 md:gap-6">
      <RouterLink to="/" class="shrink-0 rounded-md">
        <span class="sr-only">Olive, home</span>
        <img :src="lockupLight" alt="" class="hidden h-9 md:block dark:md:hidden">
        <img :src="lockupDark" alt="" class="hidden h-9 dark:md:block">
        <img :src="markLight" alt="" class="h-[30px] md:hidden dark:hidden">
        <img :src="markDark" alt="" class="hidden h-[30px] dark:block dark:md:hidden">
      </RouterLink>
      <!-- Below md, BottomNav has the sections. -->
      <nav aria-label="Sections" class="hidden items-center gap-1 md:flex">
        <RouterLink
          v-for="s in sections"
          :key="s.path"
          :to="s.path"
          :aria-current="isCurrentSection(s.path, route.path) ? 'page' : undefined"
          :class="[
            'inline-flex h-11 items-center rounded-full px-3.5 text-[15px] font-semibold whitespace-nowrap',
            isCurrentSection(s.path, route.path) ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:text-foreground',
          ]"
        >
          {{ s.label }}
        </RouterLink>
      </nav>
      <div class="ml-auto flex items-center gap-3">
        <Button v-if="showAddReview" as-child class="hidden md:inline-flex">
          <RouterLink to="/reviews/new"><PlusIcon aria-hidden="true" />Add review</RouterLink>
        </Button>
        <UserMenu
          :user="currentUser"
          :sign-in-path="signIn"
          :journal-path="currentUser ? `/reviewers/${currentUser.id}` : '/'"
          account-path="/account"
          :sign-in-current="route.path === '/sign-in'"
          @sign-out="signOut(router)"
        />
      </div>
    </div>
  </header>
</template>
