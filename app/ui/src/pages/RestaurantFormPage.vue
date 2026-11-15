<script setup lang="ts">
import type { Location, Place, Restaurant } from '@reviews/sdk'
import { CircleAlertIcon, MapPinIcon } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import ErrorPage from '@/components/app/ErrorPage.vue'
import LocationField from '@/components/app/LocationField.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { FormControl, FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { TagInput } from '@/components/ui/tag-input'
import { Textarea } from '@/components/ui/textarea'
import { usePageTitle } from '@/composables/usePageTitle'
import { apiFailure, fieldErrors, restaurantsApi, writeLimitText } from '@/lib/api'
import { clearDraft, saveDraft, takeDraft } from '@/lib/drafts'
import { nearbyRadius, sameLocation } from '@/lib/places'
import { normalizeWebsite, priceLevels } from '@/lib/restaurantDetails'
import { findOwnReviewId, restaurantAccess } from '@/lib/restaurants'
import { tagList } from '@/lib/reviews'
import { currentUser } from '@/lib/session'
import { checkLongText, checkName, checkOpeningHours, checkPhone, checkWebsite } from '@/lib/validation'

const props = defineProps<{ restaurantId?: string }>()
const router = useRouter()

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
// stored is the location of the restaurant before the edit.
const stored = ref<Location | null>(null)
const nearby = ref<Restaurant[]>([])
const saving = ref(false)
const nameLocked = ref(false)
const loading = ref(editing.value)
const failed = ref<{ status: number | null } | null>(null)
const draftKey = computed(() => `restaurant:${props.restaurantId ?? 'new'}`)
// The reviews of other users lock a stored location, like the name.
const locationLocked = computed(() => nameLocked.value && stored.value !== null)
const cancelPath = computed(() => (props.restaurantId ? `/restaurants/${props.restaurantId}` : '/restaurants'))

onMounted(async () => {
  if (props.restaurantId) {
    try {
      const userId = currentUser.value?.id
      const [r, hasOwnReview]: [Restaurant, boolean | null] = await Promise.all([
        restaurantsApi.getRestaurant({ restaurantId: props.restaurantId }),
        userId
          ? findOwnReviewId(props.restaurantId, userId).then(
              (id) => !!id,
              () => null,
            )
          : false,
      ])
      const access = restaurantAccess(r, userId, hasOwnReview)
      if (!access.canChange) {
        // Without a creator, the access depends on the failed lookup.
        failed.value = { status: !r.creator && hasOwnReview === null ? null : 403 }
        return
      }
      nameLocked.value = access.hasOtherReviews
      stored.value = r.location
      fields.value = {
        name: r.name,
        description: r.description ?? '',
        tags: tagList(r.tags),
        location: r.location,
        website: r.website ?? '',
        phone: r.phone ?? '',
        openingHours: r.openingHours ?? '',
        priceLevel: r.priceLevel ?? null,
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
    // A draft from before decision 00038 has no details.
    fields.value = {
      ...emptyFields,
      ...draft,
      name: nameLocked.value ? fields.value.name : draft.name,
      location: locationLocked.value ? fields.value.location : draft.location,
    }
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
    if (!location || sameLocation(location, stored.value)) {
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
    // A restaurant from before locations can stay without one.
    location: fields.value.location
      ? checkName(fields.value.location.address, 'Enter the address.')
      : editing.value && !stored.value
        ? ''
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
            name: nameLocked.value ? undefined : name,
            description: description || null,
            tags: tags.length ? new Set(tags) : null,
            location: !location || locationLocked.value || sameLocation(location, stored.value) ? undefined : location,
            website: website || null,
            phone: phone || null,
            openingHours: openingHours || null,
            priceLevel,
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
    void router.push(`/restaurants/${saved.id}`)
  } catch (error) {
    const { status, problem } = await apiFailure(error)
    // A 401 opens the sign in page. The draft then waits for the return.
    // After a 429, the draft waits for the end of the rate limit.
    if (status !== 401 && status !== 429) {
      clearDraft(draftKey.value)
    }
    if (status === 422) {
      errors.value = fieldErrors(problem)
    } else if (status === 409 && problem?.restaurant) {
      errors.value = { location: 'Another restaurant has this place.' }
      alert.value = {
        title: 'This restaurant is already here',
        description: 'Add your review to it, or select a different place.',
        restaurant: problem.restaurant,
      }
    } else if (status === 409 && (problem?.property === 'name' || problem?.property === 'location')) {
      const field = problem.property
      errors.value = { [field]: `The ${field} cannot change because other people have reviewed this restaurant.` }
    } else if (status === 409) {
      alert.value = { title: 'Restaurant not saved', description: 'Reload the page and try again.' }
    } else if (status === 403) {
      alert.value = { title: 'You cannot change this restaurant', description: 'Only the person who added it can change it. If that person has no account now, write a review first.' }
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
    <h1 class="text-4xl font-extrabold tracking-tight">{{ editing ? 'Edit restaurant' : 'Add a restaurant' }}</h1>
    <div v-if="loading" aria-busy="true" aria-label="Loading restaurant" class="grid gap-4">
      <Skeleton class="h-9" /><Skeleton class="h-24" /><Skeleton class="h-9" />
    </div>
    <form v-else class="grid gap-6" novalidate @submit.prevent="submit">
      <Alert v-if="alert" variant="destructive">
        <CircleAlertIcon aria-hidden="true" />
        <AlertTitle>{{ alert.title }}</AlertTitle>
        <AlertDescription>
          {{ alert.description }}
          <RouterLink v-if="alert.restaurant" :to="`/restaurants/${alert.restaurant}`" class="font-medium underline underline-offset-4">
            Open the restaurant
          </RouterLink>
        </AlertDescription>
      </Alert>
      <FormItem id="restaurant-name" :error="errors.name" :description="nameLocked">
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input
            v-model="fields.name"
            name="name"
            maxlength="255"
            required
            placeholder="Luigi's Trattoria"
            :readonly="nameLocked"
          />
        </FormControl>
        <FormDescription v-if="nameLocked">
          The name cannot change because other people have reviewed this restaurant.
        </FormDescription>
        <FormMessage />
      </FormItem>
      <FormItem id="restaurant-location" :error="errors.location" description>
        <FormLabel>Location</FormLabel>
        <LocationField
          id="restaurant-location"
          v-model="fields.location"
          :locked="locationLocked"
          :invalid="!!errors.location"
          :described-by="`restaurant-location-description${errors.location ? ' restaurant-location-message' : ''}`"
          @selected="usePlace"
        />
        <FormDescription>
          {{
            locationLocked
              ? 'The location cannot change because other people have reviewed this restaurant.'
              : 'Search for the restaurant or its address and select it. You can also enter the location by hand.'
          }}
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
              <template v-if="r.location">, {{ r.location.address }}</template>
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
          :invalid="!!errors.tags"
          :described-by="`restaurant-tags-description${errors.tags ? ' restaurant-tags-message' : ''}`"
          @error="errors = { ...errors, tags: $event }"
        />
        <FormDescription>Optional. Press Enter or comma after each tag, for example wood-fired.</FormDescription>
        <FormMessage />
      </FormItem>
      <fieldset class="grid gap-6">
        <legend class="mb-1 text-xl font-semibold tracking-tight">Details</legend>
        <p class="text-muted-foreground -mt-4 text-sm">Optional. A selected place can fill some of them.</p>
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
      <p v-if="nameLocked" class="text-muted-foreground text-sm">
        Other people have reviewed this restaurant. You cannot delete it.
      </p>
    </form>
  </main>
</template>
