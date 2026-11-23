<script setup lang="ts">
import { useRoute } from 'vue-router'
import { Button } from '@/components/ui/button'
import { signInPath } from '@/composables/useAuth'
import { currentUser } from '@/lib/session'

// VisitorNote tells a signed out visitor of a shared page what the app is.
defineProps<{ name: string }>()

// The visitor comes back to this page after the sign in or the sign up.
const route = useRoute()
</script>

<template>
  <aside v-if="!currentUser" class="bg-accent flex max-w-2xl flex-col gap-3 rounded-2xl p-5" aria-labelledby="visitor-note-heading">
    <h2 id="visitor-note-heading" class="text-accent-foreground font-serif text-[21px] leading-[26px] font-semibold">
      {{ name }} keeps food notes here
    </h2>
    <p class="text-[15px] leading-[23px]">
      Sponga47 Food is a food journal for a small group of friends and family. Each score goes from 1 to 10, and a
      higher score is always better.
    </p>
    <div class="flex flex-wrap gap-2.5">
      <Button as-child><RouterLink :to="{ path: '/sign-up', query: { redirect: route.fullPath } }">Create account</RouterLink></Button>
      <Button variant="outline" as-child><RouterLink :to="signInPath(route.fullPath)">Sign in</RouterLink></Button>
    </div>
  </aside>
</template>
