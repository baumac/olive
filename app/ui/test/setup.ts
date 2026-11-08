import { config } from '@vue/test-utils'
import { afterEach } from 'vitest'

// RouterLink renders as a plain link. The tests do not need a router.
config.global.stubs = { RouterLink: { props: ['to'], template: '<a :href="String(to)"><slot /></a>' } }

afterEach(() => {
  localStorage.clear()
  sessionStorage.clear()
  document.body.innerHTML = ''
})
