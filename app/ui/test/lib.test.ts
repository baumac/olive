import type { Problem } from '@reviews/sdk'
import { describe, expect, it } from 'vitest'
import spec from '../../../specs/api/public-openapi.yaml?raw'
import { apiFailure, failureText, fieldErrors, notApprovedText, notCreatorText } from '@/lib/api'
import { listFailureText } from '@/lib/lists'
import { restaurantFilterDefs, restaurantFilterParams } from '@/lib/restaurantFilters'
import { reviewFilterParams } from '@/lib/reviewFilters'
import { formatCount, formatDate, formatDay, formatScore, initials, localDay, parseDay, parseTimestamp } from '@/lib/format'
import { changedPatch, ownReviewsText, parentAccess } from '@/lib/parents'
import { directionsLink } from '@/lib/places'
import { overallOf, scoreFields } from '@/lib/reviews'
import { checkLongText, checkName, checkPassword, checkReviewDay, checkTag, safeRedirect, suggestTag } from '@/lib/validation'
import { rejectWith } from './problem'

describe('format', () => {
  const now = new Date('2024-11-10T12:00:00.000Z')

  it('reads the timestamp format of the API and of an old saved session', () => {
    expect(parseTimestamp('2024-11-04T01:26:55.002Z').toISOString()).toBe('2024-11-04T01:26:55.002Z')
    expect(parseTimestamp('2024-11-04 01:26:55.002Z').toISOString()).toBe('2024-11-04T01:26:55.002Z')
  })

  it('shows relative time for the last 7 days and a date after', () => {
    expect(formatDate('2024-11-10T11:59:40.000Z', now)).toBe('just now')
    expect(formatDate('2024-11-10T11:55:00.000Z', now)).toBe('5 minutes ago')
    expect(formatDate('2024-11-10T09:00:00.000Z', now)).toBe('3 hours ago')
    expect(formatDate('2024-11-09T10:00:00.000Z', now)).toBe('yesterday')
    expect(formatDate('2024-11-06T10:00:00.000Z', now)).toBe('4 days ago')
    expect(formatDate('2024-11-01T10:00:00.000Z', now)).toBe('1 Nov 2024')
    expect(formatDate('2024-11-10T12:00:03.000Z', now)).toBe('just now')
    expect(formatDate('2024-11-20T12:00:00.000Z', now)).toBe('20 Nov 2024')
  })

  it('formats counts with plurals', () => {
    expect(formatCount(1, 'review')).toBe('1 review')
    expect(formatCount(0, 'review')).toBe('0 reviews')
    expect(formatCount(1204, 'review')).toBe('1,204 reviews')
  })

  it('makes initials', () => {
    expect(initials('Ana')).toBe('AN')
    expect(initials(null)).toBe('?')
  })
})

