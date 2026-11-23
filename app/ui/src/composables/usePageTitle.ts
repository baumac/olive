import type { MaybeRefOrGetter } from 'vue'
import { toValue, watchEffect } from 'vue'

export function usePageTitle(title: MaybeRefOrGetter<string>) {
  watchEffect(() => {
    const t = toValue(title)
    document.title = t ? `${t} · Sponga47 Food Reviews` : 'Sponga47 Food Reviews'
  })
}
