<script setup lang="ts">
import type { SavedList } from '@reviews/sdk'
import { HeartIcon, ListIcon } from 'lucide-vue-next'
import { ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useLoad } from '@/composables/useLoad'
import { listsApi } from '@/lib/api'
import { formatCount } from '@/lib/format'
import { listFailureText } from '@/lib/lists'

// own is true on the page of the signed in user. It shows the create form.
const props = defineProps<{ reviewerId: string; own: boolean }>()

const lists = useLoad<SavedList[]>(async () => (await listsApi.listReviewerLists({ reviewerId: props.reviewerId })).items)
watch(() => props.reviewerId, () => void lists.load(), { immediate: true })

const name = ref('')
const creating = ref(false)
const error = ref('')

async function create() {
  if (creating.value || name.value.trim() === '') {
    return
  }
  creating.value = true
  error.value = ''
  try {
    const list = await listsApi.createList({ savedListCreate: { name: name.value.trim() } })
    const [favorites, ...others] = lists.data.value ?? []
    lists.data.value = favorites ? [favorites, list, ...others] : [list]
    name.value = ''
  } catch (e) {
    error.value = await listFailureText(e, 'The list was not created. Try again in a moment.')
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <div v-if="lists.loading.value" aria-busy="true" aria-label="Loading lists" class="flex gap-2">
      <Skeleton class="h-9 w-32 rounded-full" />
      <Skeleton class="h-9 w-32 rounded-full" />
    </div>
    <p v-else-if="lists.failed.value" class="text-destructive text-sm" role="alert">
      The lists did not load.
      <button type="button" class="text-foreground font-medium underline underline-offset-4" @click="lists.load">Try again</button>
    </p>
    <ul v-else class="flex flex-wrap gap-2">
      <li v-for="list in lists.data.value" :key="list.id">
        <RouterLink
          :to="`/lists/${list.id}`"
          class="hover:border-ring inline-flex max-w-full items-center gap-2 rounded-full border px-4 py-1.5 text-sm font-medium"
        >
          <component :is="list.kind === 'favorites' ? HeartIcon : ListIcon" class="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
          <span class="min-w-0 break-words">{{ list.name }}</span>
          <span class="text-muted-foreground font-normal">{{ formatCount(list.itemCount, 'restaurant') }}</span>
        </RouterLink>
      </li>
    </ul>
    <form v-if="own" class="flex max-w-md flex-wrap gap-2" @submit.prevent="create">
      <label for="new-list-name" class="sr-only">Name of a new list</label>
      <Input id="new-list-name" v-model="name" class="min-w-40 flex-1" placeholder="New list" maxlength="255" autocomplete="off" />
      <Button type="submit" variant="secondary" :disabled="creating">Create list</Button>
      <p v-if="error" class="text-destructive w-full text-sm" role="alert">{{ error }}</p>
    </form>
  </div>
</template>