describe('validation', () => {
  it('checks tags like the API', () => {
    expect(checkTag('wood-fired', [])).toBe('')
    expect(checkTag('Wood Fired', [])).toContain('wood-fired')
    expect(checkTag('pasta', ['pasta'])).toContain('already')
    expect(checkTag('x', Array.from({ length: 20 }, (_, i) => `t${i}`))).toContain('at most 20')
    expect(suggestTag('  Date Night!! ')).toBe('date-night')
  })

  it('accepts the letters and digits of all scripts in tags', () => {
    for (const tag of ['café', 'ラーメン', 'चाय', '2-für-1', 'ψάρι']) {
      expect(checkTag(tag, [])).toBe('')
    }
    // The API changes a separate accent to the composed form.
    expect(checkTag('cafe\u0301', [])).toBe('')
    expect(checkTag('pasta\ufe0f', [])).toContain('pasta')
    expect(checkTag('pasta\u17b4', [])).toContain('pasta')
    expect(checkTag('Ψάρι', [])).toContain('ψάρι')
    expect(checkTag('\u3164', [])).not.toBe('')
    expect(suggestTag('Crème Brûlée')).toBe('crème-brûlée')
    expect(suggestTag('İstanbul')).toBe('istanbul')
    expect(suggestTag('\u0301a-\u0301b')).toBe('a-b')
    expect(suggestTag('😀'.repeat(70))).toBe('')
    expect(Array.from(suggestTag('é'.repeat(70)))).toHaveLength(63)
  })

  it('refuses hidden characters like the API', () => {
    expect(checkName('Luigi\u202es', 'empty')).toContain('hidden')
    expect(checkName('Luigi\u200b', 'empty')).toContain('hidden')
    expect(checkName('Luigi\u061cs', 'empty')).toContain('hidden')
    expect(checkName('Piz\u00adza', 'empty')).toContain('hidden')
    expect(checkName('\u3164\u3164', 'empty')).toContain('hidden')
    expect(checkName('Family \u{1F468}\u200d\u{1F469}', 'empty')).toBe('')
    expect(checkName('\u{1F3F4}\u{E0067}\u{E0062}\u{E0073}\u{E0063}\u{E0074}\u{E007F} pub', 'empty')).toBe('')
    expect(checkLongText('Line one\nLine two')).toBe('')
    expect(checkLongText('Bad\u2066')).toContain('hidden')
  })

  it('checks the password length in characters and bytes', () => {
    expect(checkPassword('short')).not.toBe('')
    expect(checkPassword('correct-horse')).toBe('')
    expect(checkPassword('é'.repeat(40))).not.toBe('')
  })

  it('accepts only redirects to this site', () => {
    expect(safeRedirect('/restaurants/abc')).toBe('/restaurants/abc')
    expect(safeRedirect('//evil.example')).toBe('/')
    expect(safeRedirect('/\\evil.example')).toBe('/')
    expect(safeRedirect('https://evil.example')).toBe('/')
    expect(safeRedirect(undefined)).toBe('/')
  })
})

describe('fieldErrors', () => {
  it('maps array items to their property', () => {
    const errors = fieldErrors({
      type: 'about:blank',
      title: 'Unprocessable Content',
      status: 422,
      errors: [
        { property: 'tags[1]', message: 'must be unique' },
        { property: 'name', message: 'must have 1 to 255 characters' },
      ],
    })
    expect(errors).toEqual({ tags: 'Must be unique.', name: 'Must have 1 to 255 characters.' })
  })
})

describe('reviewFilterParams', () => {
  it('changes the filters to SDK parameters', () => {
    expect(reviewFilterParams({ search: 'carbonara', food: '4' })).toEqual({
      search: 'carbonara',
      overall: undefined,
      food: { gte: 4 },
      service: undefined,
      space: undefined,
      price: undefined,
    })
    expect(reviewFilterParams({ overall: '8' }).overall).toEqual({ gte: 8 })
  })
})

describe('ratings', () => {
  it('keeps one decimal of an average', () => {
    expect([formatScore(7.6), formatScore(9), formatScore(10)]).toEqual(['7.6', '9', '10'])
  })

  it('names the price score by its meaning', () => {
    expect(scoreFields.map((f) => `${f.name} ${f.label}`)).toEqual([
      'space Space', 'service Service', 'food Food', 'price Value',
    ])
  })
})

