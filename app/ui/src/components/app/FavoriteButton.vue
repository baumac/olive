<script setup lang="ts">
import { HeartIcon } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import { signInPath } from '@/composables/useAuth'
import { apiFailure, listsApi } from '@/lib/api'
import { currentUser } from '@/lib/session'

const props = withDefaults(
  defineProps<{
    restaurantId: string
    name: string
    favorite: boolean
    // labeled shows the text "Favorite" next to the heart.
    labeled?: boolean
  }>(),
  { labeled: false },
)

const emit = defineEmits<{ change: [favorite: boolean] }>()

const route = useRoute()
const router = useRouter()

const on = ref(props.favorite)
// A card can get a different restaurant with the same favorite value.
watch([() => props.restaurantId, () => props.favorite], () => (on.value = props.favorite))
const busy = ref(false)

// A restaurant that loaded before a sign out can still have the flag.
const pressed = computed(() => on.value && currentUser.value !== null)

async function toggle() {
  if (!currentUser.value) {
    void router.push(signInPath(route.fullPath))
    return
  }
  if (busy.value) {
    return
  }
  busy.value = true
  const next = !on.value
  // The heart changes before the response. A failed request changes it back.
  on.value = next
  const request = { listRef: 'favorites', restaurantId: props.restaurantId }
  try {
    await (next ? listsApi.addListRestaurant(request) : listsApi.removeListRestaurant(request))
    emit('change', next)
  } catch (error) {
    on.value = !next
    const { status } = await apiFailure(error)
    const description =
      status === 409
        ? 'Your favorites list is full.'
        : status === 404
          ? 'The restaurant does not exist now.'
          : status === 401
            ? 'Sign in again, then try again.'
            : 'Try again in a moment.'
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
