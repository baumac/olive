export interface ScoreField {
  name: 'space' | 'service' | 'food' | 'price' | 'taste' | 'ease' | 'instructions' | 'value'
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
    description: 'Decoration, comfort, atmosphere, cleanliness, and kitchen setup.',
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
]

// recipeScoreFields are the scores of a recipe review. See decision 00005.
export const recipeScoreFields: ScoreField[] = [
  {
    name: 'taste',
    label: 'Taste',
    max: 10,
    description: 'How good the result tastes.',
    lowest: 'bland',
    highest: 'delicious',
  },
  {
    name: 'ease',
    label: 'Ease',
    max: 10,
    description: 'How easy the recipe is to make. A high score is easy.',
    lowest: 'hard',
    highest: 'easy',
  },
  {
    name: 'instructions',
    label: 'Instructions',
    max: 10,
    description: 'How clear the steps are, and whether the times and the amounts work.',
    lowest: 'unclear',
    highest: 'clear',
  },
  {
    name: 'value',
    label: 'Value',
    max: 10,
    description: 'Whether the result is worth the cost of the ingredients and the time.',
    lowest: 'poor value',
    highest: 'great value',
  },
]

// overallOf gives the overall score of a review with one decimal. It returns
// null until each score has a value.
export function overallOf(scores: Record<string, number | null | undefined>, fields: ScoreField[] = scoreFields): number | null {
  let sum = 0
  for (const f of fields) {
    const v = scores[f.name]
    if (v == null) {
      return null
    }
    sum += v
  }
  return Math.round((sum / fields.length) * 10) / 10
}

export function tagList(tags: Set<string> | undefined | null): string[] {
  return tags ? Array.from(tags) : []
}