describe('restaurantFilterParams', () => {
  it('changes the filters to SDK parameters', () => {
    expect(restaurantFilterParams({ search: 'lui', tags: ['pasta'], overall: '8', list: 'l7d2f9h4j1k8m3p', price_level: '1-2' })).toEqual({
      search: 'lui',
      tagsHasAny: ['pasta'],
      overall: { gte: 8 },
      listEq: 'l7d2f9h4j1k8m3p',
      priceLevel: { gte: 1, lte: 2 },
    })
    expect(restaurantFilterParams({})).toEqual({ search: undefined, tagsHasAny: undefined, overall: undefined, listEq: undefined, priceLevel: undefined })
  })

  it('selects a range of price levels', () => {
    const def = restaurantFilterDefs.find((d) => d.key === 'price_level')!
    expect(def.options?.map((o) => `${o.value} ${o.label}`)).toEqual([
      '1 Inexpensive',
      '1-2 Moderate or less',
      '1-3 Expensive or less',
      '2-4 Moderate or more',
      '3-4 Expensive or more',
      '4 Very expensive',
    ])
    expect(def.chip('3-4')).toBe('Price level: Expensive or more')
    expect(def.chip('9')).toBe('Price level: 9')
    expect(restaurantFilterParams({ price_level: '4' }).priceLevel).toEqual({ eq: 4 })
    for (const v of ['2', '1-4', 'x', '']) {
      expect(restaurantFilterParams({ price_level: v }).priceLevel, v).toBeUndefined()
    }
    expect(restaurantFilterParams({ price_level: ['1'] }).priceLevel).toBeUndefined()
  })
})

describe('scoreFields', () => {
  it('has the score descriptions of the API spec', () => {
    for (const f of scoreFields) {
      const schema = `${f.name.charAt(0).toUpperCase()}${f.name.slice(1)}Score`
      const match = new RegExp(`\\n {4}${schema}:\\n {6}description: (.+)\\n`).exec(spec)
      expect(match?.[1], schema).toBe(f.description)
    }
  })
})

describe('parentAccess', () => {
  const creator = { id: 'u5n1c8r3t6y0b2e', name: 'Ana' }
  const other = 'u2g6k0m4o8q1s5w'
  const of = (reviewCount: number, ownReviewCount: number, c: typeof creator | null = creator) => ({ creator: c, reviewCount, ownReviewCount })

  it('lets the creator delete the restaurant', () => {
    expect(parentAccess(of(0, 0), creator.id)).toEqual({ canEdit: true, isCreator: true, hasOtherReviews: false })
    expect(parentAccess(of(1, 1), creator.id)).toEqual({ canEdit: true, isCreator: true, hasOtherReviews: false })
    expect(parentAccess(of(1, 0), creator.id)).toEqual({ canEdit: true, isCreator: true, hasOtherReviews: true })
    expect(parentAccess(of(2, 1), creator.id)).toEqual({ canEdit: true, isCreator: true, hasOtherReviews: true })
  })

  it('lets each signed in user edit the restaurant', () => {
    expect(parentAccess(of(1, 0), other)).toEqual({ canEdit: true, isCreator: false, hasOtherReviews: true })
    expect(parentAccess(of(0, 0), undefined)).toEqual({ canEdit: false, isCreator: false, hasOtherReviews: false })
  })

  it('lets a reviewer delete a restaurant without a creator', () => {
    expect(parentAccess(of(1, 1, null), creator.id)).toEqual({ canEdit: true, isCreator: true, hasOtherReviews: false })
    expect(parentAccess(of(1, 0, null), creator.id)).toEqual({ canEdit: true, isCreator: false, hasOtherReviews: true })
  })
})

describe('changedPatch', () => {
  const loaded = { name: 'Dal', tags: new Set(['a', 'b']) as Set<string> | null, steps: ['Boil.'], servings: 2 as number | null }

  it('keeps only the changed properties', () => {
    expect(changedPatch({ ...loaded, tags: new Set(['b', 'a']), steps: ['Boil.'] }, loaded)).toEqual({})
    expect(changedPatch({ ...loaded, servings: null, steps: ['Boil.', 'Fry.'] }, loaded)).toEqual({ servings: null, steps: ['Boil.', 'Fry.'] })
    expect(changedPatch({ ...loaded, tags: null }, loaded)).toEqual({ tags: null })
  })
})

