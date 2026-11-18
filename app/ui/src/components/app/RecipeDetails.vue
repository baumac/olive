<script setup lang="ts">
import type { Recipe } from '@reviews/sdk'
import { ClockIcon, ExternalLinkIcon, UsersIcon } from 'lucide-vue-next'
import { formatCount, formatMinutes } from '@/lib/format'

defineProps<{ recipe: Recipe }>()
</script>

<template>
  <div class="grid gap-6">
    <ul v-if="recipe.totalTime || recipe.servings || recipe.source" class="text-muted-foreground flex flex-wrap gap-x-6 gap-y-2 text-sm">
      <li v-if="recipe.totalTime" class="flex items-center gap-1.5">
        <ClockIcon class="size-4 shrink-0" aria-hidden="true" />
        <span><span class="sr-only">Total time: </span>{{ formatMinutes(recipe.totalTime) }}</span>
      </li>
      <li v-if="recipe.servings" class="flex items-center gap-1.5">
        <UsersIcon class="size-4 shrink-0" aria-hidden="true" />
        <span>{{ formatCount(recipe.servings, 'serving') }}</span>
      </li>
      <li v-if="recipe.source" class="flex min-w-0 items-center gap-1.5">
        <ExternalLinkIcon class="size-4 shrink-0" aria-hidden="true" />
        <a :href="recipe.source" target="_blank" rel="noopener noreferrer nofollow" class="text-foreground truncate underline underline-offset-4">
          Original recipe<span class="sr-only">, opens a new tab</span>
        </a>
      </li>
    </ul>
    <div class="grid gap-8 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
      <section class="grid content-start gap-3" aria-labelledby="ingredients-heading">
        <h2 id="ingredients-heading" class="text-2xl font-semibold tracking-tight">Ingredients</h2>
        <ul class="grid list-disc gap-1.5 pl-5">
          <li v-for="(ingredient, i) in recipe.ingredients" :key="i" class="break-words">{{ ingredient }}</li>
        </ul>
      </section>
      <section class="grid content-start gap-3" aria-labelledby="steps-heading">
        <h2 id="steps-heading" class="text-2xl font-semibold tracking-tight">Steps</h2>
        <ol class="grid list-decimal gap-3 pl-5">
          <li v-for="(step, i) in recipe.steps" :key="i" class="leading-7 break-words whitespace-pre-line">{{ step }}</li>
        </ol>
      </section>
    </div>
  </div>
</template>
