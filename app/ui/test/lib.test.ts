import { describe, expect, it } from 'vitest'
import { fieldErrors } from '@/lib/api'
import { formatCount, formatDate, initials, parseTimestamp } from '@/lib/format'
import { checkLongText, checkName, checkPassword, checkTag, safeRedirect, suggestTag } from '@/lib/validation'

describe('format', () => {
  const now = new Date('2024-11-10T12:00:00.000Z')

  it('reads the PocketBase timestamp format', () => {
    expect(parseTimestamp('2024-11-04 01:26:55.002Z').toISOString()).toBe('2024-11-04T01:26:55.002Z')
  })

  it('shows relative time for the last 7 days and a date after', () => {
    expect(formatDate('2024-11-10 11:59:40.000Z', now)).toBe('just now')
    expect(formatDate('2024-11-10 11:55:00.000Z', now)).toBe('5 minutes ago')
    expect(formatDate('2024-11-10 09:00:00.000Z', now)).toBe('3 hours ago')
    expect(formatDate('2024-11-09 10:00:00.000Z', now)).toBe('yesterday')
    expect(formatDate('2024-11-06 10:00:00.000Z', now)).toBe('4 days ago')
    expect(formatDate('2024-11-01 10:00:00.000Z', now)).toBe('1 Nov 2024')
    expect(formatDate('2024-11-10 12:00:03.000Z', now)).toBe('just now')
    expect(formatDate('2024-11-20 12:00:00.000Z', now)).toBe('20 Nov 2024')
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
    expect(checkName('Family \u{1F468}\u200d\u{1F469}', 'empty')).toBe('')
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
