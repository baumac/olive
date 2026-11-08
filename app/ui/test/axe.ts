import axe from 'axe-core'
import { expect } from 'vitest'

// expectAccessible runs axe on an element. jsdom does not render. The color
// contrast rule therefore cannot run. tokens.json has the contrast values.
export async function expectAccessible(el: Element) {
  const results = await axe.run(el, { rules: { 'color-contrast': { enabled: false }, region: { enabled: false } } })
  expect(results.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(' | ')}`)).toEqual([])
}
