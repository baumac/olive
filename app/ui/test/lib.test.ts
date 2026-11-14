import { describe, expect, it } from 'vitest'
import spec from '../../../specs/api/public-openapi.yaml?raw'
import { fieldErrors } from '@/lib/api'
import { restaurantFilterParams } from '@/lib/restaurantFilters'
import { reviewFilterParams } from '@/lib/reviewFilters'
import { formatCount, formatDate, formatScore, initials, parseTimestamp } from '@/lib/format'
import { restaurantAccess } from '@/lib/restaurants'
import { directionsLink } from '@/lib/places'
import { overallOf, scoreFields } from '@/lib/reviews'
import { checkLongText, checkName, checkPassword, checkTag, safeRedirect, suggestTag } from '@/lib/validation'

describe('format', () => {
  const now = new Date('2024-11-10T12:00:00.000Z')

  it('reads the PocketBase timestamp format', () => {
    expect(parseTimestamp('2024-11-04T01:26:55.002Z').toISOString()).toBe('2024-11-04T01:26:55.002Z')
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
    expect(safeRedirect('//evil.example')).toBe('/restaurants')
    expect(safeRedirect('/\\evil.example')).toBe('/restaurants')
    expect(safeRedirect('https://evil.example')).toBe('/restaurants')
    expect(safeRedirect(undefined)).toBe('/restaurants')
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
      cleanliness: undefined,
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
      'space Space', 'service Service', 'food Food', 'price Value', 'cleanliness Cleanliness',
    ])
  })
})

describe('restaurantFilterParams', () => {
  it('changes the filters to SDK parameters', () => {
    expect(restaurantFilterParams({ search: 'lui', tags: ['pasta'], overall: '8' })).toEqual({
      search: 'lui',
      tagsHasAny: ['pasta'],
      overall: { gte: 8 },
    })
    expect(restaurantFilterParams({})).toEqual({ search: undefined, tagsHasAny: undefined, overall: undefined })
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

describe('restaurantAccess', () => {
  const creator = { id: 'u5n1c8r3t6y0b2e', name: 'Ana' }

  it('lets the creator change the restaurant', () => {
    expect(restaurantAccess({ creator, reviewCount: 0 }, creator.id, false)).toEqual({ canChange: true, hasOtherReviews: false })
    expect(restaurantAccess({ creator, reviewCount: 1 }, creator.id, true)).toEqual({ canChange: true, hasOtherReviews: false })
    expect(restaurantAccess({ creator, reviewCount: 1 }, creator.id, false)).toEqual({ canChange: true, hasOtherReviews: true })
    expect(restaurantAccess({ creator, reviewCount: 2 }, creator.id, true)).toEqual({ canChange: true, hasOtherReviews: true })
    expect(restaurantAccess({ creator, reviewCount: 0 }, creator.id, true)).toEqual({ canChange: true, hasOtherReviews: false })
    expect(restaurantAccess({ creator, reviewCount: 1 }, 'u2g6k0m4o8q1s5w', true).canChange).toBe(false)
    expect(restaurantAccess({ creator, reviewCount: 0 }, undefined, false).canChange).toBe(false)
  })

  it('lets a reviewer change a restaurant without a creator', () => {
    expect(restaurantAccess({ creator: null, reviewCount: 1 }, creator.id, true)).toEqual({ canChange: true, hasOtherReviews: false })
    expect(restaurantAccess({ creator: null, reviewCount: 1 }, creator.id, false)).toEqual({ canChange: false, hasOtherReviews: true })
  })

  it('leaves the decision to the API after a failed lookup', () => {
    expect(restaurantAccess({ creator, reviewCount: 1 }, creator.id, null)).toEqual({ canChange: true, hasOtherReviews: false })
    expect(restaurantAccess({ creator, reviewCount: 2 }, creator.id, null)).toEqual({ canChange: true, hasOtherReviews: true })
    expect(restaurantAccess({ creator: null, reviewCount: 1 }, creator.id, null).canChange).toBe(false)
  })
})

describe('overallOf', () => {
  it('gives the average with one decimal after each score has a value', () => {
    expect(overallOf({ space: 7, service: 9, food: 8, price: 6, cleanliness: 8 })).toBe(7.6)
    expect(overallOf({ space: 7, service: 9, food: 8, price: 6, cleanliness: null })).toBeNull()
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
