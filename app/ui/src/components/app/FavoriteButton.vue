<script setup lang="ts">
import { HeartIcon } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import { signInPath } from '@/composables/useAuth'
import { apiFailure, failureText } from '@/lib/api'
import { tellPendingApproval } from '@/lib/approval'
import { kinds } from '@/lib/reviewKinds'
import { currentUser } from '@/lib/session'

// Give a restaurantId or a recipeId.
const props = withDefaults(
  defineProps<{
    restaurantId?: string
    recipeId?: string
    name: string
    favorite: boolean
    // labeled shows the text "Favorite" next to the heart.
    labeled?: boolean
  }>(),
  { restaurantId: undefined, recipeId: undefined, labeled: false },
)

const kind = computed(() => (props.recipeId !== undefined ? kinds.recipe : kinds.restaurant))
const itemId = computed(() => (props.recipeId ?? props.restaurantId)!)

const emit = defineEmits<{ change: [favorite: boolean] }>()

const route = useRoute()
const router = useRouter()

const on = ref(props.favorite)
// A card can get a different restaurant or recipe with the same favorite
// value.
watch([itemId, () => props.favorite], () => (on.value = props.favorite))
const busy = ref(false)

// A restaurant or a recipe that loaded before a sign out can still have the
// flag.
const pressed = computed(() => on.value && currentUser.value !== null)

async function toggle() {
  if (!currentUser.value) {
    void router.push(signInPath(route.fullPath))
    return
  }
  if (busy.value || tellPendingApproval()) {
    return
  }
  busy.value = true
  const next = !on.value
  // The heart changes before the response. A failed request changes it back.
  on.value = next
  try {
    await (next ? kind.value.addToList('favorites', itemId.value) : kind.value.removeFromList('favorites', itemId.value))
    emit('change', next)
  } catch (error) {
    on.value = !next
    const failure = await apiFailure(error)
    const description =
      failure.problem?.code === 'list_restaurant_limit' || failure.problem?.code === 'list_recipe_limit'
        ? 'Your favorites list is full.'
        : failure.status === 404
          ? `The ${kind.value.noun} does not exist now.`
          : failureText(failure)
    toast.error(next ? 'Not added to favorites' : 'Not removed from favorites', { description })
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <!-- The name is the same in the two states. aria-pressed tells the state. -->
  <Button
    type="button"
    :variant="labeled ? 'outline' : 'secondary'"
    :size="labeled ? 'default' : 'icon'"
    :class="labeled ? '' : 'rounded-full shadow-sm'"
    :aria-label="`Favorite ${name}`"
    :aria-pressed="pressed"
    @click="toggle"
  >
    <HeartIcon :class="pressed ? 'fill-current' : ''" aria-hidden="true" />
    <template v-if="labeled">Favorite</template>
  </Button>
</template>