describe('review dates', () => {
  it('reads a date as a local day', () => {
    expect(parseDay('2024-11-04')).toEqual(new Date(2024, 10, 4))
    for (const bad of ['2024-02-30', '2023-02-29', '2024-13-01', '2024-11-4', '04/11/2024', '']) {
      expect(parseDay(bad)).toBeNull()
    }
  })

  it('shows the day without a change of time zone', () => {
    expect(formatDay('2024-11-04')).toBe('4 Nov 2024')
    expect(formatDay('2024-01-01')).toBe('1 Jan 2024')
  })

  it('gives the local date of the browser', () => {
    expect(localDay(new Date(2024, 10, 4, 23, 59))).toBe('2024-11-04')
    expect(localDay(new Date(2024, 0, 9, 0, 1))).toBe('2024-01-09')
  })

  it('allows the dates from 1900 to today', () => {
    expect(checkReviewDay('2024-11-04', '2024-11-04')).toBe('')
    expect(checkReviewDay('1900-01-01', '2024-11-04')).toBe('')
    expect(checkReviewDay('2024-11-05', '2024-11-04')).toBe('Enter a date from 1 Jan 1900 to today.')
    expect(checkReviewDay('1899-12-31', '2024-11-04')).toBe('Enter a date from 1 Jan 1900 to today.')
    expect(checkReviewDay('', '2024-11-04')).toBe('Enter a date.')
  })
})

describe('own reviews', () => {
  it('tells that the delete removes each own review', () => {
    expect(ownReviewsText(1)).toBe('Your review and its photos will be deleted too.')
    expect(ownReviewsText(3)).toBe('Your 3 reviews and their photos will be deleted too.')
  })
})

describe('overallOf', () => {
  it('gives the average with one decimal after each score has a value', () => {
    expect(overallOf({ space: 7, service: 9, food: 8, price: 6 })).toBe(7.5)
    expect(overallOf({ space: 7, service: 9, food: 8, price: null })).toBeNull()
  })
})

describe('directionsLink', () => {
  const point = { latitude: 41.3766, longitude: 2.1893 }

  it('opens the map app of a phone and OpenStreetMap on other devices', () => {
    expect(directionsLink(point, 'Mozilla/5.0 (Linux; Android 16; Pixel 9)')).toBe('geo:41.3766,2.1893?q=41.3766,2.1893')
    expect(directionsLink(point, 'Mozilla/5.0 (iPhone; CPU iPhone OS 19_0 like Mac OS X)')).toBe('https://maps.apple.com/?daddr=41.3766,2.1893')
    expect(directionsLink(point, 'Mozilla/5.0 (X11; Linux x86_64)')).toBe('https://www.openstreetmap.org/directions?to=41.3766%2C2.1893')
  })
})

describe('problem codes', () => {
  it('tells the reason of a 403', async () => {
    expect(failureText(await apiFailure(rejectWith(403, { code: 'not_approved' })))).toBe(notApprovedText)
    expect(failureText(await apiFailure(rejectWith(403, { code: 'not_creator' })))).toBe(notCreatorText)
    expect(failureText(await apiFailure(rejectWith(403)), 'Not saved.')).toBe('Not saved.')
    expect(failureText(await apiFailure(rejectWith(401)))).toBe('Sign in again, then try again.')
  })

  it('has a text for each code of the list operations', async () => {
    const fallback = 'Not saved.'
    const texts: Partial<Record<NonNullable<Problem['code']>, string>> = {
      list_name_taken: 'You have a list with this name.',
      list_limit: 'You have the maximum number of lists. Delete a list first.',
      list_restaurant_limit: 'This list is full. Remove a restaurant first.',
      favorites_list: 'The favorites list cannot be renamed or deleted.',
      not_creator: 'Only the person who made the list can change it.',
      not_approved: notApprovedText,
    }
    for (const [code, text] of Object.entries(texts)) {
      const problem = { code: code as Problem['code'] }
      expect(await listFailureText(rejectWith(code.startsWith('list_') ? 409 : 403, problem), fallback)).toBe(text)
    }
    expect(await listFailureText(rejectWith(409), fallback)).toBe(fallback)
  })
})
