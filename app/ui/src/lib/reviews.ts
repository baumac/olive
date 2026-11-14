export interface ScoreField {
  name: 'space' | 'service' | 'food' | 'price' | 'cleanliness'
  label: string
  max: number
  // description is a copy of the score description in the API spec.
  description: string
  // lowest and highest are short. They show at the ends of the score row.
  lowest: string
  highest: string
}

export const scoreFields: ScoreField[] = [
  {
    name: 'space',
    label: 'Space',
    max: 10,
    description: 'Decoration, comfort, atmosphere, and kitchen setup.',
    lowest: 'uncomfortable',
    highest: 'comfortable',
  },
  {
    name: 'service',
    label: 'Service',
    max: 10,
    description: 'Professionalism, speed, friendliness, and attentiveness of the waitstaff.',
    lowest: 'slow',
    highest: 'attentive',
  },
  {
    name: 'food',
    label: 'Food',
    max: 10,
    description: 'Presentation, flavor, ingredient quality, and culinary execution.',
    lowest: 'bland',
    highest: 'full of flavor',
  },
  {
    name: 'price',
    label: 'Value',
    max: 10,
    description: 'Value for money, and whether the final bill matches the quality provided.',
    lowest: 'poor value',
    highest: 'great value',
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

// overallOf gives the overall score of a review, like the API. It returns
// null until each score has a value.
export function overallOf(scores: Record<string, number | null | undefined>): number | null {
  let sum = 0
  for (const f of scoreFields) {
    const v = scores[f.name]
    if (v == null) {
      return null
    }
    sum += v
  }
  return Math.round((sum / scoreFields.length) * 10) / 10
}

export function tagList(tags: Set<string> | undefined | null): string[] {
  return tags ? Array.from(tags) : []
}
