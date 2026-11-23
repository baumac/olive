import { computed } from 'vue'
import { useRoute } from 'vue-router'
import type { ReviewKind } from '@/lib/reviewKinds'

// KindChoice is the selection of KindSwitch. all shows both kinds.
export type KindChoice = 'all' | ReviewKind

// useKindQuery reads the kind of a page with KindSwitch. kind=restaurants or
// kind=recipes in the URL selects one kind. All is the default.
export function useKindQuery() {
  const route = useRoute()
  return computed<KindChoice>(() => {
    if (route.query.kind === 'recipes') {
      return 'recipe'
    }
    return route.query.kind === 'restaurants' ? 'restaurant' : 'all'
  })
}
