import type { MaybeRefOrGetter } from 'vue'
import { toValue, watchEffect } from 'vue'

export function usePageTitle(title: MaybeRefOrGetter<string>) {
  watchEffect(() => {
    const t = toValue(title)
    document.title = t ? `${t} · Olive` : 'Olive, a food journal'
  })
}
