import { useColorMode } from '@vueuse/core'
import { computed } from 'vue'

export const themeOptions = [
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
  { value: 'auto', label: 'System' },
] as const

// useTheme gives the theme that the person selected. The value "auto"
// follows the system setting.
export function useTheme() {
  // The inline script in index.html reads the same storage key.
  const mode = useColorMode({ storageKey: 'vueuse-color-scheme', disableTransition: false })
  const choice = computed({
    get: () => mode.store.value,
    set: (v) => (mode.store.value = v),
  })
  const label = computed(() => themeOptions.find((o) => o.value === choice.value)?.label ?? 'System')
  return { choice, label }
}
