import { describe, expect, it } from 'vitest'
import indexHtml from '../index.html?raw'
import { usePageTitle } from '@/composables/usePageTitle'

// The app server builds the titles of link previews from the same tags.
describe('usePageTitle', () => {
  const index = new DOMParser().parseFromString(indexHtml, 'text/html')
  const siteName = index.querySelector('meta[property="og:site_name"]')?.getAttribute('content')

  it('ends a title with og:site_name of index.html', () => {
    usePageTitle('Ana')
    expect(document.title).toBe(`Ana · ${siteName}`)
  })

  it('uses the title of index.html without a page title', () => {
    usePageTitle('')
    expect(document.title).toBe(index.title)
  })
})
