import type { RecipeReview, RestaurantReview } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import EntryCard from '@/components/app/EntryCard.vue'
import { ScoreStamp } from '@/components/ui/score-stamp'
import { useEntries } from '@/composables/useEntries'
import { groupByMonth, groupByWritten, takeMerged } from '@/lib/entries'
import { expectAccessible } from './axe'

function visit(id: string, date: string, created: string): RestaurantReview {
  return {
    id,
    restaurant: 'k3x9m2p7q1w8z4a',
    restaurantName: "Luigi's Trattoria",
    date,
    overall: 7.5,
    space: 7,
    service: 9,
    food: 8,
    price: 6,
    title: `Visit ${id}`,
    creator: { id: 'u5n1c8r3t6y0b2e', name: 'Ana' },
    photos: [],
    created,
    updated: created,
  }
}

function cook(id: string, date: string, created: string): RecipeReview {
  return {
    id,
    recipe: 'c4r8e2p6i0s3t7a',
    recipeName: 'Shakshuka',
    date,
    overall: 8.25,
    taste: 9,
    ease: 8,
    instructions: 8,
    value: 8,
    title: `Cook ${id}`,
    creator: { id: 'u5n1c8r3t6y0b2e', name: 'Ana' },
    photos: [],
    created,
    updated: created,
  }
}

const ids = (items: { id: string }[]) => items.map((i) => i.id)

describe('takeMerged', () => {
  it('merges two sources by the time they were written', () => {
    const sources = [
      { buffer: [visit('a', '2026-11-01', '2026-11-22T10:00:00.000Z'), visit('b', '2026-11-01', '2026-11-20T10:00:00.000Z')], cursor: null },
      { buffer: [cook('c', '2026-11-01', '2026-11-21T10:00:00.000Z')], cursor: null },
    ]
    expect(ids(takeMerged(sources, 'created'))).toEqual(['a', 'c', 'b'])
  })

  it('stops where a source runs out but has more pages', () => {
    const sources = [
      { buffer: [visit('a', '2026-11-01', '2026-11-22T10:00:00.000Z'), visit('b', '2026-11-01', '2026-11-10T10:00:00.000Z')], cursor: null },
      { buffer: [cook('c', '2026-11-01', '2026-11-21T10:00:00.000Z')], cursor: 'next' },
    ]
    expect(ids(takeMerged(sources, 'created'))).toEqual(['a', 'c'])
    expect(ids(sources[0]!.buffer)).toEqual(['b'])
  })

  it('sorts by the date first, then by the time written for a tie of dates', () => {
    const sources = [
      { buffer: [visit('a', '2026-11-14', '2026-11-14T10:00:00.000Z')], cursor: null },
      { buffer: [cook('c', '2026-11-14', '2026-11-15T10:00:00.000Z'), cook('d', '2026-10-01', '2026-11-22T10:00:00.000Z')], cursor: null },
    ]
    expect(ids(takeMerged(sources, 'date'))).toEqual(['c', 'a', 'd'])
  })

  it('takes all entries of the other source when one source is empty', () => {
    const sources = [
      { buffer: [], cursor: null },
      { buffer: [cook('c', '2026-11-14', '2026-11-15T10:00:00.000Z')], cursor: null },
    ]
    expect(ids(takeMerged(sources, 'created'))).toEqual(['c'])
  })
})

describe('groups', () => {
  it('groups the table by the past week and the months', () => {
    const now = new Date('2026-11-23T12:00:00.000Z')
    const groups = groupByWritten(
      [
        visit('a', '2026-10-02', '2026-11-22T10:00:00.000Z'),
        visit('b', '2026-11-01', '2026-11-10T10:00:00.000Z'),
        visit('c', '2026-10-01', '2026-10-05T10:00:00.000Z'),
      ],
      now,
    )
    expect(groups.map((g) => [g.label, ids(g.entries)])).toEqual([
      ['Past week', ['a']],
      ['Earlier in November', ['b']],
      ['October 2026', ['c']],
    ])
  })

  it('groups a journal by the month of the date', () => {
    const groups = groupByMonth([visit('a', '2026-11-22', '2026-11-22T10:00:00.000Z'), cook('b', '2026-10-30', '2026-11-22T11:00:00.000Z')])
    expect(groups.map((g) => [g.label, ids(g.entries)])).toEqual([
      ['November 2026', ['a']],
      ['October 2026', ['b']],
    ])
  })
})

describe('useEntries', () => {
  it('loads the next page of the source that ran out', async () => {
    const pages: Record<string, { items: RestaurantReview[] | RecipeReview[]; nextCursor: string | null }> = {
      'visits:': { items: [visit('a', '2026-11-01', '2026-11-22T10:00:00.000Z')], nextCursor: 'v2' },
      'visits:v2': { items: [visit('b', '2026-11-01', '2026-11-12T10:00:00.000Z')], nextCursor: null },
      'cooks:': { items: [cook('c', '2026-11-01', '2026-11-21T10:00:00.000Z'), cook('d', '2026-11-01', '2026-11-11T10:00:00.000Z')], nextCursor: null },
    }
    const entries = useEntries(
      () => ['visits', 'cooks'].map((name) => async (cursor?: string) => pages[`${name}:${cursor ?? ''}`]!),
      () => 'created',
    )
    await entries.reload()
    expect(ids(entries.items.value)).toEqual(['a'])
    expect(entries.hasMore.value).toBe(true)
    await entries.loadMore()
    expect(ids(entries.items.value)).toEqual(['a', 'c', 'b', 'd'])
    expect(entries.hasMore.value).toBe(false)
  })
})

describe('EntryCard and ScoreStamp', () => {
  async function router() {
    const r = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p(.*)*', component: { render: () => null } }] })
    await r.push('/')
    return r
  }

  it('shows a review without photos as a row with the overall', async () => {
    const w = mount(EntryCard, {
      props: { review: visit('a', '2026-11-14', '2026-11-14T10:00:00.000Z') },
      global: { plugins: [await router()] },
      attachTo: document.body.appendChild(document.createElement('div')),
    })
    await flushPromises()
    expect(w.get('h3 a').text()).toBe("Luigi's Trattoria: Visit a")
    expect(w.text()).toContain('Ana · Visited on 14 Nov 2026')
    expect(w.get('[role=img]').attributes('aria-label')).toBe('Overall: 7.5 out of 10')
    await expectAccessible(w.element)
  })

  it('names the overall of the stamp', () => {
    const w = mount(ScoreStamp, { props: { value: 8.25 } })
    expect(w.attributes('aria-label')).toBe('Overall: 8.3 out of 10')
    expect(w.text()).toBe('8.3')
  })
})
