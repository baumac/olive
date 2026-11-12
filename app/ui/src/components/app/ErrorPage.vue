<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { signInPath } from '@/composables/useAuth'
import { usePageTitle } from '@/composables/usePageTitle'

const props = defineProps<{
  // status is null for a network error.
  status: number | null
}>()

defineEmits<{ retry: [] }>()

const route = useRoute()
const router = useRouter()

const text = computed(() => {
  if (props.status === 404) {
    return {
      title: 'We cannot find this page',
      description: 'The page may have been deleted, or the link has a typo.',
    }
  }
  if (props.status === 403) {
    return { title: 'You cannot open this page', description: 'Sign in with the account that created this record.' }
  }
  return { title: 'Something went wrong', description: 'This is a problem on our side. Try again in a moment.' }
})

usePageTitle(() => text.value.title)
</script>

<template>
  <main id="main" class="mx-auto flex max-w-2xl flex-col items-start gap-4 px-4 py-12 sm:px-6">
    <p v-if="status" class="text-primary text-sm font-semibold tabular-nums">{{ status }}</p>
    <h1 class="text-4xl font-extrabold tracking-tight">{{ text.title }}</h1>
    <p class="text-muted-foreground text-xl">{{ text.description }}</p>
    <div class="flex flex-wrap gap-2 pt-2">
      <template v-if="status === 404">
        <Button as-child><RouterLink to="/restaurants">Go to restaurants</RouterLink></Button>
        <Button variant="outline" @click="router.back()">Go back</Button>
      </template>
      <template v-else-if="status === 403">
        <Button as-child><RouterLink :to="signInPath(route.fullPath)">Sign in</RouterLink></Button>
      </template>
      <template v-else>
        <Button @click="$emit('retry')">Try again</Button>
        <Button variant="outline" as-child><RouterLink to="/restaurants">Go to restaurants</RouterLink></Button>
      </template>
    </div>
  </main>
</template>
