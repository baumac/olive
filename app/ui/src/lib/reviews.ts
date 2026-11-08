import type { Review } from '@reviews/sdk'

export interface ScoreField {
  name: 'space' | 'service' | 'food' | 'price' | 'cleanliness'
  label: string
  max: number
}

export const scoreFields: ScoreField[] = [
  { name: 'space', label: 'Space', max: 10 },
  { name: 'service', label: 'Service', max: 10 },
  { name: 'food', label: 'Food', max: 10 },
  { name: 'price', label: 'Price', max: 10 },
  { name: 'cleanliness', label: 'Cleanliness', max: 10 },
]

export interface Score extends ScoreField {
  value: number
}

export function scoresOf(review: Review): Score[] {
  return scoreFields.map((f) => ({ ...f, value: review[f.name] }))
}

export function tagList(tags: Set<string> | undefined | null): string[] {
  return tags ? Array.from(tags) : []
}
