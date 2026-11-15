<script setup lang="ts">
import type { Restaurant } from '@reviews/sdk'
import { BanknoteIcon, ClockIcon, GlobeIcon, PhoneIcon } from 'lucide-vue-next'
import { computed } from 'vue'
import { priceLevelLabel, telLink, websiteHost } from '@/lib/restaurantDetails'

const props = defineProps<{ restaurant: Restaurant }>()

const priceLevel = computed(() => priceLevelLabel(props.restaurant.priceLevel))
const hasDetails = computed(() => {
  const r = props.restaurant
  return !!(r.website || r.phone || r.openingHours || priceLevel.value)
})
</script>

<template>
  <dl v-if="hasDetails" class="grid gap-2" aria-label="Details">
    <div v-if="restaurant.website" class="flex items-start gap-2">
      <dt class="flex">
        <GlobeIcon class="text-muted-foreground mt-1 size-4 shrink-0" aria-hidden="true" />
        <span class="sr-only">Website</span>
      </dt>
      <dd class="min-w-0 break-words">
        <!-- A user typed the URL. -->
        <a :href="restaurant.website" target="_blank" rel="nofollow ugc noopener noreferrer" class="underline underline-offset-4">
          {{ websiteHost(restaurant.website) }}<span class="sr-only">, opens in a new tab</span>
        </a>
      </dd>
    </div>
    <div v-if="restaurant.phone" class="flex items-start gap-2">
      <dt class="flex">
        <PhoneIcon class="text-muted-foreground mt-1 size-4 shrink-0" aria-hidden="true" />
        <span class="sr-only">Phone</span>
      </dt>
      <dd><a :href="telLink(restaurant.phone)" class="underline underline-offset-4">{{ restaurant.phone }}</a></dd>
    </div>
    <div v-if="restaurant.openingHours" class="flex items-start gap-2">
      <dt class="flex">
        <ClockIcon class="text-muted-foreground mt-1 size-4 shrink-0" aria-hidden="true" />
        <span class="sr-only">Opening hours</span>
      </dt>
      <dd class="min-w-0 break-words whitespace-pre-line">{{ restaurant.openingHours }}</dd>
    </div>
    <div v-if="priceLevel" class="flex items-start gap-2">
      <dt class="flex gap-2">
        <BanknoteIcon class="text-muted-foreground mt-1 size-4 shrink-0" aria-hidden="true" />
        <span>Price level:</span>
      </dt>
      <dd>{{ priceLevel }}</dd>
    </div>
  </dl>
</template>
