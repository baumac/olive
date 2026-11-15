import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { EmptyState } from '@/components/ui/empty-state'
import { FilterBar } from '@/components/ui/filter-bar'
import { RatingDisplay } from '@/components/ui/rating-display'
import { RatingInput } from '@/components/ui/rating-input'
import { SignInForm } from '@/components/ui/sign-in-form'
import { SignUpForm } from '@/components/ui/sign-up-form'
import { TagInput } from '@/components/ui/tag-input'
import { checkPassword } from '@/lib/validation'
import { MessageSquareIcon } from 'lucide-vue-next'
import { expectAccessible } from './axe'

function attach() {
  const el = document.createElement('div')
  document.body.appendChild(el)
  return el
}

describe('RatingDisplay', () => {
  it('names the score for screen readers', async () => {
    const w = mount(RatingDisplay, { props: { value: 8, max: 10, label: 'Food' }, attachTo: attach() })
    expect(w.get('[role=img]').attributes('aria-label')).toBe('Food: 8 out of 10')
    expect(w.text()).toBe('8')
    await expectAccessible(w.element)
  })

  it('shows an average with its decimal', () => {
    const w = mount(RatingDisplay, { props: { value: 7.6, max: 10, label: 'Overall' }, attachTo: attach() })
    expect(w.get('[role=img]').attributes('aria-label')).toBe('Overall: 7.6 out of 10')
    expect(w.text()).toBe('7.6')
  })

  it('shows the label only on request', () => {
    const w = mount(RatingDisplay, { props: { value: 8, max: 10, label: 'Food', showLabel: true } })
    expect(w.text()).toBe('Food8')
  })
})

describe('RatingInput', () => {
  it('has no value at first and selects with a click', async () => {
    const w = mount(RatingInput, { props: { max: 5, label: 'Rating', modelValue: null }, attachTo: attach() })
    const radios = w.findAll('[role=radio]')
    expect(radios).toHaveLength(5)
    expect(radios.every((r) => r.attributes('aria-checked') === 'false')).toBe(true)
    await radios[3]!.trigger('click')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([4])
    await expectAccessible(w.element)
  })

  it('links its help text and the error message to the group', async () => {
    const w = mount(RatingInput, {
      props: { max: 10, label: 'Value', id: 'price', description: 'Value for money.', lowest: 'poor value', highest: 'great value', describedBy: 'price-error' },
      attachTo: attach(),
    })
    expect(w.get('[role=radiogroup]').attributes('aria-describedby')).toBe('price-description price-ends price-error')
    expect(w.get('#price-description').text()).toBe('Value for money.')
    expect(w.get('#price-ends .sr-only').text()).toBe('1 means poor value. 10 means great value.')
    expect(w.findAll('#price-ends [aria-hidden=true]').map((s) => s.text())).toEqual(['poor value', 'great value'])
    await expectAccessible(w.element)
  })

  it('has no description without help text', () => {
    const w = mount(RatingInput, { props: { max: 10, label: 'Price' } })
    expect(w.get('[role=radiogroup]').attributes('aria-describedby')).toBeUndefined()
  })
})

describe('TagInput', () => {
  it('adds a tag on Enter and removes it with its button', async () => {
    const w = mount(TagInput, { props: { modelValue: [], id: 'tags', 'aria-label': 'Tags' }, attachTo: attach() })
    const input = w.get('input')
    await input.setValue('pasta')
    await input.trigger('keydown', { key: 'Enter' })
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([['pasta']])

    await w.setProps({ modelValue: ['pasta'] })
    await w.get('button[aria-label="Remove pasta"]').trigger('click')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([[]])
  })

  it('clears the error when the person types again', async () => {
    const w = mount(TagInput, { props: { modelValue: [] } })
    const input = w.get('input')
    await input.setValue('Bad Tag')
    await input.trigger('keydown', { key: 'Enter' })
    await input.setValue('')
    expect(w.emitted('error')?.at(-1)).toEqual([''])
  })

  it('does not change an invalid tag and suggests the valid form', async () => {
    const w = mount(TagInput, { props: { modelValue: [] } })
    const input = w.get('input')
    await input.setValue('Wood Fired')
    await input.trigger('keydown', { key: 'Enter' })
    expect(w.emitted('update:modelValue')).toBeUndefined()
    expect(w.emitted('error')?.at(-1)?.[0]).toContain('wood-fired')
  })
})

describe('SignInForm', () => {
  it('checks the fields before it sends them', async () => {
    const w = mount(SignInForm, {
      props: { loading: false, error: null, notice: null, signUpPath: '/sign-up', forgotPasswordPath: '/forgot-password' },
      attachTo: attach(),
    })
    await w.get('form').trigger('submit')
    expect(w.emitted('submit')).toBeUndefined()
    expect(w.text()).toContain('Enter your email address.')
    expect(w.get('#sign-in-email').attributes('aria-invalid')).toBe('true')

    await w.get('#sign-in-email').setValue('ana@example.com')
    await w.get('#sign-in-password').setValue('correct-horse')
    await w.get('form').trigger('submit')
    expect(w.emitted('submit')?.[0]).toEqual([{ email: 'ana@example.com', password: 'correct-horse' }])
    await expectAccessible(w.element)
  })

  it('shows the session ended notice', () => {
    const w = mount(SignInForm, {
      props: { loading: false, error: null, notice: 'session-ended', signUpPath: '/sign-up', forgotPasswordPath: '/forgot-password' },
    })
    expect(w.text()).toContain('Your session has ended')
  })

  it('toggles the password visibility', async () => {
    const w = mount(SignInForm, { props: { loading: false, error: null, notice: null, signUpPath: '/sign-up', forgotPasswordPath: '/forgot-password' } })
    const toggle = w.get('button[aria-label="Show password"]')
    expect(w.get('#sign-in-password').attributes('type')).toBe('password')
    await toggle.trigger('click')
    expect(toggle.attributes('aria-pressed')).toBe('true')
    expect(w.get('#sign-in-password').attributes('type')).toBe('text')
  })
})

