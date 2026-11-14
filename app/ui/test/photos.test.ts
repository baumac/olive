import type { Photo } from '@reviews/sdk'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { PhotoGrid } from '@/components/ui/photo-grid'
import { PhotoUpload } from '@/components/ui/photo-upload'
import { PhotoViewer } from '@/components/ui/photo-viewer'
import { restaurantsApi } from '@/lib/api'
import { addPhoto } from '@/lib/photos'
import { checkAlt, checkPhotoFile } from '@/lib/validation'
import { expectAccessible } from './axe'

function photo(id: string, alt: string | null): Photo {
  const base = `/v1/restaurants/k3x9m2p7q1w8z4a/reviews/r1b8c5e2g9i6k3m/photos/${id}`
  return { id, alt, width: 800, height: 600, urls: { full: `${base}/full`, thumb: `${base}/thumb` }, created: '2024-11-06T19:04:30.811Z' }
}

const photos = [photo('p4h7j1l5n9q2s6v', 'A plate of carbonara'), photo('p1a2b3c4d5e6f7g', null)]

function attach() {
  const el = document.createElement('div')
  document.body.appendChild(el)
  return el
}

function fileOf(type: string, size: number): File {
  const file = new File(['x'], 'photo', { type })
  Object.defineProperty(file, 'size', { value: size })
  return file
}

async function choose(input: HTMLInputElement, file: File) {
  Object.defineProperty(input, 'files', { value: [file], configurable: true })
  input.dispatchEvent(new Event('change'))
  await nextTick()
}

describe('photo validation', () => {
  it('accepts JPEG, PNG, and WebP of at most 10 MB', () => {
    expect(checkPhotoFile(fileOf('image/jpeg', 10 * 1024 * 1024))).toBe('')
    expect(checkPhotoFile(fileOf('image/webp', 1))).toBe('')
    expect(checkPhotoFile(fileOf('image/gif', 1))).toContain('JPEG, PNG, or WebP')
    expect(checkPhotoFile(fileOf('image/png', 11 * 1024 * 1024))).toContain('at most 10 MB')
  })

  it('allows an empty alt text but not hidden characters', () => {
    expect(checkAlt('')).toBe('')
    expect(checkAlt('Pasta')).toBe('')
    expect(checkAlt('   ')).not.toBe('')
    expect(checkAlt('Pasta​')).not.toBe('')
  })
})

describe('PhotoUpload', () => {
  beforeEach(() => {
    URL.createObjectURL = vi.fn(() => 'blob:preview')
    URL.revokeObjectURL = vi.fn()
  })

  it('refuses a GIF and a file of 11 MB before the upload', async () => {
    const w = mount(PhotoUpload, { attachTo: attach() })
    const input = w.get('input[type=file]').element as HTMLInputElement
    await choose(input, fileOf('image/gif', 100))
    expect(w.text()).toContain('Select a JPEG, PNG, or WebP image of at most 10 MB.')
    expect(input.getAttribute('aria-invalid')).toBe('true')
    expect(input.value).toBe('')
    await choose(input, fileOf('image/jpeg', 11 * 1024 * 1024))
    expect(w.find('img').exists()).toBe(false)
    expect(w.emitted('add')).toBeUndefined()
    await expectAccessible(w.element)
  })

  it('shows a preview and sends the file with its alt text', async () => {
    const w = mount(PhotoUpload, { attachTo: attach() })
    const file = fileOf('image/png', 100)
    await choose(w.get('input[type=file]').element as HTMLInputElement, file)
    expect(w.get('img').attributes('src')).toBe('blob:preview')
    await w.get('input[name=alt]').setValue('A plate of pasta')
    await w.get('button').trigger('click')
    expect(w.emitted('add')?.[0]).toEqual([file, 'A plate of pasta'])
    await expectAccessible(w.element)
  })
})

describe('PhotoGrid', () => {
  it('names each thumbnail by its alt text or its place', async () => {
    const w = mount(PhotoGrid, { props: { photos, label: 'Photos of the review' }, attachTo: attach() })
    const buttons = w.findAll('button')
    expect(buttons.map((b) => b.attributes('aria-label'))).toEqual(['A plate of carbonara', 'Photo 2 of 2'])
    expect(w.get('img').attributes()).toMatchObject({ width: '800', height: '600', loading: 'lazy' })
    await buttons[1]!.trigger('click')
    expect(w.emitted('open')?.[0]).toEqual([1])
    ;(w.vm as unknown as { focus: (i: number) => void }).focus(1)
    expect(document.activeElement).toBe(buttons[1]!.element)
    await expectAccessible(w.element)
  })
})

describe('PhotoViewer', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('moves with the arrow keys and reports the photo on close', async () => {
    const w = mount(PhotoViewer, { props: { photos, index: 0, 'onUpdate:index': (i: number | null) => w.setProps({ index: i }) }, attachTo: attach() })
    await flushPromises()
    const dialog = () => document.querySelector('[role=dialog]') as HTMLElement
    expect(dialog().textContent).toContain('Photo 1 of 2')
    dialog().dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }))
    await flushPromises()
    expect(dialog().textContent).toContain('Photo 2 of 2')
    expect(dialog().querySelector('img')?.getAttribute('alt')).toBe('Photo 2 of 2')
    await expectAccessible(dialog())
    ;(document.querySelector('[aria-label=Close]') as HTMLElement).click()
    await flushPromises()
    expect(w.emitted('close')?.[0]).toEqual([1])
  })

  it('gives the creator the edit and delete actions', async () => {
    mount(PhotoViewer, { props: { photos, index: 0, editable: true }, attachTo: attach() })
    await flushPromises()
    const text = document.querySelector('[role=dialog]')?.textContent ?? ''
    expect(text).toContain('Edit description')
    expect(text).toContain('Delete photo')
  })

  it('has no actions for other people', async () => {
    mount(PhotoViewer, { props: { photos, index: 0 }, attachTo: attach() })
    await flushPromises()
    expect(document.querySelector('[role=dialog]')?.textContent).not.toContain('Delete photo')
  })
})

describe('addPhoto', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('keeps the added photo when the alt text fails', async () => {
    vi.spyOn(restaurantsApi, 'addReviewPhoto').mockResolvedValue(photos[1]!)
    vi.spyOn(restaurantsApi, 'updateReviewPhoto').mockRejectedValue(new Error('offline'))
    const got = await addPhoto('k3x9m2p7q1w8z4a', 'r1b8c5e2g9i6k3m', fileOf('image/png', 10), 'A menu')
    expect(got).toEqual({ photo: photos[1], altSaved: false })
  })

  it('sends the type of the file', async () => {
    const add = vi.spyOn(restaurantsApi, 'addReviewPhoto').mockResolvedValue(photos[1]!)
    await addPhoto('k3x9m2p7q1w8z4a', 'r1b8c5e2g9i6k3m', fileOf('image/webp', 10), '')
    const override = add.mock.calls[0]![1] as (c: { init: RequestInit }) => Promise<RequestInit>
    const init = await override({ init: { headers: { Authorization: 'Bearer t', 'Content-Type': 'image/*' } } })
    expect(init.headers).toEqual({ Authorization: 'Bearer t', 'Content-Type': 'image/webp' })
  })
})
