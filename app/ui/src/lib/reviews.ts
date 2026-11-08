import type { Review } from '@reviews/sdk'

export const kindLabels: Record<string, string> = {
  basic: 'Basic',
  joc_de_cartes: 'Joc de cartes',
}

export interface ScoreField {
  name: 'rating' | 'space' | 'service' | 'food' | 'price'
  label: string
  max: number
}

export const kindScores: Record<string, ScoreField[]> = {
  basic: [{ name: 'rating', label: 'Rating', max: 5 }],
  joc_de_cartes: [
    { name: 'space', label: 'Space', max: 10 },
    { name: 'service', label: 'Service', max: 10 },
    { name: 'food', label: 'Food', max: 10 },
    { name: 'price', label: 'Price', max: 10 },
  ],
}

// kindOf gives the kind of a review. A later release of the API can add
// kinds that the SDK types do not list.
export function kindOf(review: Review): string {
  return (review as { kind: string }).kind
}

export interface Score extends ScoreField {
  value: number
}

// scoresOf gives no scores for an unknown kind.
export function scoresOf(review: Review): Score[] {
  const values = review as unknown as Record<string, unknown>
  return (kindScores[kindOf(review)] ?? []).flatMap((f) =>
    typeof values[f.name] === 'number' ? [{ ...f, value: values[f.name] as number }] : [],
  )
}

export function tagList(tags: Set<string> | undefined | null): string[] {
  return tags ? Array.from(tags) : []
}
