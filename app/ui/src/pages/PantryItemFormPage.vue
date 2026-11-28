<script setup lang="ts">
import type { PantryItem, PantryItemCategory } from '@reviews/sdk'
import { CircleAlertIcon } from 'lucide-vue-next'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import ErrorPage from '@/components/app/ErrorPage.vue'
import UndoAlert from '@/components/app/UndoAlert.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { FormControl, FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { TagInput } from '@/components/ui/tag-input'
import { Textarea } from '@/components/ui/textarea'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, fieldErrors, notApprovedText, pantryApi, writeLimitText } from '@/lib/api'
import { clearDraft, saveDraft, takeDraft } from '@/lib/drafts'
import { type Undo, findUndo, undoNumber } from '@/lib/history'
import { categories, isCategory, pantryItemMeta, pantryIcon } from '@/lib/pantry'
import { changedPatch, parentAccess } from '@/lib/parents'
import { tagList } from '@/lib/reviews'
import { tagSuggester } from '@/lib/tags'
import { currentUser } from '@/lib/session'
import { checkLongText, checkName } from '@/lib/validation'

const suggestTags = tagSuggester('pantry')
const props = defineProps<{ pantryItemId?: string }>()
const router = useRouter()
const route = useRoute()

const editing = computed(() => props.pantryItemId !== undefined)
usePageTitle(() => (editing.value ? 'Edit pantry item' : 'Add a pantry item'))

interface Fields {
  name: string
  // category is empty until the person selects one.
  category: PantryItemCategory | ''
  producer: string
  origin: string
  style: string
  description: string
  tags: string[]
}

const emptyFields = (): Fields => ({ name: '', category: '', producer: '', origin: '', style: '', description: '', tags: [] })
const fields = ref<Fields>(emptyFields())
const errors = ref<Record<string, string>>({})
const alert = ref<{ title: string; description: string } | null>(null)
const saving = ref(false)
// loaded has the values of the pantry item before the edit.
const loaded = ref<Fields>(emptyFields())
// deleteBlocked is true when the tastings of other users block the delete
// of the creator. Each user can change the pantry item. See decision 00004.
const deleteBlocked = ref(false)
const loading = ref(editing.value)
const failed = ref<{ status: number | null } | null>(null)
const draftKey = computed(() => `pantry-item:${props.pantryItemId ?? 'new'}`)
// undo is the change that ?undo of the history page undoes. See decision
// 00004.
const undo = ref<Undo | null | undefined>(undefined)
// A new pantry item from the review form returns to it. See decision 00005.
const fromReview = computed(() => !props.pantryItemId && route.query.review === 'new')
const cancelPath = computed(() => {
  if (fromReview.value) {
    return '/reviews/new?kind=pantry'
  }
  return props.pantryItemId ? `/pantry/${props.pantryItemId}` : '/pantry'
})
// subjectPath opens a pantry item that exists. From the review form, it
// opens a tasting of it.
function subjectPath(id: string) {
  return fromReview.value ? `/reviews/new?pantry_item=${id}` : `/pantry/${id}`
}

function fieldsOf(p: Pick<PantryItem, 'name' | 'category' | 'producer' | 'origin' | 'style' | 'description' | 'tags'>): Fields {
  return {
    name: p.name,
    category: isCategory(p.category) ? p.category : '',
    producer: p.producer ?? '',
    origin: p.origin ?? '',
    style: p.style ?? '',
    description: p.description ?? '',
    tags: tagList(p.tags),
  }
}

onMounted(async () => {
  if (props.pantryItemId) {
    try {
      const p = await pantryApi.getPantryItem({ pantryItemId: props.pantryItemId })
      const access = parentAccess(p, currentUser.value?.id)
      deleteBlocked.value = access.isCreator && access.hasOtherReviews
      loaded.value = fieldsOf(p)
      fields.value = { ...loaded.value, tags: [...loaded.value.tags] }
      const number = undoNumber(route.query.undo)
      if (number) {
        undo.value = await findUndo('pantryItem', p.id, number).catch(() => null)
        if (undo.value) {
          fields.value = fieldsOf({ ...p, ...undo.value.values })
        }
      }
    } catch (error) {
      failed.value = { status: (await apiFailure(error)).status }
      return
    } finally {
      loading.value = false
    }
  }
  const draft = takeDraft<Fields>(draftKey.value)
  if (draft) {
    fields.value = { ...emptyFields(), ...draft }
  }
})

// sameName has the pantry items with the name of the form. Two wines of one
// producer can share a name. The check is a hint. A failed check shows
// nothing. See decision 00012.
const sameName = ref<PantryItem[]>([])
let nameCheck = 0
let nameTimer: ReturnType<typeof setTimeout> | undefined
onBeforeUnmount(() => clearTimeout(nameTimer))

