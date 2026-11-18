import { computed } from 'vue'
import { useRoute } from 'vue-router'
import type { ReviewKind } from '@/lib/reviewKinds'

// useKindQuery reads the kind of a page with KindSwitch. kind=recipes in the
// URL selects the recipes. The restaurants are the default.
export function useKindQuery() {
  const route = useRoute()
  return computed<ReviewKind>(() => (route.query.kind === 'recipes' ? 'recipe' : 'restaurant'))
}
