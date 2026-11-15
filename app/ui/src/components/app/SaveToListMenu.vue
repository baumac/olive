<script setup lang="ts">
import type { SavedList } from '@reviews/sdk'
import { ListPlusIcon } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { signInPath } from '@/composables/useAuth'
import { listsApi } from '@/lib/api'
import { tellPendingApproval } from '@/lib/approval'
import { listFailureText } from '@/lib/lists'
import { currentUser } from '@/lib/session'

// The favorites list is not in the menu. The heart of FavoriteButton
// changes it.
const props = defineProps<{ restaurantId: string }>()

const route = useRoute()
const router = useRouter()

const open = ref(false)
const lists = ref<SavedList[]>([])
const state = ref<'loading' | 'done' | 'failed'>('loading')
const error = ref('')
const name = ref('')
const creating = ref(false)

async function load() {
  const user = currentUser.value
  if (!user) {
    return
  }
  state.value = 'loading'
  try {
    const page = await listsApi.listReviewerLists({ reviewerId: user.id, restaurantEq: props.restaurantId })
    lists.value = page.items.filter((l) => l.kind === 'custom')
    state.value = 'done'
  } catch {
    state.value = 'failed'
  }
}

function setOpen(value: boolean) {
  if (value && !currentUser.value) {
    void router.push(signInPath(route.fullPath))
    return
  }
  if (value && tellPendingApproval()) {
    return
  }
  open.value = value
  if (value) {
    error.value = ''
    void load()
  }
}

async function toggle(list: SavedList, checked: boolean) {
  error.value = ''
  list.hasRestaurant = checked
  const request = { listRef: list.id, restaurantId: props.restaurantId }
  try {
    await (checked ? listsApi.addListRestaurant(request) : listsApi.removeListRestaurant(request))
  } catch (e) {
    list.hasRestaurant = !checked
    error.value = await listFailureText(e, `${list.name} did not change. Select it to try again.`)
  }
}

async function create() {
  if (creating.value || name.value.trim() === '') {
    return
  }
  creating.value = true
  error.value = ''
  try {
    const list = await listsApi.createList({ savedListCreate: { name: name.value.trim() } })
    name.value = ''
    lists.value = [{ ...list, hasRestaurant: false }, ...lists.value]
    // The list exists also when the add fails. Its checkbox is the retry.
    await toggle(lists.value[0]!, true)
  } catch (e) {
    error.value = await listFailureText(e, 'The list was not created. Try again in a moment.')
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <Popover :open="open" @update:open="setOpen">
    <PopoverTrigger as-child>
      <Button type="button" variant="outline"><ListPlusIcon aria-hidden="true" />Save to list</Button>
    </PopoverTrigger>
    <PopoverContent align="end" class="flex w-72 flex-col gap-3">
      <p id="save-to-list-title" class="text-sm font-semibold">Save to list</p>
      <p v-if="state === 'loading'" class="text-muted-foreground text-sm" aria-busy="true">Loading your lists</p>
      <p v-else-if="state === 'failed'" class="text-destructive text-sm" role="alert">
        Your lists did not load.
        <button type="button" class="text-foreground font-medium underline underline-offset-4" @click="load">Try again</button>
      </p>
      <template v-else>
        <ul v-if="lists.length" class="flex max-h-60 flex-col gap-1 overflow-y-auto" aria-labelledby="save-to-list-title">
          <li v-for="list in lists" :key="list.id">
            <label class="hover:bg-accent flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm">
              <input
                type="checkbox"
                class="accent-primary size-4"
                :checked="list.hasRestaurant"
                @change="toggle(list, ($event.target as HTMLInputElement).checked)"
              />
              <span class="min-w-0 break-words">{{ list.name }}</span>
            </label>
          </li>
        </ul>
        <p v-else class="text-muted-foreground text-sm">You have no lists yet. Make the first one.</p>
        <form class="flex gap-2" @submit.prevent="create">
          <label for="save-to-list-name" class="sr-only">Name of a new list</label>
          <Input id="save-to-list-name" v-model="name" placeholder="New list" maxlength="255" autocomplete="off" />
          <Button type="submit" variant="secondary" :disabled="creating">Create</Button>
        </form>
      </template>
      <p v-if="error" class="text-destructive text-sm" role="alert">{{ error }}</p>
    </PopoverContent>
  </Popover>
</template>
