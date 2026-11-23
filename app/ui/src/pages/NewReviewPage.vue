<script setup lang="ts">
import type { RecipeSummary, Restaurant } from '@reviews/sdk'
import { ChefHatIcon, UtensilsIcon } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import SubjectPicker from '@/components/app/SubjectPicker.vue'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { usePageTitle } from '@/composables/usePageTitle'
import { type ReviewKind, kinds } from '@/lib/reviewKinds'
import ReviewFormPage from './ReviewFormPage.vue'

// NewReviewPage is "Add review" of the navigation. The person selects the
// kind and the subject, then fills the review form. See decision 00005.
const props = defineProps<{ restaurantId?: string; recipeId?: string }>()
const router = useRouter()

usePageTitle('Add review')

const kind = ref<ReviewKind>(props.recipeId ? 'recipe' : 'restaurant')
const subject = ref<{ id: string; name: string } | null>(null)
const loadingSubject = ref(false)
const form = ref<InstanceType<typeof ReviewFormPage> | null>(null)

// The draft of each kind keeps the fields while the subject changes.
const draftName = computed(() => `review:new:${kind.value}`)

// A link from the subject form gives the new subject.
async function loadLinked() {
  const id = props.recipeId ?? props.restaurantId
  if (!id) {
    return
  }
  loadingSubject.value = true
  try {
    const parent = await kinds[kind.value].getParent(id)
    subject.value = { id: parent.id, name: parent.name }
  } catch {
    subject.value = null
  } finally {
    loadingSubject.value = false
  }
}
watch(() => [props.restaurantId, props.recipeId], () => void loadLinked(), { immediate: true })

function chooseKind(next: ReviewKind) {
  if (next !== kind.value) {
    form.value?.saveDraftNow()
    kind.value = next
    subject.value = null
  }
}

function select(item: Restaurant | RecipeSummary) {
  subject.value = { id: item.id, name: item.name }
}

function change() {
  form.value?.saveDraftNow()
  subject.value = null
}

function addNew() {
  form.value?.saveDraftNow()
  void router.push({ path: `${kinds[kind.value].path}/new`, query: { review: 'new' } })
}

const choices: { kind: ReviewKind; label: string; icon: typeof UtensilsIcon }[] = [
  { kind: 'restaurant', label: 'I ate out', icon: UtensilsIcon },
  { kind: 'recipe', label: 'I cooked', icon: ChefHatIcon },
]
</script>

<template>
  <main id="main" class="mx-auto flex max-w-2xl flex-col gap-6 px-4 py-8 sm:px-6">
    <h1 class="font-serif text-[34px] leading-10 font-semibold">Add review</h1>
    <section class="grid gap-4" aria-labelledby="what-heading">
      <h2 id="what-heading" class="font-serif text-[26px] leading-8 font-semibold">What did you eat?</h2>
      <div role="group" aria-labelledby="what-heading" class="flex gap-2.5">
        <button
          v-for="c in choices"
          :key="c.kind"
          type="button"
          :aria-pressed="kind === c.kind"
          :class="[
            'flex h-[88px] flex-1 flex-col items-center justify-center gap-2 rounded-2xl border text-[15px] font-semibold',
            kind === c.kind ? 'bg-accent text-accent-foreground border-primary border-2' : 'bg-card border-input hover:bg-accent',
          ]"
          @click="chooseKind(c.kind)"
        >
          <component :is="c.icon" class="size-[26px]" aria-hidden="true" />
          {{ c.label }}
        </button>
      </div>
    </section>
    <Skeleton v-if="loadingSubject" class="h-[74px] rounded-xl" />
    <div v-else-if="subject" class="grid gap-2">
      <p class="text-sm font-semibold">{{ kind === 'recipe' ? 'Recipe' : 'Restaurant' }}</p>
      <div class="bg-card border-input flex items-center gap-3 rounded-xl border p-3">
        <span class="bg-accent text-primary flex size-12 shrink-0 items-center justify-center rounded-lg">
          <component :is="kind === 'recipe' ? ChefHatIcon : UtensilsIcon" class="size-[22px]" aria-hidden="true" />
        </span>
        <span class="min-w-0 flex-1 font-serif text-lg font-semibold wrap-anywhere">{{ subject.name }}</span>
        <Button type="button" variant="outline" size="sm" :aria-label="`Change the ${kind === 'recipe' ? 'recipe' : 'restaurant'}`" @click="change">Change</Button>
      </div>
    </div>
    <SubjectPicker v-else :kind="kind" @select="select" @add-new="addNew" />
    <ReviewFormPage
      v-if="subject"
      ref="form"
      :key="`${kind}:${subject.id}`"
      embedded
      :draft-name="draftName"
      :restaurant-id="kind === 'restaurant' ? subject.id : undefined"
      :recipe-id="kind === 'recipe' ? subject.id : undefined"
    />
  </main>
</template>
