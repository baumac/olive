<script setup lang="ts">
import { ArrowDownIcon, ArrowUpIcon, CircleAlertIcon, PlusIcon, XIcon } from 'lucide-vue-next'
import { computed, nextTick, onMounted, ref } from 'vue'
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
import { apiFailure, fieldErrors, notApprovedText, recipesApi, writeLimitText } from '@/lib/api'
import { clearDraft, saveDraft, takeDraft } from '@/lib/drafts'
import { normalizeWebsite } from '@/lib/restaurantDetails'
import { parentAccess } from '@/lib/parents'
import { tagList } from '@/lib/reviews'
import { currentUser } from '@/lib/session'
import {
  checkIngredients,
  checkLongText,
  checkName,
  checkStep,
  checkWebsite,
  checkWholeNumber,
  ingredientLines,
  maxSteps,
} from '@/lib/validation'

const props = defineProps<{ recipeId?: string }>()
const router = useRouter()

const editing = computed(() => props.recipeId !== undefined)
usePageTitle(() => (editing.value ? 'Edit recipe' : 'Add a recipe'))

interface Fields {
  name: string
  description: string
  tags: string[]
  // ingredients has one ingredient on each line.
  ingredients: string
  steps: string[]
  source: string
  totalTime: string
  servings: string
}

const emptyFields = (): Fields => ({
  name: '',
  description: '',
  tags: [],
  ingredients: '',
  steps: [''],
  source: '',
  totalTime: '',
  servings: '',
})
const fields = ref<Fields>(emptyFields())
// stepKeys keep the focus and the text of a step when a step moves.
let nextKey = 0
const stepKeys = ref<number[]>([nextKey++])
const errors = ref<Record<string, string>>({})
const alert = ref<{ title: string; description: string; recipe?: string } | null>(null)
const saving = ref(false)
// locked is true when other people reviewed the recipe. The API then
// refuses a change of the name, the ingredients, the steps, and a stored
// source.
const locked = ref(false)
const storedSource = ref('')
// sourceConflict keeps the lock of a source conflict when the stored source
// did not load.
const sourceConflict = ref(false)
const sourceLocked = computed(() => locked.value && (storedSource.value !== '' || sourceConflict.value))
const lockedText = computed(
  () =>
    `Other people have reviewed this recipe. The name, the ingredients, ${sourceLocked.value ? 'the steps, and the source' : 'and the steps'} cannot change, and you cannot delete it.`,
)
const loading = ref(editing.value)
const failed = ref<{ status: number | null } | null>(null)
const draftKey = computed(() => `recipe:${props.recipeId ?? 'new'}`)
const cancelPath = computed(() => (props.recipeId ? `/recipes/${props.recipeId}` : '/recipes'))

// showStoredSource gets the source after a conflict. The page can have an old
// source. A locked field then shows the value that the API keeps.
async function showStoredSource(property: string) {
  try {
    const r = await recipesApi.getRecipe({ recipeId: props.recipeId! })
    storedSource.value = r.source ?? ''
  } catch {
    sourceConflict.value ||= property === 'source'
  }
  if (sourceLocked.value && storedSource.value !== '') {
    fields.value.source = storedSource.value
  }
}

function setSteps(steps: string[]) {
  fields.value.steps = steps.length ? steps : ['']
  stepKeys.value = fields.value.steps.map(() => nextKey++)
}

onMounted(async () => {
  if (props.recipeId) {
    try {
      const r = await recipesApi.getRecipe({ recipeId: props.recipeId })
      const access = parentAccess(r, currentUser.value?.id)
      if (!access.canChange) {
        failed.value = { status: 403 }
        return
      }
      locked.value = access.hasOtherReviews
      storedSource.value = r.source ?? ''
      fields.value = {
        name: r.name,
        description: r.description ?? '',
        tags: tagList(r.tags),
        ingredients: r.ingredients.join('\n'),
        steps: [],
        source: r.source ?? '',
        totalTime: r.totalTime ? String(r.totalTime) : '',
        servings: r.servings ? String(r.servings) : '',
      }
      setSteps(r.steps)
    } catch (error) {
      failed.value = { status: (await apiFailure(error)).status }
      return
    } finally {
      loading.value = false
    }
  }
  const draft = takeDraft<Fields>(draftKey.value)
  if (draft) {
    const current = fields.value
    fields.value = {
      ...emptyFields(),
      ...draft,
      name: locked.value ? current.name : draft.name,
      ingredients: locked.value ? current.ingredients : draft.ingredients,
      source: sourceLocked.value ? current.source : draft.source,
    }
    setSteps(locked.value ? current.steps : draft.steps)
  }
})

