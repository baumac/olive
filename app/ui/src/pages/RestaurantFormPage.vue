<script setup lang="ts">
import type { Restaurant } from '@reviews/sdk'
import { CircleAlertIcon } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import ErrorPage from '@/components/app/ErrorPage.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { FormControl, FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { TagInput } from '@/components/ui/tag-input'
import { Textarea } from '@/components/ui/textarea'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, fieldErrors, restaurantsApi } from '@/lib/api'
import { clearDraft, saveDraft, takeDraft } from '@/lib/drafts'
import { tagList } from '@/lib/reviews'
import { currentUser } from '@/lib/session'
import { checkLongText, checkName } from '@/lib/validation'

const props = defineProps<{ restaurantId?: string }>()
const router = useRouter()

const editing = computed(() => props.restaurantId !== undefined)
usePageTitle(() => (editing.value ? 'Edit restaurant' : 'Add a restaurant'))

interface Fields {
  name: string
  description: string
  tags: string[]
}

const fields = ref<Fields>({ name: '', description: '', tags: [] })
const errors = ref<Record<string, string>>({})
const alert = ref<{ title: string; description: string } | null>(null)
const saving = ref(false)
const loading = ref(editing.value)
const failed = ref<{ status: number | null } | null>(null)
const draftKey = computed(() => `restaurant:${props.restaurantId ?? 'new'}`)
const cancelPath = computed(() => (props.restaurantId ? `/restaurants/${props.restaurantId}` : '/restaurants'))

onMounted(async () => {
  if (props.restaurantId) {
    try {
      const r: Restaurant = await restaurantsApi.getRestaurant({ restaurantId: props.restaurantId })
      if (r.creator?.id !== currentUser.value?.id) {
        failed.value = { status: 403 }
        return
      }
      fields.value = { name: r.name, description: r.description ?? '', tags: tagList(r.tags) }
    } catch (error) {
      failed.value = { status: (await apiFailure(error)).status }
      return
    } finally {
      loading.value = false
    }
  }
  const draft = takeDraft<Fields>(draftKey.value)
  if (draft) {
    fields.value = draft
  }
})

function validate(): boolean {
  errors.value = {
    name: checkName(fields.value.name, 'Enter a name with at least one character that is not a space.'),
    description: checkLongText(fields.value.description),
    tags: errors.value.tags ?? '',
  }
  const first = ['name', 'description', 'tags'].find((k) => errors.value[k])
  if (first) {
    document.getElementById(`restaurant-${first}`)?.focus()
  }
  return !first
}

async function submit() {
  alert.value = null
  if (!validate()) {
    return
  }
  saving.value = true
  saveDraft(draftKey.value, fields.value)
  const { name, description, tags } = fields.value
  try {
    const saved = props.restaurantId
      ? await restaurantsApi.updateRestaurant({
          restaurantId: props.restaurantId,
          restaurantPatch: { name, description: description || null, tags: tags.length ? new Set(tags) : null },
        })
      : await restaurantsApi.createRestaurant({
          restaurantCreate: { name, description: description || undefined, tags: tags.length ? new Set(tags) : undefined },
        })
    clearDraft(draftKey.value)
    toast.success(editing.value ? 'Restaurant saved' : 'Restaurant added', { description: saved.name })
    void router.push(`/restaurants/${saved.id}`)
  } catch (error) {
    const { status, problem } = await apiFailure(error)
    // A 401 opens the sign in page. The draft then waits for the return.
    if (status !== 401) {
      clearDraft(draftKey.value)
    }
    if (status === 422) {
      errors.value = fieldErrors(problem)
    } else if (status === 409) {
      errors.value = { name: 'The name cannot change because the restaurant has reviews.' }
    } else if (status === 403) {
      alert.value = { title: 'You cannot change this restaurant', description: 'Only the person who added it can change it.' }
    } else if (status === 404) {
      alert.value = { title: 'This restaurant no longer exists', description: 'Someone deleted it. Go back to the restaurants.' }
    } else if (status !== 401) {
      alert.value = { title: 'Restaurant not saved', description: 'Check your connection and try again in a moment.' }
    }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <ErrorPage v-if="failed" :status="failed.status" @retry="router.go(0)" />
  <main v-else id="main" class="mx-auto flex max-w-2xl flex-col gap-6 px-4 py-8 sm:px-6">
    <h1 class="text-4xl font-extrabold tracking-tight">{{ editing ? 'Edit restaurant' : 'Add a restaurant' }}</h1>
    <div v-if="loading" aria-busy="true" aria-label="Loading restaurant" class="grid gap-4">
      <Skeleton class="h-9" /><Skeleton class="h-24" /><Skeleton class="h-9" />
    </div>
    <form v-else class="grid gap-6" novalidate @submit.prevent="submit">
      <Alert v-if="alert" variant="destructive">
        <CircleAlertIcon aria-hidden="true" />
        <AlertTitle>{{ alert.title }}</AlertTitle>
        <AlertDescription>{{ alert.description }}</AlertDescription>
      </Alert>
      <FormItem id="restaurant-name" :error="errors.name">
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input v-model="fields.name" name="name" maxlength="255" required placeholder="Luigi's Trattoria" />
        </FormControl>
        <FormMessage />
      </FormItem>
      <FormItem id="restaurant-description" :error="errors.description" description>
        <FormLabel>Description</FormLabel>
        <FormControl>
          <Textarea v-model="fields.description" name="description" maxlength="10000" rows="4" />
        </FormControl>
        <FormDescription>Optional. What kind of place is it?</FormDescription>
        <FormMessage />
      </FormItem>
      <FormItem id="restaurant-tags" :error="errors.tags" description>
        <FormLabel>Tags</FormLabel>
        <TagInput
          id="restaurant-tags"
          v-model="fields.tags"
          :invalid="!!errors.tags"
          :described-by="`restaurant-tags-description${errors.tags ? ' restaurant-tags-message' : ''}`"
          @error="errors = { ...errors, tags: $event }"
        />
        <FormDescription>Optional. Press Enter or comma after each tag, for example wood-fired.</FormDescription>
        <FormMessage />
      </FormItem>
      <div class="flex flex-wrap gap-2">
        <Button type="submit" :disabled="saving">
          {{ saving ? 'Saving…' : editing ? 'Save restaurant' : 'Add restaurant' }}
        </Button>
        <Button variant="outline" as-child><RouterLink :to="cancelPath">Cancel</RouterLink></Button>
      </div>
    </form>
  </main>
</template>
