<script setup lang="ts">
import type { Location, Place, Restaurant } from '@reviews/sdk'
import { CircleAlertIcon, MapPinIcon } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import ErrorPage from '@/components/app/ErrorPage.vue'
import LocationField from '@/components/app/LocationField.vue'
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
import { apiFailure, fieldErrors, notApprovedText, restaurantsApi, writeLimitText } from '@/lib/api'
import { clearDraft, saveDraft, takeDraft } from '@/lib/drafts'
import { type Undo, findUndo, undoNumber } from '@/lib/history'
import { nearbyRadius, sameLocation } from '@/lib/places'
import { normalizeWebsite, priceLevels } from '@/lib/restaurantDetails'
import { changedPatch, parentAccess } from '@/lib/parents'
import { tagList } from '@/lib/reviews'
import { tagSuggester } from '@/lib/tags'
import { currentUser } from '@/lib/session'
import { checkLongText, checkName, checkOpeningHours, checkPhone, checkWebsite } from '@/lib/validation'

const suggestTags = tagSuggester('restaurants')
const props = defineProps<{ restaurantId?: string }>()
const router = useRouter()
const route = useRoute()

const editing = computed(() => props.restaurantId !== undefined)
usePageTitle(() => (editing.value ? 'Edit restaurant' : 'Add a restaurant'))

interface Fields {
  name: string
  description: string
  tags: string[]
  location: Location | null
  website: string
  phone: string
  openingHours: string
  priceLevel: number | null
}

const emptyFields: Fields = { name: '', description: '', tags: [], location: null, website: '', phone: '', openingHours: '', priceLevel: null }
const fields = ref<Fields>({ ...emptyFields })
const errors = ref<Record<string, string>>({})
const alert = ref<{ title: string; description: string; restaurant?: string } | null>(null)
const nearby = ref<Restaurant[]>([])
// loaded has the values of the restaurant before the edit.
const loaded = ref<Fields>({ ...emptyFields })
const saving = ref(false)
// deleteBlocked is true when the reviews of other users block the delete of
// the creator. Each user can change the restaurant. See decision 00004.
const deleteBlocked = ref(false)
const loading = ref(editing.value)
const failed = ref<{ status: number | null } | null>(null)
const draftKey = computed(() => `restaurant:${props.restaurantId ?? 'new'}`)
// undo is the change that ?undo of the history page undoes. See decision
// 00004.
const undo = ref<Undo | null | undefined>(undefined)
// A new restaurant from the review form returns to it. See decision 00005.
const fromReview = computed(() => !props.restaurantId && route.query.review === 'new')
const cancelPath = computed(() => {
  if (fromReview.value) {
    return '/reviews/new'
  }
  return props.restaurantId ? `/restaurants/${props.restaurantId}` : '/restaurants'
})
// subjectPath opens a restaurant that exists. From the review form, it
// opens a review of it.
function subjectPath(id: string) {
  return fromReview.value ? `/reviews/new?restaurant=${id}` : `/restaurants/${id}`
}

function fieldsOf(r: Restaurant): Fields {
  return {
    name: r.name,
    description: r.description ?? '',
    tags: tagList(r.tags),
    location: r.location,
    website: r.website ?? '',
    phone: r.phone ?? '',
    openingHours: r.openingHours ?? '',
    priceLevel: r.priceLevel ?? null,
  }
}

