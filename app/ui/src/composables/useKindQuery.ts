import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { type ReviewKind, kindOfQuery } from '@/lib/reviewKinds'

// KindChoice is the selection of KindSwitch. all shows all kinds.
export type KindChoice = 'all' | ReviewKind

// useKindQuery reads the kind of a page with KindSwitch. kind=restaurants,
// kind=recipes, or kind=pantry in the URL selects one kind. All is the
// default.
export function useKindQuery() {
  const route = useRoute()
  return computed<KindChoice>(() => kindOfQuery(route.query.kind) ?? 'all')
}
