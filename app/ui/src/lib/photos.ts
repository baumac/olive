import type { Photo } from '@reviews/sdk'
import { type ReviewKind, kinds } from './reviewKinds'

export interface AddedPhoto {
  photo: Photo
  // altSaved is false when the photo exists but its alt text failed.
  altSaved: boolean
}

// addPhoto adds a photo to the review of a restaurant or a recipe.
export async function addPhoto(
  parentId: string,
  reviewId: string,
  file: File,
  alt: string,
  kind: ReviewKind = 'restaurant',
): Promise<AddedPhoto> {
  const api = kinds[kind]
  const photo = await api.addPhoto(parentId, reviewId, file)
  if (!alt) {
    return { photo, altSaved: true }
  }
  // A failed alt text must not look like a failed upload. A retry would
  // add the photo a second time.
  try {
    return { photo: await api.updatePhoto(parentId, reviewId, photo.id, alt), altSaved: true }
  } catch {
    return { photo, altSaved: false }
  }
}

export interface PendingPhoto {
  file: File
  alt: string
}

// pending keeps the queued photos of a new review during a sign in. A
// draft in sessionStorage cannot hold files.
const pending = new Map<string, PendingPhoto[]>()

export function keepPendingPhotos(key: string, photos: PendingPhoto[]) {
  pending.set(key, photos)
}

export function takePendingPhotos(key: string): PendingPhoto[] {
  const photos = pending.get(key) ?? []
  pending.delete(key)
  return photos
}

// photoLabel is the accessible name of a photo without alt text.
export function photoLabel(photo: Photo, index: number, count: number): string {
  return photo.alt || `Photo ${index + 1} of ${count}`
}