async function addStep() {
  fields.value.steps.push('')
  stepKeys.value.push(nextKey++)
  await nextTick()
  document.getElementById(`recipe-step-${fields.value.steps.length - 1}`)?.focus()
}

async function removeStep(i: number) {
  fields.value.steps.splice(i, 1)
  stepKeys.value.splice(i, 1)
  await nextTick()
  document.getElementById(`recipe-step-${Math.min(i, fields.value.steps.length - 1)}`)?.focus()
}

async function moveStep(i: number, by: number) {
  const j = i + by
  const steps = fields.value.steps
  const keys = stepKeys.value
  ;[steps[i], steps[j]] = [steps[j]!, steps[i]!]
  ;[keys[i], keys[j]] = [keys[j]!, keys[i]!]
  await nextTick()
  document.getElementById(`recipe-step-${j}-${by < 0 ? 'up' : 'down'}`)?.focus()
}

function validate(): boolean {
  const f = fields.value
  f.source = normalizeWebsite(f.source)
  const next: Record<string, string> = {
    name: checkName(f.name, 'Enter a name with at least one character that is not a space.'),
    description: checkLongText(f.description),
    tags: errors.value.tags ?? '',
    ingredients: checkIngredients(ingredientLines(f.ingredients)),
    source: checkWebsite(f.source),
    total_time: checkWholeNumber(f.totalTime, 1, 10080),
    servings: checkWholeNumber(f.servings, 1, 100),
  }
  f.steps.forEach((step, i) => {
    next[`step-${i}`] = checkStep(step)
  })
  errors.value = next
  const order = ['name', 'description', 'tags', 'ingredients', ...f.steps.map((_, i) => `step-${i}`), 'source', 'total_time', 'servings']
  const first = order.find((k) => next[k])
  if (first) {
    document.getElementById(`recipe-${first}`)?.focus()
  }
  return !first
}

// withStepErrors gives each step field the API error of its own item. An
// error of the steps array itself stays on steps.
function withStepErrors(errs: Record<string, string>, problem: Parameters<typeof fieldErrors>[0]): Record<string, string> {
  const items = (problem?.errors ?? []).filter((e) => /^steps\[\d+\]/.test(e.property))
  for (const e of items) {
    const index = /^steps\[(\d+)\]/.exec(e.property)![1]
    errs[`step-${index}`] ??= fieldErrors({ ...problem!, errors: [e] }).steps ?? e.message
  }
  if (items.length && !(problem?.errors ?? []).some((e) => e.property === 'steps')) {
    delete errs.steps
  }
  return errs
}

