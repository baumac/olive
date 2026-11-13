<script setup lang="ts">
import type { Restaurant } from '@reviews/sdk'
import { MapPinIcon } from 'lucide-vue-next'
import { InitialsAvatar } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { RatingDisplay } from '@/components/ui/rating-display'
import { formatCount } from '@/lib/format'
import { tagList } from '@/lib/reviews'
import DateText from './DateText.vue'

defineProps<{ restaurant: Restaurant }>()
</script>

<template>
  <Card class="h-full">
    <CardHeader>
      <CardTitle as="h2" class="text-base leading-snug">
          <RouterLink :to="`/restaurants/${restaurant.id}`" class="rounded-sm hover:underline">
            {{ restaurant.name }}
          </RouterLink>
        </CardTitle>
      <CardDescription v-if="restaurant.description" class="line-clamp-3 whitespace-pre-line">
        {{ restaurant.description }}
      </CardDescription>
    </CardHeader>
    <CardContent class="grid gap-3">
      <p v-if="restaurant.location" class="text-muted-foreground flex items-start gap-1.5 text-sm">
        <MapPinIcon class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
        <span class="line-clamp-2"><span class="sr-only">Address: </span>{{ restaurant.location.address }}</span>
      </p>
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
        <RatingDisplay v-if="restaurant.rating" :value="restaurant.rating.overall" :max="10" label="Overall" />
        <span class="text-muted-foreground text-sm">
          {{ restaurant.reviewCount ? formatCount(restaurant.reviewCount, 'review') : 'No reviews yet' }}
        </span>
      </div>
      <div v-if="tagList(restaurant.tags).length" class="flex flex-wrap gap-1.5">
        <Badge v-for="tag in tagList(restaurant.tags)" :key="tag" variant="secondary">{{ tag }}</Badge>
      </div>
    </CardContent>
    <CardFooter class="text-muted-foreground mt-auto gap-2 text-sm">
      <InitialsAvatar :name="restaurant.creator?.name ?? null" size="sm" />
      <span>{{ restaurant.creator?.name ?? 'Deleted user' }} · <DateText :created="restaurant.created" :updated="restaurant.updated" /></span>
    </CardFooter>
  </Card>
</template>
