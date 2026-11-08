<script setup lang="ts">
import type { Review } from '@reviews/sdk'
import { computed } from 'vue'
import { InitialsAvatar } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Card, CardAction, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { RatingDisplay } from '@/components/ui/rating-display'
import { kindOf, scoresOf, tagList } from '@/lib/reviews'
import DateText from './DateText.vue'
import KindName from './KindName.vue'

const props = defineProps<{ review: Review }>()

const scores = computed(() => scoresOf(props.review))
const tags = computed(() => tagList(props.review.tags))
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle as="h3" class="text-base leading-snug">
          <RouterLink :to="`/restaurants/${review.restaurant}/reviews/${review.id}`" class="rounded-sm hover:underline">
            {{ review.title }}
          </RouterLink>
        </CardTitle>
      <CardAction>
        <Badge variant="outline"><KindName :kind="kindOf(review)" /></Badge>
      </CardAction>
    </CardHeader>
    <CardContent class="grid gap-4">
      <div v-if="scores.length" class="grid gap-1.5">
        <RatingDisplay v-for="s in scores" :key="s.name" :value="s.value" :max="s.max" :label="s.label" />
      </div>
      <p v-if="review.feedback" class="line-clamp-3 whitespace-pre-line">{{ review.feedback }}</p>
    </CardContent>
    <CardFooter class="text-muted-foreground flex flex-wrap gap-2 text-sm">
      <Badge v-for="tag in tags" :key="tag" variant="secondary">{{ tag }}</Badge>
      <span class="flex items-center gap-2" :class="tags.length ? 'basis-full' : undefined">
        <InitialsAvatar :name="review.creator.name" size="sm" />
        <span>{{ review.creator.name }} · <DateText :created="review.created" :updated="review.updated" /></span>
      </span>
    </CardFooter>
  </Card>
</template>