async function submit() {
  alert.value = null
  if (!validate()) {
    return
  }
  saving.value = true
  saveDraft(draftKey.value, fields.value)
  const f = fields.value
  const ingredients = ingredientLines(f.ingredients)
  const totalTime = f.totalTime.trim() ? Number(f.totalTime) : null
  const servings = f.servings.trim() ? Number(f.servings) : null
  try {
    const saved = props.recipeId
      ? await recipesApi.updateRecipe({
          recipeId: props.recipeId,
          recipePatch: {
            name: locked.value ? undefined : f.name,
            description: f.description || null,
            tags: f.tags.length ? new Set(f.tags) : null,
            ingredients: locked.value ? undefined : ingredients,
            steps: locked.value ? undefined : f.steps,
            source: sourceLocked.value ? undefined : f.source || null,
            totalTime,
            servings,
          },
        })
      : await recipesApi.createRecipe({
          recipeCreate: {
            name: f.name,
            description: f.description || undefined,
            tags: f.tags.length ? new Set(f.tags) : undefined,
            ingredients,
            steps: f.steps,
            source: f.source || undefined,
            totalTime: totalTime ?? undefined,
            servings: servings ?? undefined,
          },
        })
    clearDraft(draftKey.value)
    toast.success(editing.value ? 'Recipe saved' : 'Recipe added', { description: saved.name })
    void router.push(`/recipes/${saved.id}`)
  } catch (error) {
    const { status, problem } = await apiFailure(error)
    const code = problem?.code
    // A 401 opens the sign in page. The draft then waits for the return.
    // After a 429, the draft waits for the end of the rate limit.
    if (status !== 401 && status !== 429) {
      clearDraft(draftKey.value)
    }
    if (status === 422) {
      errors.value = withStepErrors(fieldErrors(problem), problem)
    } else if (code === 'source_taken') {
      errors.value = { source: 'Another recipe has this source.' }
      alert.value = {
        title: 'This recipe is already here',
        description: 'Add your review to it, or enter a different source.',
        recipe: problem?.recipe,
      }
    } else if (code === 'other_reviews' && problem?.property) {
      errors.value = { [problem.property]: `The ${problem.property} cannot change because other people have reviewed this recipe.` }
      locked.value = true
      await showStoredSource(problem.property)
      await nextTick()
      document.getElementById(`recipe-${problem.property}`)?.focus()
    } else if (status === 409) {
      alert.value = { title: 'Recipe not saved', description: 'Reload the page and try again.' }
    } else if (code === 'not_approved') {
      alert.value = { title: 'Your account waits for approval', description: notApprovedText }
    } else if (status === 403) {
      alert.value = {
        title: 'You cannot change this recipe',
        description: 'Only the person who added it can change it. If that person has no account now, write a review first.',
      }
    } else if (status === 404) {
      alert.value = { title: 'This recipe no longer exists', description: 'Someone deleted it. Go back to the recipes.' }
    } else if (status === 429) {
      alert.value = { title: 'Recipe not saved', description: writeLimitText }
    } else if (status !== 401) {
      alert.value = { title: 'Recipe not saved', description: 'Check your connection and try again in a moment.' }
    }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <ErrorPage v-if="failed" :status="failed.status" @retry="router.go(0)" />
  <main v-else id="main" class="mx-auto flex max-w-2xl flex-col gap-6 px-4 py-8 sm:px-6">
    <h1 class="text-4xl font-extrabold tracking-tight">{{ editing ? 'Edit recipe' : 'Add a recipe' }}</h1>
    <div v-if="loading" aria-busy="true" aria-label="Loading recipe" class="grid gap-4">
      <Skeleton class="h-9" /><Skeleton class="h-24" /><Skeleton class="h-9" />
    </div>
    <form v-else class="grid gap-6" novalidate @submit.prevent="submit">
      <Alert v-if="alert" variant="destructive">
        <CircleAlertIcon aria-hidden="true" />
        <AlertTitle>{{ alert.title }}</AlertTitle>
        <AlertDescription>
          {{ alert.description }}
          <RouterLink v-if="alert.recipe" :to="`/recipes/${alert.recipe}`" class="font-medium underline underline-offset-4">
            Open the recipe
          </RouterLink>
        </AlertDescription>
      </Alert>
      <p v-if="locked" class="text-muted-foreground text-sm">{{ lockedText }}</p>
      <FormItem id="recipe-name" :error="errors.name">
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input v-model="fields.name" name="name" maxlength="255" required placeholder="Shakshuka" :readonly="locked" />
        </FormControl>
        <FormMessage />
      </FormItem>
      <FormItem id="recipe-description" :error="errors.description" description>
        <FormLabel>Description</FormLabel>
        <FormControl>
          <Textarea v-model="fields.description" name="description" maxlength="10000" rows="3" />
        </FormControl>
        <FormDescription>Optional. What kind of dish is it?</FormDescription>
        <FormMessage />
      </FormItem>
      <FormItem id="recipe-tags" :error="errors.tags" description>
        <FormLabel>Tags</FormLabel>
        <TagInput
          id="recipe-tags"
          v-model="fields.tags"
          :invalid="!!errors.tags"
          :described-by="`recipe-tags-description${errors.tags ? ' recipe-tags-message' : ''}`"
          @error="errors = { ...errors, tags: $event }"
        />
        <FormDescription>Optional. Press Enter or comma after each tag, for example breakfast.</FormDescription>
        <FormMessage />
      </FormItem>
      <FormItem id="recipe-ingredients" :error="errors.ingredients" description>
        <FormLabel>Ingredients</FormLabel>
        <FormControl>
          <Textarea
            v-model="fields.ingredients"
            name="ingredients"
            rows="6"
            required
            :readonly="locked"
            placeholder="6 eggs&#10;1 can of tomatoes&#10;1 onion"
          />
        </FormControl>
        <FormDescription>One ingredient with its amount on each line.</FormDescription>
        <FormMessage />
      </FormItem>
      <fieldset id="recipe-steps" class="grid gap-4" tabindex="-1" :aria-describedby="errors.steps ? 'recipe-steps-message' : undefined">
        <legend class="mb-1 text-sm font-medium">Steps</legend>
        <p v-if="errors.steps" id="recipe-steps-message" class="text-destructive text-sm" role="alert">{{ errors.steps }}</p>
        <ol class="grid gap-4">
          <li v-for="(_, i) in fields.steps" :key="stepKeys[i]" class="grid gap-2">
            <label :for="`recipe-step-${i}`" class="text-sm font-medium">Step {{ i + 1 }}</label>
            <Textarea
              :id="`recipe-step-${i}`"
              v-model="fields.steps[i]"
              :name="`step-${i + 1}`"
              rows="3"
              maxlength="2000"
              :readonly="locked"
              :aria-invalid="!!errors[`step-${i}`]"
              :aria-describedby="errors[`step-${i}`] ? `recipe-step-${i}-message` : undefined"
            />
            <p v-if="errors[`step-${i}`]" :id="`recipe-step-${i}-message`" class="text-destructive text-sm">{{ errors[`step-${i}`] }}</p>
            <div v-if="!locked" class="flex flex-wrap gap-2">
              <Button
                :id="`recipe-step-${i}-up`"
                type="button"
                variant="outline"
                size="sm"
                :disabled="i === 0"
                :aria-label="`Move step ${i + 1} up`"
                @click="moveStep(i, -1)"
              >
                <ArrowUpIcon aria-hidden="true" />
              </Button>
              <Button
                :id="`recipe-step-${i}-down`"
                type="button"
                variant="outline"
                size="sm"
                :disabled="i === fields.steps.length - 1"
                :aria-label="`Move step ${i + 1} down`"
                @click="moveStep(i, 1)"
              >
                <ArrowDownIcon aria-hidden="true" />
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                :disabled="fields.steps.length === 1"
                :aria-label="`Remove step ${i + 1}`"
                @click="removeStep(i)"
              >
                <XIcon aria-hidden="true" />Remove
              </Button>
            </div>
          </li>
        </ol>
        <Button v-if="!locked && fields.steps.length < maxSteps" type="button" variant="secondary" class="w-fit" @click="addStep">
          <PlusIcon aria-hidden="true" />Add step
        </Button>
      </fieldset>
      <fieldset class="grid gap-6">
        <legend class="mb-1 text-xl font-semibold tracking-tight">Details</legend>
        <p class="text-muted-foreground text-sm">Optional.</p>
        <FormItem id="recipe-source" :error="errors.source" description>
          <FormLabel>Source</FormLabel>
          <FormControl>
            <Input
              v-model="fields.source"
              name="source"
              type="url"
              inputmode="url"
              maxlength="2048"
              placeholder="https://recipes.example/shakshuka"
              :readonly="sourceLocked"
            />
          </FormControl>
          <FormDescription>The web page of the original recipe.</FormDescription>
          <FormMessage />
        </FormItem>
        <FormItem id="recipe-total_time" :error="errors.total_time" description>
          <FormLabel>Total time in minutes</FormLabel>
          <FormControl>
            <Input v-model="fields.totalTime" name="total-time" inputmode="numeric" maxlength="5" class="w-full sm:w-40" />
          </FormControl>
          <FormDescription>From the start of the first step to the end of the last step.</FormDescription>
          <FormMessage />
        </FormItem>
        <FormItem id="recipe-servings" :error="errors.servings">
          <FormLabel>Servings</FormLabel>
          <FormControl>
            <Input v-model="fields.servings" name="servings" inputmode="numeric" maxlength="3" class="w-full sm:w-40" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </fieldset>
      <div class="flex flex-wrap gap-2">
        <Button type="submit" :disabled="saving">
          {{ saving ? 'Saving…' : editing ? 'Save recipe' : 'Add recipe' }}
        </Button>
        <Button variant="outline" as-child><RouterLink :to="cancelPath">Cancel</RouterLink></Button>
      </div>
    </form>
  </main>
</template>