describe('SignUpForm', () => {
  it('shows the UI text for each rule and the API errors', async () => {
    const w = mount(SignUpForm, {
      props: { loading: false, error: null, errors: {}, signInPath: '/sign-in' },
      attachTo: attach(),
    })
    await w.get('#sign-up-password').setValue('short')
    await w.get('#sign-up-confirm').setValue('shorter')
    await w.get('form').trigger('submit')
    expect(w.text()).toContain('Enter your name.')
    expect(w.text()).toContain(checkPassword('short'))
    expect(w.text()).toContain('The passwords do not match.')
    expect(w.emitted('submit')).toBeUndefined()

    await w.setProps({ errors: { email: 'Enter an email address, for example ana@example.com.' } })
    expect(w.text()).toContain('for example ana@example.com')
    await expectAccessible(w.element)
  })
})

describe('FilterBar', () => {
  it('shows a chip for each filter and removes it', async () => {
    const w = mount(FilterBar, {
      props: {
        defs: [{ key: 'tags', label: 'Tags', type: 'tags', chip: (v: string) => `Tag: ${v}` }],
        sortOptions: [{ value: '-created', label: 'Newest' }],
        sortLabel: 'Sort by',
        status: '2 reviews',
        filters: { tags: ['italian', 'pasta'] },
        sort: '-created',
      },
      attachTo: attach(),
    })
    expect(w.get('[role=status]').text()).toBe('2 reviews')
    await w.get('button[aria-label="Remove filter tag italian"]').trigger('click')
    expect(w.emitted('update:filters')?.at(-1)).toEqual([{ tags: ['pasta'] }])
    await expectAccessible(w.element)
  })

  it('sends a search after a pause in typing', async () => {
    vi.useFakeTimers()
    try {
      const w = mount(FilterBar, {
        props: {
          defs: [{ key: 'name', label: 'Search by name', type: 'search', chip: (v: string) => `Name: ${v}` }],
          sortOptions: [{ value: '-created', label: 'Newest' }],
          sortLabel: 'Sort by',
          status: '',
          filters: {},
          sort: '-created',
        },
        attachTo: attach(),
      })
      expect(w.find('button[aria-haspopup]').exists()).toBe(false)
      const input = w.get('input[type=search]')
      await input.setValue('lu')
      await input.setValue(' lui ')
      expect(w.emitted('update:filters')).toBeUndefined()
      vi.advanceTimersByTime(300)
      expect(w.emitted('update:filters')).toEqual([[{ name: 'lui' }]])
      vi.useRealTimers()
      await expectAccessible(w.element)
    } finally {
      vi.useRealTimers()
    }
  })

  it('drops a pending search when the filters change from outside', async () => {
    vi.useFakeTimers()
    try {
      const w = mount(FilterBar, {
        props: {
          defs: [{ key: 'search', label: 'Search', type: 'search', chip: (v: string) => `Search: ${v}` }],
          sortOptions: [{ value: '-created', label: 'Newest' }],
          sortLabel: 'Sort by',
          status: '',
          filters: { search: 'old' },
          sort: '-created',
        },
        attachTo: attach(),
      })
      const input = w.get('input[type=search]')
      await input.setValue('pasta')
      await w.setProps({ filters: {} })
      vi.advanceTimersByTime(300)
      expect(w.emitted('update:filters')).toBeUndefined()
      expect((input.element as HTMLInputElement).value).toBe('')
    } finally {
      vi.useRealTimers()
    }
  })

  it('keeps typing while its own search comes back', async () => {
    vi.useFakeTimers()
    try {
      const w = mount(FilterBar, {
        props: {
          defs: [{ key: 'search', label: 'Search', type: 'search', chip: (v: string) => `Search: ${v}` }],
          sortOptions: [{ value: '-created', label: 'Newest' }],
          sortLabel: 'Sort by',
          status: '',
          filters: {},
          sort: '-created',
        },
        attachTo: attach(),
      })
      const input = w.get('input[type=search]')
      await input.setValue('pas')
      vi.advanceTimersByTime(300)
      await input.setValue('pasta')
      await w.setProps({ filters: { search: 'pas' } })
      expect((input.element as HTMLInputElement).value).toBe('pasta')
      vi.advanceTimersByTime(300)
      expect(w.emitted('update:filters')?.at(-1)).toEqual([{ search: 'pasta' }])
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('EmptyState', () => {
  it('is accessible', async () => {
    const w = mount(EmptyState, {
      props: { icon: MessageSquareIcon, title: 'No reviews yet', description: 'Be the first to review it.' },
      attachTo: attach(),
    })
    await expectAccessible(w.element)
  })
})