function sameText(a: string, b: string): boolean {
  return a.trim().toLocaleLowerCase() === b.trim().toLocaleLowerCase()
}

watch(
  () => fields.value.name,
  (name) => {
    clearTimeout(nameTimer)
    const check = ++nameCheck
    sameName.value = []
    if (!/[\p{L}\p{N}]/u.test(name) || (editing.value && sameText(name, loaded.value.name))) {
      return
    }
    nameTimer = setTimeout(async () => {
      try {
        const page = await pantryApi.listPantryItems({ search: name.trim().slice(0, 255), limit: 20 })
        if (check === nameCheck) {
          sameName.value = page.items.filter((p) => p.id !== props.pantryItemId && sameText(p.name, name)).slice(0, 5)
        }
      } catch {
        sameName.value = []
      }
    }, 400)
  },
)

// The Select of Reka UI does not allow an empty value. The placeholder
// shows for no value.
const categoryValue = computed({
  get: () => fields.value.category || undefined,
  set: (v: string | undefined) => {
    fields.value.category = isCategory(v) ? v : ''
  },
})

// optionalName checks a detail only when it has a value.
function optionalName(v: string): string {
  return v.trim() ? checkName(v, 'Enter at least one character that is not a space.') : ''
}

function validate(): boolean {
  const f = fields.value
  const next: Record<string, string> = {
    name: checkName(f.name, 'Enter a name with at least one character that is not a space.'),
    category: f.category ? '' : 'Select a category.',
    producer: optionalName(f.producer),
    origin: optionalName(f.origin),
    style: optionalName(f.style),
    description: checkLongText(f.description),
    tags: errors.value.tags ?? '',
  }
  errors.value = next
  const first = ['name', 'category', 'producer', 'origin', 'style', 'description', 'tags'].find((k) => next[k])
  if (first) {
    document.getElementById(`pantry-item-${first}`)?.focus()
  }
  return !first
}

function patchOf(f: Fields) {
  return {
    name: f.name,
    category: f.category || undefined,
    producer: f.producer.trim() ? f.producer : null,
    origin: f.origin.trim() ? f.origin : null,
    style: f.style.trim() ? f.style : null,
    description: f.description || null,
    tags: f.tags.length ? new Set(f.tags) : null,
  }
}

