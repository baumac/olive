import type { RestaurantRating, Review } from '@reviews/sdk'

export interface ScoreField {
  name: 'space' | 'service' | 'food' | 'price' | 'cleanliness'
  label: string
  max: number
  // description is a copy of the score description in the API spec.
  description: string
  lowest: string
  highest: string
}

export const scoreFields: ScoreField[] = [
  {
    name: 'space',
    label: 'Space',
    max: 10,
    description: 'Decoration, comfort, atmosphere, and kitchen setup.',
    lowest: 'uncomfortable, dull, and badly set up',
    highest: 'comfortable, pleasant, and well set up',
  },
  {
    name: 'service',
    label: 'Service',
    max: 10,
    description: 'Professionalism, speed, friendliness, and attentiveness of the waitstaff.',
    lowest: 'slow and unfriendly',
    highest: 'fast, friendly, and attentive',
  },
  {
    name: 'food',
    label: 'Food',
    max: 10,
    description: 'Presentation, flavor, ingredient quality, and culinary execution.',
    lowest: 'badly made and bland',
    highest: 'well made and full of flavor',
  },
  {
    name: 'price',
    label: 'Price',
    max: 10,
    description: 'Value for money, and whether the final bill matches the quality provided.',
    lowest: 'poor value for money',
    highest: 'great value for money',
  },
  {
    name: 'cleanliness',
    label: 'Cleanliness',
    max: 10,
    description: 'Cleanliness of the dining room, the tableware, and the restrooms.',
    lowest: 'dirty',
    highest: 'spotless',
  },
]

export const scoreLegend = 'A higher score is always better. A high price score means good value for money.'

export interface Score extends Pick<ScoreField, 'label' | 'max'> {
  name: ScoreField['name'] | 'overall'
  value: number
}

export function scoresOf(review: Review): Score[] {
  return scoreFields.map((f) => ({ ...f, value: review[f.name] }))
}

// ratingScores puts overall first. The other averages follow in the order of
// the review scores.
export function ratingScores(rating: RestaurantRating): Score[] {
  return [{ name: 'overall', label: 'Overall', max: 10, value: rating.overall }, ...scoreFields.map((f) => ({ ...f, value: rating[f.name] }))]
}

export function tagList(tags: Set<string> | undefined | null): string[] {
  return tags ? Array.from(tags) : []
}