onMounted(async () => {
  if (props.restaurantId) {
    try {
      const r = await restaurantsApi.getRestaurant({ restaurantId: props.restaurantId })
      const access = parentAccess(r, currentUser.value?.id)
      deleteBlocked.value = access.isCreator && access.hasOtherReviews
      loaded.value = fieldsOf(r)
      fields.value = { ...loaded.value, tags: [...loaded.value.tags] }
      const number = undoNumber(route.query.undo)
      if (number) {
        undo.value = await findUndo('restaurant', r.id, number).catch(() => null)
        if (undo.value) {
          fields.value = fieldsOf({ ...r, ...undo.value.values })
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
    // A draft of an older build can miss a field.
    fields.value = { ...emptyFields, ...draft }
  }
})


// A restaurant near the new location can be the same restaurant. The check
// is a hint. A failed check shows nothing.
let nearbyCheck = 0
watch(
  () => fields.value.location,
  async (location) => {
    const check = ++nearbyCheck
    nearby.value = []
    if (!location || sameLocation(location, loaded.value.location)) {
      return
    }
    try {
      const page = await restaurantsApi.listRestaurants({
        near: { latitude: location.latitude, longitude: location.longitude, radius: nearbyRadius },
        limit: 6,
      })
      if (check === nearbyCheck) {
        nearby.value = page.items.filter((r) => r.id !== props.restaurantId).slice(0, 5)
      }
    } catch {
      nearby.value = []
    }
  },
)

// usePlace fills only the empty fields. The name of a stored restaurant
// stays.
function usePlace(place: Place) {
  const f = fields.value
  if (place.name && !editing.value && f.name.trim() === '') {
    f.name = place.name
  }
  if (place.website && f.website.trim() === '') {
    f.website = place.website
  }
  if (place.phone && f.phone.trim() === '') {
    f.phone = place.phone
  }
  if (place.openingHours && f.openingHours.trim() === '') {
    f.openingHours = place.openingHours
  }
}

// The Select of Reka UI does not allow an empty value.
const noPriceLevel = 'none'
const priceLevelValue = computed({
  get: () => (fields.value.priceLevel === null ? noPriceLevel : String(fields.value.priceLevel)),
  set: (v: string) => {
    fields.value.priceLevel = v === noPriceLevel ? null : Number(v)
  },
})

function validate(): boolean {
  fields.value.website = normalizeWebsite(fields.value.website)
  fields.value.phone = fields.value.phone.trim()
  fields.value.openingHours = fields.value.openingHours.trim()
  errors.value = {
    name: checkName(fields.value.name, 'Enter a name with at least one character that is not a space.'),
    description: checkLongText(fields.value.description),
    tags: errors.value.tags ?? '',
    location: fields.value.location
      ? checkName(fields.value.location.address, 'Enter the address.')
      : 'Search for the place and select it, or enter the location by hand.',
    website: checkWebsite(fields.value.website),
    phone: checkPhone(fields.value.phone),
    opening_hours: checkOpeningHours(fields.value.openingHours),
  }
  const first = ['name', 'location', 'description', 'tags', 'website', 'phone', 'opening_hours'].find((k) => errors.value[k])
  if (first) {
    document.getElementById(`restaurant-${first}`)?.focus()
  }
  return !first
}

// patchOf has the properties of a patch except the location.
function patchOf(f: Fields) {
  return {
    name: f.name,
    description: f.description || null,
    tags: f.tags.length ? new Set(f.tags) : null,
    website: f.website || null,
    phone: f.phone || null,
    openingHours: f.openingHours || null,
    priceLevel: f.priceLevel,
  }
}

async function submit() {
  alert.value = null
  if (!validate()) {
    return
  }
  saving.value = true
  saveDraft(draftKey.value, fields.value)
  const { name, description, tags, location, priceLevel } = fields.value
  const { website, phone, openingHours } = fields.value
  try {
    const saved = props.restaurantId
      ? await restaurantsApi.updateRestaurant({
          restaurantId: props.restaurantId,
          restaurantPatch: {
            ...changedPatch(patchOf(fields.value), patchOf(loaded.value)),
            location: !location || sameLocation(location, loaded.value.location) ? undefined : location,
          },
        })
      : await restaurantsApi.createRestaurant({
          restaurantCreate: {
            name,
            description: description || undefined,
            tags: tags.length ? new Set(tags) : undefined,
            location: location!,
            website: website || undefined,
            phone: phone || undefined,
            openingHours: openingHours || undefined,
            priceLevel: priceLevel ?? undefined,
          },
        })
    clearDraft(draftKey.value)
    toast.success(editing.value ? 'Restaurant saved' : 'Restaurant added', { description: saved.name })
    if (fromReview.value) {
      void router.push({ path: '/reviews/new', query: { restaurant: saved.id } })
      return
    }
    void router.push(`/restaurants/${saved.id}`)
  } catch (error) {
    const { status, problem } = await apiFailure(error)
    const code = problem?.code
    // A 401 opens the sign in page. The draft then waits for the return.
    // After a 429, the draft waits for the end of the rate limit.
    if (status !== 401 && status !== 429) {
      clearDraft(draftKey.value)
    }
    if (status === 422) {
      errors.value = fieldErrors(problem)
    } else if (code === 'place_taken' && problem?.restaurant) {
      errors.value = { location: 'Another restaurant has this place.' }
      alert.value = {
        title: 'This restaurant is already here',
        description: 'Add your review to it, or select a different place.',
        restaurant: problem.restaurant,
      }
    } else if (status === 409) {
      alert.value = { title: 'Restaurant not saved', description: 'Reload the page and try again.' }
    } else if (code === 'not_approved') {
      alert.value = { title: 'Your account waits for approval', description: notApprovedText }
    } else if (status === 403) {
      alert.value = { title: 'Restaurant not saved', description: 'Reload the page and try again.' }
    } else if (status === 404) {
      alert.value = { title: 'This restaurant no longer exists', description: 'Someone deleted it. Go back to the restaurants.' }
    } else if (status === 429) {
      alert.value = { title: 'Restaurant not saved', description: writeLimitText }
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
    <h1 class="font-serif text-[34px] leading-10 font-semibold tracking-[-0.01em]">{{ editing ? 'Edit restaurant' : 'Add a restaurant' }}</h1>
    <div v-if="loading" aria-busy="true" aria-label="Loading restaurant" class="grid gap-4">
      <Skeleton class="h-9" /><Skeleton class="h-24" /><Skeleton class="h-9" />
    </div>
    <form v-else class="grid gap-6" novalidate @submit.prevent="submit">
      <UndoAlert v-if="undo !== undefined" kind="restaurant" :undo="undo" />
      <Alert v-if="alert" variant="destructive">
        <CircleAlertIcon aria-hidden="true" />
        <AlertTitle>{{ alert.title }}</AlertTitle>
        <AlertDescription>
          {{ alert.description }}
          <RouterLink v-if="alert.restaurant" :to="subjectPath(alert.restaurant)" class="font-medium underline underline-offset-4">
            {{ fromReview ? 'Review this restaurant' : 'Open the restaurant' }}
          </RouterLink>
        </AlertDescription>
      </Alert>
      <FormItem id="restaurant-name" :error="errors.name">
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input
            v-model="fields.name"
            name="name"
            maxlength="255"
            required
            placeholder="Luigi's Trattoria"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
      <FormItem id="restaurant-location" :error="errors.location" description>
        <FormLabel>Location</FormLabel>
        <LocationField
          id="restaurant-location"
          v-model="fields.location"
          :invalid="!!errors.location"
          :described-by="`restaurant-location-description${errors.location ? ' restaurant-location-message' : ''}`"
          @selected="usePlace"
        />
        <FormDescription>
          Search for the restaurant or its address and select it. You can also enter the location by hand.
        </FormDescription>
        <FormMessage />
      </FormItem>
      <Alert v-if="nearby.length">
        <MapPinIcon aria-hidden="true" />
        <AlertTitle>Is it one of these?</AlertTitle>
        <AlertDescription>
          <p>These restaurants are at this location. Open one to add your review to it.</p>
          <ul class="list-disc pl-5">
            <li v-for="r in nearby" :key="r.id">
              <RouterLink :to="`/restaurants/${r.id}`" class="font-medium underline underline-offset-4">{{ r.name }}</RouterLink>
              , {{ r.location.address }}
            </li>
          </ul>
        </AlertDescription>
      </Alert>
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
          :suggest="suggestTags"
          :invalid="!!errors.tags"
          :described-by="`restaurant-tags-description${errors.tags ? ' restaurant-tags-message' : ''}`"
          @error="errors = { ...errors, tags: $event }"
        />
        <FormDescription>Optional. Press Enter or comma after each tag, for example wood-fired.</FormDescription>
        <FormMessage />
      </FormItem>
      <fieldset class="grid gap-6">
        <legend class="mb-1 font-serif text-[22px] leading-7 font-semibold">Details</legend>
        <p class="text-muted-foreground text-sm">Optional. A selected place can fill some of them.</p>
        <FormItem id="restaurant-website" :error="errors.website">
          <FormLabel>Website</FormLabel>
          <FormControl>
            <Input
              v-model="fields.website"
              name="website"
              type="url"
              inputmode="url"
              autocomplete="url"
              maxlength="2048"
              placeholder="https://cansole.example"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
        <FormItem id="restaurant-phone" :error="errors.phone">
          <FormLabel>Phone</FormLabel>
          <FormControl>
            <Input v-model="fields.phone" name="phone" type="tel" autocomplete="tel" maxlength="40" placeholder="+34 932 21 50 12" />
          </FormControl>
          <FormMessage />
        </FormItem>
        <FormItem id="restaurant-opening_hours" :error="errors.opening_hours" description>
          <FormLabel>Opening hours</FormLabel>
          <FormControl>
            <Textarea v-model="fields.openingHours" name="opening-hours" maxlength="500" rows="3" />
          </FormControl>
          <FormDescription>For example, Tu-Sa 13:00-16:00 on one line and Su 13:00-16:00 on the next.</FormDescription>
          <FormMessage />
        </FormItem>
        <FormItem id="restaurant-price_level" :error="errors.price_level" description>
          <FormLabel>Price level</FormLabel>
          <Select v-model="priceLevelValue">
            <FormControl>
              <SelectTrigger class="w-full sm:w-60">
                <SelectValue />
              </SelectTrigger>
            </FormControl>
            <SelectContent>
              <SelectItem :value="noPriceLevel">Not set</SelectItem>
              <SelectItem v-for="l in priceLevels" :key="l.value" :value="String(l.value)">{{ l.label }}</SelectItem>
            </SelectContent>
          </Select>
          <FormDescription>How much a meal costs, compared to other restaurants in the city.</FormDescription>
          <FormMessage />
        </FormItem>
      </fieldset>
      <div class="flex flex-wrap gap-2">
        <Button type="submit" :disabled="saving">
          {{ saving ? 'Saving…' : editing ? 'Save restaurant' : 'Add restaurant' }}
        </Button>
        <Button variant="outline" as-child><RouterLink :to="cancelPath">Cancel</RouterLink></Button>
      </div>
      <p v-if="deleteBlocked" class="text-muted-foreground text-sm">
        Other people have reviewed this restaurant. You cannot delete it.
      </p>
    </form>
  </main>
</template>