async function submit() {
  alert.value = null
  if (!validate()) {
    return
  }
  saving.value = true
  saveDraft(draftKey.value, fields.value)
  const f = fields.value
  try {
    const saved = props.pantryItemId
      ? await pantryApi.updatePantryItem({
          pantryItemId: props.pantryItemId,
          pantryItemPatch: { ...changedPatch(patchOf(f), patchOf(loaded.value)) },
        })
      : await pantryApi.createPantryItem({
          pantryItemCreate: {
            name: f.name,
            category: f.category as PantryItemCategory,
            producer: f.producer.trim() ? f.producer : undefined,
            origin: f.origin.trim() ? f.origin : undefined,
            style: f.style.trim() ? f.style : undefined,
            description: f.description || undefined,
            tags: f.tags.length ? new Set(f.tags) : undefined,
          },
        })
    clearDraft(draftKey.value)
    toast.success(editing.value ? 'Pantry item saved' : 'Pantry item added', { description: saved.name })
    if (fromReview.value) {
      void router.push({ path: '/reviews/new', query: { pantry_item: saved.id } })
      return
    }
    void router.push(`/pantry/${saved.id}`)
  } catch (error) {
    const { status, problem } = await apiFailure(error)
    // A 401 opens the sign in page. The draft then waits for the return.
    // After a 429, the draft waits for the end of the rate limit.
    if (status !== 401 && status !== 429) {
      clearDraft(draftKey.value)
    }
    if (status === 422) {
      errors.value = fieldErrors(problem)
    } else if (problem?.code === 'not_approved') {
      alert.value = { title: 'Your account waits for approval', description: notApprovedText }
    } else if (status === 403) {
      alert.value = { title: 'Pantry item not saved', description: 'Reload the page and try again.' }
    } else if (status === 404) {
      alert.value = { title: 'This pantry item no longer exists', description: 'Someone deleted it. Go back to the pantry.' }
    } else if (status === 429) {
      alert.value = { title: 'Pantry item not saved', description: writeLimitText }
    } else if (status !== 401) {
      alert.value = { title: 'Pantry item not saved', description: 'Check your connection and try again in a moment.' }
    }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <ErrorPage v-if="failed" :status="failed.status" @retry="router.go(0)" />
  <main v-else id="main" class="mx-auto flex max-w-2xl flex-col gap-6 px-4 py-8 sm:px-6">
    <h1 class="font-serif text-[34px] leading-10 font-semibold tracking-[-0.01em]">{{ editing ? 'Edit pantry item' : 'Add a pantry item' }}</h1>
    <div v-if="loading" aria-busy="true" aria-label="Loading pantry item" class="grid gap-4">
      <Skeleton class="h-9" /><Skeleton class="h-9" /><Skeleton class="h-24" />
    </div>
    <form v-else class="grid gap-6" novalidate @submit.prevent="submit">
      <UndoAlert v-if="undo !== undefined" kind="pantryItem" :undo="undo" />
      <Alert v-if="alert" variant="destructive">
        <CircleAlertIcon aria-hidden="true" />
        <AlertTitle>{{ alert.title }}</AlertTitle>
        <AlertDescription>{{ alert.description }}</AlertDescription>
      </Alert>
      <FormItem id="pantry-item-name" :error="errors.name">
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input v-model="fields.name" name="name" maxlength="255" required placeholder="Wehlener Sonnenuhr Kabinett" />
        </FormControl>
        <FormMessage />
      </FormItem>
      <Alert v-if="sameName.length">
        <component :is="pantryIcon" aria-hidden="true" />
        <AlertTitle>Is it one of these?</AlertTitle>
        <AlertDescription>
          <p>These pantry items have the same name. Open one to add your tasting to it.</p>
          <ul class="list-disc pl-5">
            <li v-for="p in sameName" :key="p.id">
              <RouterLink :to="subjectPath(p.id)" class="font-medium underline underline-offset-4">{{ p.name }}</RouterLink>
              , {{ pantryItemMeta(p) }}
            </li>
          </ul>
        </AlertDescription>
      </Alert>
      <FormItem id="pantry-item-category" :error="errors.category">
        <FormLabel>Category</FormLabel>
        <Select v-model="categoryValue">
          <FormControl>
            <SelectTrigger class="w-full sm:w-60">
              <SelectValue placeholder="Select a category" />
            </SelectTrigger>
          </FormControl>
          <SelectContent>
            <SelectItem v-for="c in categories" :key="c.value" :value="c.value">{{ c.label }}</SelectItem>
          </SelectContent>
        </Select>
        <FormMessage />
      </FormItem>
      <fieldset class="grid gap-6">
        <legend class="mb-1 font-serif text-[22px] leading-7 font-semibold">Details</legend>
        <p class="text-muted-foreground text-sm">Optional.</p>
        <FormItem id="pantry-item-producer" :error="errors.producer" description>
          <FormLabel>Producer</FormLabel>
          <FormControl>
            <Input v-model="fields.producer" name="producer" maxlength="255" placeholder="Weingut Example" />
          </FormControl>
          <FormDescription>The winery, the brewery, the dairy, or the chocolate maker.</FormDescription>
          <FormMessage />
        </FormItem>
        <FormItem id="pantry-item-origin" :error="errors.origin">
          <FormLabel>Origin</FormLabel>
          <FormControl>
            <Input v-model="fields.origin" name="origin" maxlength="255" placeholder="Mosel, Germany" />
          </FormControl>
          <FormMessage />
        </FormItem>
        <FormItem id="pantry-item-style" :error="errors.style" description>
          <FormLabel>Style</FormLabel>
          <FormControl>
            <Input v-model="fields.style" name="style" maxlength="255" placeholder="Riesling" />
          </FormControl>
          <FormDescription>The grape, the beer style, the type of cheese, or the cocoa content.</FormDescription>
          <FormMessage />
        </FormItem>
        <FormItem id="pantry-item-description" :error="errors.description" description>
          <FormLabel>Description</FormLabel>
          <FormControl>
            <Textarea v-model="fields.description" name="description" maxlength="10000" rows="3" />
          </FormControl>
          <FormDescription>What is it like?</FormDescription>
          <FormMessage />
        </FormItem>
        <FormItem id="pantry-item-tags" :error="errors.tags" description>
          <FormLabel>Tags</FormLabel>
          <TagInput
            id="pantry-item-tags"
            v-model="fields.tags"
            :suggest="suggestTags"
            :invalid="!!errors.tags"
            :described-by="`pantry-item-tags-description${errors.tags ? ' pantry-item-tags-message' : ''}`"
            @error="errors = { ...errors, tags: $event }"
          />
          <FormDescription>Press Enter or comma after each tag, for example raw-milk or 2019. Tags are saved in lowercase with hyphens.</FormDescription>
          <FormMessage />
        </FormItem>
      </fieldset>
      <div class="flex flex-wrap gap-2">
        <Button type="submit" :disabled="saving">
          {{ saving ? 'Saving…' : editing ? 'Save pantry item' : 'Add pantry item' }}
        </Button>
        <Button variant="outline" as-child><RouterLink :to="cancelPath">Cancel</RouterLink></Button>
      </div>
      <p v-if="deleteBlocked" class="text-muted-foreground text-sm">Other people have tasted this pantry item. You cannot delete it.</p>
    </form>
  </main>
</template>
